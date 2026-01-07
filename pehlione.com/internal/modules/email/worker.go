package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxSendAttempts = 8

type Worker struct {
	svc       *OutboxService
	sender    Sender
	renderer  *Renderer
	workerID  string
	batchSize int
}

func NewWorker(db *gorm.DB, sender Sender, renderer *Renderer) *Worker {
	return &Worker{
		svc:       NewService(db),
		sender:    sender,
		renderer:  renderer,
		workerID:  fmt.Sprintf("worker-%s", uuid.NewString()),
		batchSize: 10,
	}
}

// Run starts the email worker loop.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				log.Printf("Email worker %s tick error: %v", w.workerID, err)
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	batch, err := w.svc.LockBatch(ctx, w.workerID, w.batchSize)
	if err != nil {
		log.Printf("Email worker %s: lock batch error: %v", w.workerID, err)
		return err
	}
	if len(batch) == 0 {
		// Silently return - no messages to process
		return nil
	}

	log.Printf("Email worker %s: processing %d messages", w.workerID, len(batch))
	for _, job := range batch {
		w.processOne(ctx, job)
	}
	return nil
}

func (w *Worker) processOne(ctx context.Context, job OutboxEmail) {
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		log.Printf("Email worker %s: invalid payload for %d: %v", w.workerID, job.ID, err)
		if markErr := w.svc.MarkFailed(ctx, job.ID, truncateStr(err.Error(), 255)); markErr != nil {
			log.Printf("Email worker %s: mark failed error for %d: %v", w.workerID, job.ID, markErr)
		}
		return
	}

	subject := "pehlione notification"
	htmlBody := ""
	textBody := ""
	if w.renderer != nil {
		result, err := w.renderer.Render(job.Template, payload)
		if err != nil {
			log.Printf("Email worker %s: render failed for %d: %v", w.workerID, job.ID, err)
			if markErr := w.svc.MarkRetry(ctx, job.ID, truncateStr(err.Error(), 255), backoff(job.AttemptCount+1)); markErr != nil {
				log.Printf("Email worker %s: mark retry error for %d: %v", w.workerID, job.ID, markErr)
			}
			return
		}
		subject = result.Subject
		htmlBody = string(result.HTML)
		textBody = string(result.Text)
	} else {
		if subj, ok := payload["subject"].(string); ok && subj != "" {
			subject = subj
		}
		if html, ok := payload["html"].(string); ok {
			htmlBody = html
		}
		if text, ok := payload["text"].(string); ok {
			textBody = text
		}
	}

	attemptNum := job.AttemptCount + 1
	log.Printf("Email worker %s: sending %d to %s (attempt %d)", w.workerID, job.ID, job.ToEmail, attemptNum)
	sendErr := w.sender.Send(ctx, Message{
		To:      job.ToEmail,
		Subject: subject,
		Text:    textBody,
		HTML:    htmlBody,
	})

	if sendErr == nil {
		log.Printf("Email worker %s: successfully sent %d to %s", w.workerID, job.ID, job.ToEmail)
		if err := w.svc.MarkSent(ctx, job.ID); err != nil {
			log.Printf("Email worker %s: mark sent failed for %d: %v", w.workerID, job.ID, err)
		}
		return
	}

	log.Printf("Email worker %s: send failed for %d to %s: %v (attempt %d/%d)", w.workerID, job.ID, job.ToEmail, sendErr, attemptNum, maxSendAttempts)
	errMsg := truncateStr(sendErr.Error(), 255)
	if attemptNum >= maxSendAttempts {
		if err := w.svc.MarkFailed(ctx, job.ID, errMsg); err != nil {
			log.Printf("Email worker %s: mark failed error for %d: %v", w.workerID, job.ID, err)
		}
		return
	}

	delay := backoff(attemptNum)
	if err := w.svc.MarkRetry(ctx, job.ID, errMsg, delay); err != nil {
		log.Printf("Email worker %s: mark retry error for %d: %v", w.workerID, job.ID, err)
	}
}

func backoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 1 * time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 15 * time.Minute
	default:
		return 30 * time.Minute
	}
}

func truncateStr(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

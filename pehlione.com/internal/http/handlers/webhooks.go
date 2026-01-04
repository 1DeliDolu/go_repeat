package handlers

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/modules/payments"
)

type WebhookHandler struct {
	Logger     *slog.Logger
	Provider   payments.Provider
	WebhookSvc *payments.WebhookService
}

func NewWebhookHandler(logger *slog.Logger, p payments.Provider, svc *payments.WebhookService) *WebhookHandler {
	return &WebhookHandler{Logger: logger, Provider: p, WebhookSvc: svc}
}

// POST /webhooks/:provider
// Body is raw JSON; signature header validated by provider adapter.
func (h *WebhookHandler) Handle(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid body"})
		return
	}

	ev, err := h.Provider.VerifyAndParseWebhook(c.Request.Header, body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid signature or payload"})
		return
	}

	if err := h.WebhookSvc.Handle(c.Request.Context(), h.Provider.Name(), ev, body); err != nil {
		// 500 => provider retry etsin
		h.Logger.Error("webhook apply failed", "event_id", ev.EventID, "type", ev.Type, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

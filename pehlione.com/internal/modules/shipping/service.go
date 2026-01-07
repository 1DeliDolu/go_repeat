package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pehlione.com/app/internal/emails"
	emailmod "pehlione.com/app/internal/modules/email"
	"pehlione.com/app/internal/modules/orders"
)

var (
	ErrCarrierRequired      = errors.New("carrier is required")
	ErrTrackingRequired     = errors.New("tracking number is required")
	ErrOrderNotShippable    = errors.New("order is not in a shippable state")
	ErrProviderUnavailable  = errors.New("shipping provider unavailable")
	ErrActorRequired        = errors.New("actor user id required")
	errRecipientUnavailable = errors.New("shipping address missing")
)

type Service struct {
	db       *gorm.DB
	provider Provider
	emailSvc *emailmod.OutboxService
	baseURL  string
}

func NewService(db *gorm.DB, provider Provider, emailSvc *emailmod.OutboxService, baseURL string) *Service {
	return &Service{db: db, provider: provider, emailSvc: emailSvc, baseURL: baseURL}
}

type QueueShipmentInput struct {
	OrderID     string
	ActorUserID string
	Carrier     string
	Service     string
	Note        string
}

type ManualShipmentInput struct {
	OrderID     string
	ActorUserID string
	Carrier     string
	TrackingNo  string
	TrackingURL string
	Note        string
}

type shipmentJobPayload struct {
	ShipmentID  string  `json:"shipment_id"`
	OrderID     string  `json:"order_id"`
	Carrier     string  `json:"carrier"`
	Service     string  `json:"service"`
	ActorUserID string  `json:"actor_user_id"`
	Note        string  `json:"note"`
	Recipient   Address `json:"recipient"`
	Items       []Item  `json:"items"`
	Email       string  `json:"email"`
	Status      string  `json:"-"` // ignored
}

func (s *Service) QueueShipment(ctx context.Context, in QueueShipmentInput) (Shipment, error) {
	carrier := strings.TrimSpace(in.Carrier)
	if carrier == "" {
		return Shipment{}, ErrCarrierRequired
	}
	actor := strings.TrimSpace(in.ActorUserID)
	if actor == "" {
		return Shipment{}, ErrActorRequired
	}
	if s.provider == nil {
		return Shipment{}, ErrProviderUnavailable
	}
	service := strings.TrimSpace(in.Service)
	note := strings.TrimSpace(in.Note)

	var shipment Shipment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ord orders.Order
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ord, "id = ?", in.OrderID).Error; err != nil {
			return err
		}
		if !isShippableStatus(ord.Status) {
			return ErrOrderNotShippable
		}

		addr, err := parseAddress(ord.ShippingAddressJSON)
		if err != nil {
			return errRecipientUnavailable
		}

		emailAddr, err := s.lookupOrderEmail(ctx, tx, ord)
		if err != nil {
			return err
		}

		var orderItems []orders.OrderItem
		if err := tx.WithContext(ctx).
			Order("created_at ASC").
			Find(&orderItems, "order_id = ?", ord.ID).Error; err != nil {
			return err
		}

		items := make([]Item, 0, len(orderItems))
		for _, it := range orderItems {
			items = append(items, Item{
				Name:  it.ProductName,
				SKU:   it.SKU,
				Qty:   it.Quantity,
				Price: it.LineTotalCents,
			})
		}

		now := time.Now()
		shipmentID := uuid.NewString()
		shipment = Shipment{
			ID:        shipmentID,
			OrderID:   ord.ID,
			Provider:  providerName(s.provider),
			Carrier:   carrier,
			Status:    StatusPending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if service != "" {
			shipment.Service = &service
		}
		if note != "" {
			shipment.Note = &note
		}

		if err := tx.WithContext(ctx).Create(&shipment).Error; err != nil {
			return err
		}

		payload := shipmentJobPayload{
			ShipmentID:  shipmentID,
			OrderID:     ord.ID,
			Carrier:     carrier,
			Service:     service,
			ActorUserID: actor,
			Note:        note,
			Recipient:   addr,
			Items:       items,
			Email:       emailAddr,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		job := ShipmentJob{
			ShipmentID:   shipmentID,
			Status:       JobPending,
			Payload:      datatypes.JSON(data),
			AttemptCount: 0,
			ScheduledAt:  now.Add(20 * time.Second),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.WithContext(ctx).Create(&job).Error; err != nil {
			return err
		}

		noteVal := fmt.Sprintf("shipment queued carrier=%s", carrier)
		event := orders.OrderEvent{
			ID:          uuid.NewString(),
			OrderID:     ord.ID,
			ActorUserID: actor,
			Action:      "shipment_queue",
			FromStatus:  ord.Status,
			ToStatus:    ord.Status,
			Note:        &noteVal,
			CreatedAt:   now,
		}
		return tx.WithContext(ctx).Create(&event).Error
	})

	return shipment, err
}

func (s *Service) CreateManualShipment(ctx context.Context, in ManualShipmentInput) (Shipment, error) {
	carrier := strings.TrimSpace(in.Carrier)
	if carrier == "" {
		return Shipment{}, ErrCarrierRequired
	}
	actor := strings.TrimSpace(in.ActorUserID)
	if actor == "" {
		return Shipment{}, ErrActorRequired
	}
	tracking := strings.TrimSpace(in.TrackingNo)
	if tracking == "" {
		return Shipment{}, ErrTrackingRequired
	}
	trackingURL := strings.TrimSpace(in.TrackingURL)
	note := strings.TrimSpace(in.Note)

	var shipment Shipment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ord orders.Order
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ord, "id = ?", in.OrderID).Error; err != nil {
			return err
		}
		if !isShippableStatus(ord.Status) {
			return ErrOrderNotShippable
		}

		now := time.Now()
		shipmentID := uuid.NewString()
		shipment = Shipment{
			ID:        shipmentID,
			OrderID:   ord.ID,
			Provider:  "manual",
			Carrier:   carrier,
			Status:    StatusShipped,
			CreatedAt: now,
			UpdatedAt: now,
			ShippedAt: &now,
		}
		if note != "" {
			shipment.Note = &note
		}
		shipment.TrackingNumber = &tracking
		if trackingURL != "" {
			shipment.TrackingURL = &trackingURL
		}

		if err := tx.WithContext(ctx).Create(&shipment).Error; err != nil {
			return err
		}

		fromStatus := ord.Status
		if err := s.promoteOrderStatus(ctx, tx, &ord); err != nil {
			return err
		}

		eventNote := fmt.Sprintf("manual shipment tracking=%s", tracking)
		event := orders.OrderEvent{
			ID:          uuid.NewString(),
			OrderID:     ord.ID,
			ActorUserID: actor,
			Action:      "ship_manual",
			FromStatus:  fromStatus,
			ToStatus:    ord.Status,
			Note:        &eventNote,
			CreatedAt:   now,
		}
		if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
			return err
		}

		var orderItems []orders.OrderItem
		orderItems, _ = s.loadOrderItems(ctx, tx, ord.ID)

		if s.emailSvc != nil {
			if emailAddr, err := s.lookupOrderEmail(ctx, tx, ord); err == nil && emailAddr != "" {
				payload := emails.BuildOrderPayload(s.baseURL, ord, orderItems, "Shipped", "")
				if trackingURL != "" {
					payload["TrackingURL"] = trackingURL
				}
				payload["ShipmentCarrier"] = carrier
				payload["TrackingNumber"] = tracking
				payload["PreviewText"] = "Your order is on the way."
				_ = s.emailSvc.EnqueueTx(ctx, tx, emailmod.Job{
					To:       emailAddr,
					Template: emailmod.TemplateOrderShipped,
					Payload:  payload,
				})
			}
		}
		return nil
	})

	return shipment, err
}

func (s *Service) LockJobBatch(ctx context.Context, workerID string, limit int) ([]ShipmentJob, error) {
	if limit <= 0 {
		limit = 10
	}

	var jobs []ShipmentJob
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND scheduled_at <= ?", JobPending, time.Now()).
			Order("scheduled_at ASC").
			Limit(limit).
			Find(&jobs).Error; err != nil {
			return err
		}
		if len(jobs) == 0 {
			return nil
		}

		now := time.Now()
		ids := make([]int64, len(jobs))
		for i, job := range jobs {
			ids[i] = job.ID
		}

		if err := tx.Model(&ShipmentJob{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":     JobProcessing,
				"locked_at":  now,
				"locked_by":  workerID,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		return tx.Where("id IN ?", ids).Find(&jobs).Error
	})
	return jobs, err
}

func (s *Service) ProcessJob(ctx context.Context, job ShipmentJob) error {
	if s.provider == nil {
		return s.markJobFailed(ctx, job, ErrProviderUnavailable)
	}

	var payload shipmentJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return s.markJobFailed(ctx, job, err)
	}

	req := LabelRequest{
		ShipmentID: job.ShipmentID,
		OrderID:    payload.OrderID,
		Carrier:    payload.Carrier,
		Service:    payload.Service,
		Reference:  fmt.Sprintf("order:%s", payload.OrderID),
		Recipient:  payload.Recipient,
		Items:      payload.Items,
		Note:       payload.Note,
	}

	resp, err := s.provider.CreateLabel(ctx, req)
	if err != nil {
		return s.retryJob(ctx, job, err)
	}

	return s.completeShipment(ctx, job, payload, resp)
}

func (s *Service) completeShipment(ctx context.Context, job ShipmentJob, payload shipmentJobPayload, resp LabelResponse) error {
	now := time.Now()
	tracking := resp.TrackingNumber
	trackingURL := resp.TrackingURL
	labelURL := resp.LabelURL

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		update := map[string]any{
			"status":        StatusShipped,
			"tracking_no":   tracking,
			"tracking_url":  nullable(trackingURL),
			"label_url":     nullable(labelURL),
			"error_message": nil,
			"shipped_at":    now,
			"updated_at":    now,
		}
		if err := tx.WithContext(ctx).Model(&Shipment{}).
			Where("id = ?", job.ShipmentID).
			Updates(update).Error; err != nil {
			return err
		}

		var ord orders.Order
		if err := tx.WithContext(ctx).First(&ord, "id = ?", payload.OrderID).Error; err != nil {
			return err
		}

		fromStatus := ord.Status
		if err := s.promoteOrderStatus(ctx, tx, &ord); err != nil {
			return err
		}

		eventNote := fmt.Sprintf("shipment_id=%s tracking=%s", job.ShipmentID, tracking)
		event := orders.OrderEvent{
			ID:          uuid.NewString(),
			OrderID:     ord.ID,
			ActorUserID: payload.ActorUserID,
			Action:      "ship",
			FromStatus:  fromStatus,
			ToStatus:    ord.Status,
			Note:        &eventNote,
			CreatedAt:   now,
		}
		if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
			return err
		}

		var orderItems []orders.OrderItem
		orderItems, _ = s.loadOrderItems(ctx, tx, ord.ID)

		if err := tx.WithContext(ctx).Model(&ShipmentJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]any{
				"status":     JobSucceeded,
				"last_error": nil,
				"locked_at":  nil,
				"locked_by":  nil,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		if s.emailSvc != nil && payload.Email != "" {
			payloadMap := emails.BuildOrderPayload(s.baseURL, ord, orderItems, "Shipped", "")
			if trackingURL != "" {
				payloadMap["TrackingURL"] = trackingURL
			}
			payloadMap["TrackingNumber"] = tracking
			payloadMap["ShipmentCarrier"] = payload.Carrier
			payloadMap["PreviewText"] = "Your order is on the way."
			if err := s.emailSvc.EnqueueTx(ctx, tx, emailmod.Job{
				To:       payload.Email,
				Template: emailmod.TemplateOrderShipped,
				Payload:  payloadMap,
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) retryJob(ctx context.Context, job ShipmentJob, cause error) error {
	errMsg := truncateErr(cause)
	attempt := job.AttemptCount + 1
	if attempt >= 8 {
		return s.markJobFailed(ctx, job, cause)
	}

	next := time.Now().Add(backoff(attempt))
	return s.db.WithContext(ctx).Model(&ShipmentJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]any{
			"status":        JobPending,
			"attempt_count": attempt,
			"last_error":    errMsg,
			"scheduled_at":  next,
			"locked_at":     nil,
			"locked_by":     nil,
			"updated_at":    time.Now(),
		}).Error
}

func (s *Service) markJobFailed(ctx context.Context, job ShipmentJob, cause error) error {
	errMsg := truncateErr(cause)
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ShipmentJob{}).
			Where("id = ?", job.ID).
			Updates(map[string]any{
				"status":        JobFailed,
				"last_error":    errMsg,
				"locked_at":     nil,
				"locked_by":     nil,
				"updated_at":    now,
				"attempt_count": job.AttemptCount + 1,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&Shipment{}).
			Where("id = ?", job.ShipmentID).
			Updates(map[string]any{
				"status":        StatusFailed,
				"error_message": errMsg,
				"updated_at":    now,
			}).Error
	})
}

func (s *Service) promoteOrderStatus(ctx context.Context, tx *gorm.DB, ord *orders.Order) error {
	if ord.Status == "shipped" {
		return nil
	}
	if ord.Status != "paid" {
		return ErrOrderNotShippable
	}

	now := time.Now()
	if err := tx.WithContext(ctx).Model(&orders.Order{}).
		Where("id = ? AND status = 'paid'", ord.ID).
		Updates(map[string]any{
			"status":     "shipped",
			"updated_at": now,
		}).Error; err != nil {
		return err
	}
	ord.Status = "shipped"
	ord.UpdatedAt = now
	return nil
}

func (s *Service) lookupOrderEmail(ctx context.Context, tx *gorm.DB, ord orders.Order) (string, error) {
	if ord.GuestEmail != nil && strings.TrimSpace(*ord.GuestEmail) != "" {
		return strings.TrimSpace(*ord.GuestEmail), nil
	}
	if ord.UserID != nil && strings.TrimSpace(*ord.UserID) != "" {
		var email string
		if err := tx.WithContext(ctx).
			Table("users").
			Select("email").
			Where("id = ?", *ord.UserID).
			Take(&email).Error; err == nil {
			return strings.TrimSpace(email), nil
		}
	}
	return "", nil
}

func (s *Service) loadOrderItems(ctx context.Context, tx *gorm.DB, orderID string) ([]orders.OrderItem, error) {
	var items []orders.OrderItem
	err := tx.WithContext(ctx).Order("created_at ASC").Find(&items, "order_id = ?", orderID).Error
	return items, err
}

func parseAddress(data []byte) (Address, error) {
	var addr Address
	if len(data) == 0 {
		return Address{}, errors.New("missing address")
	}
	if err := json.Unmarshal(data, &addr); err != nil {
		return Address{}, err
	}
	if strings.TrimSpace(addr.Address1) == "" || strings.TrimSpace(addr.City) == "" {
		return Address{}, errors.New("invalid address")
	}
	return addr, nil
}

func isShippableStatus(status string) bool {
	switch status {
	case "paid", "shipped":
		return true
	default:
		return false
	}
}

func nullable(val string) any {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	return val
}

func backoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 15 * time.Minute
	default:
		return 30 * time.Minute
	}
}

func truncateErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 255 {
		return msg[:255]
	}
	return msg
}

func providerName(p Provider) string {
	if p == nil {
		return "manual"
	}
	return p.Name()
}

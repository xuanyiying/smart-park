package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypePaymentSuccess NotificationType = "payment_success"
	NotificationTypeVehicleEntry   NotificationType = "vehicle_entry"
	NotificationTypeVehicleExit    NotificationType = "vehicle_exit"
	NotificationTypeMonthlyExpiry  NotificationType = "monthly_expiry"
	NotificationTypeSystemAlert    NotificationType = "system_alert"
)

type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
)

type Notification struct {
	ID        uuid.UUID
	Type      NotificationType
	Title     string
	Content   string
	Recipient string
	Status    NotificationStatus
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationRepo interface {
	Create(ctx context.Context, notification *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	ListByRecipient(ctx context.Context, recipientID string, page, pageSize int) ([]*Notification, int64, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, recipientID string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Notifier interface {
	Send(ctx context.Context, notification *Notification) error
}

type NotificationUseCase struct {
	repo     NotificationRepo
	notifier Notifier
	log      *log.Helper
}

func NewNotificationUseCase(repo NotificationRepo, notifier Notifier, logger log.Logger) *NotificationUseCase {
	return &NotificationUseCase{
		repo:     repo,
		notifier: notifier,
		log:      log.NewHelper(logger),
	}
}

func (uc *NotificationUseCase) SendNotification(ctx context.Context, notification *Notification) (*Notification, error) {
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	if notification.Status == "" {
		notification.Status = NotificationStatusUnread
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	if err := uc.notifier.Send(ctx, notification); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to send notification: %v", err)
	}

	if err := uc.repo.Create(ctx, notification); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to save notification: %v", err)
		return nil, fmt.Errorf("failed to save notification: %w", err)
	}

	return notification, nil
}

func (uc *NotificationUseCase) GetNotification(ctx context.Context, id uuid.UUID) (*Notification, error) {
	notification, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	return notification, nil
}

func (uc *NotificationUseCase) ListNotifications(ctx context.Context, recipientID string, page, pageSize int) ([]*Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.ListByRecipient(ctx, recipientID, page, pageSize)
}

func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return uc.repo.MarkAsRead(ctx, id)
}

func (uc *NotificationUseCase) MarkAllAsRead(ctx context.Context, recipientID string) error {
	return uc.repo.MarkAllAsRead(ctx, recipientID)
}

func (uc *NotificationUseCase) SendPaymentSuccess(ctx context.Context, orderID, amount, method string) (*Notification, error) {
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment Success",
		Content:   fmt.Sprintf("Order %s: amount %s paid via %s", orderID, amount, method),
		Recipient: orderID,
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	return uc.SendNotification(ctx, notification)
}

func (uc *NotificationUseCase) SendVehicleEntry(ctx context.Context, plateNumber, lotName string) (*Notification, error) {
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeVehicleEntry,
		Title:     "Vehicle Entry",
		Content:   fmt.Sprintf("Vehicle %s entered %s", plateNumber, lotName),
		Recipient: plateNumber,
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	return uc.SendNotification(ctx, notification)
}

func (uc *NotificationUseCase) SendVehicleExit(ctx context.Context, plateNumber, lotName, duration string, fee float64) (*Notification, error) {
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeVehicleExit,
		Title:     "Vehicle Exit",
		Content:   fmt.Sprintf("Vehicle %s exited %s, duration: %s, fee: %.2f", plateNumber, lotName, duration, fee),
		Recipient: plateNumber,
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	return uc.SendNotification(ctx, notification)
}

func (uc *NotificationUseCase) SendMonthlyExpiry(ctx context.Context, plateNumber, expiryDate string) (*Notification, error) {
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeMonthlyExpiry,
		Title:     "Monthly Pass Expiring",
		Content:   fmt.Sprintf("Vehicle %s monthly pass expires on %s", plateNumber, expiryDate),
		Recipient: plateNumber,
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	return uc.SendNotification(ctx, notification)
}

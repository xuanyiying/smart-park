package biz

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type mockNotificationRepo struct {
	mu            sync.RWMutex
	notifications map[uuid.UUID]*Notification
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		notifications: make(map[uuid.UUID]*Notification),
	}
}

func (m *mockNotificationRepo) Create(ctx context.Context, notification *Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications[notification.ID] = notification
	return nil
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.notifications[id]
	if !ok {
		return nil, fmt.Errorf("notification not found")
	}
	return n, nil
}

func (m *mockNotificationRepo) ListByRecipient(ctx context.Context, recipientID string, page, pageSize int) ([]*Notification, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Notification
	for _, n := range m.notifications {
		if n.Recipient == recipientID {
			result = append(result, n)
		}
	}
	total := int64(len(result))
	offset := (page - 1) * pageSize
	if offset >= len(result) {
		return []*Notification{}, total, nil
	}
	end := offset + pageSize
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], total, nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notifications[id]
	if !ok {
		return fmt.Errorf("notification not found")
	}
	n.Status = NotificationStatusRead
	now := time.Now()
	n.ReadAt = &now
	return nil
}

func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, recipientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, n := range m.notifications {
		if n.Recipient == recipientID && n.Status == NotificationStatusUnread {
			n.Status = NotificationStatusRead
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockNotificationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.notifications, id)
	return nil
}

type mockNotifier struct {
	mu   sync.Mutex
	sent []*Notification
}

func newMockNotifier() *mockNotifier {
	return &mockNotifier{
		sent: make([]*Notification, 0),
	}
}

func (n *mockNotifier) Send(ctx context.Context, notification *Notification) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, notification)
	return nil
}

func (n *mockNotifier) getSent() []*Notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.sent
}

type failingNotifier struct{}

func (n *failingNotifier) Send(ctx context.Context, notification *Notification) error {
	return fmt.Errorf("notification channel unavailable")
}

func setupNotificationUseCase() (*NotificationUseCase, *mockNotificationRepo, *mockNotifier) {
	repo := newMockNotificationRepo()
	notifier := newMockNotifier()
	logger := log.NewStdLogger(os.Stdout)
	uc := NewNotificationUseCase(repo, notifier, logger)
	return uc, repo, notifier
}

func TestNotificationUseCase_SendNotification(t *testing.T) {
	uc, repo, notifier := setupNotificationUseCase()
	ctx := context.Background()

	notification := &Notification{
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment Success",
		Content:   "Order 123 paid 10.00 via wechat",
		Recipient: "user-001",
	}

	result, err := uc.SendNotification(ctx, notification)
	if err != nil {
		t.Fatalf("SendNotification() error: %v", err)
	}

	if result.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if result.Status != NotificationStatusUnread {
		t.Errorf("Status = %s, want unread", result.Status)
	}
	if result.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	saved, err := repo.GetByID(ctx, result.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if saved.Content != notification.Content {
		t.Errorf("Content = %s, want %s", saved.Content, notification.Content)
	}

	sent := notifier.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}
	if sent[0].Content != notification.Content {
		t.Errorf("sent Content = %s, want %s", sent[0].Content, notification.Content)
	}
}

func TestNotificationUseCase_SendNotification_WithExistingID(t *testing.T) {
	uc, _, _ := setupNotificationUseCase()
	ctx := context.Background()

	existingID := uuid.New()
	notification := &Notification{
		ID:        existingID,
		Type:      NotificationTypeVehicleEntry,
		Title:     "Entry",
		Content:   "Car entered",
		Recipient: "user-002",
	}

	result, err := uc.SendNotification(ctx, notification)
	if err != nil {
		t.Fatalf("SendNotification() error: %v", err)
	}
	if result.ID != existingID {
		t.Errorf("ID = %s, want %s", result.ID, existingID)
	}
}

func TestNotificationUseCase_GetNotification(t *testing.T) {
	uc, repo, _ := setupNotificationUseCase()
	ctx := context.Background()

	notification := &Notification{
		Type:      NotificationTypeSystemAlert,
		Title:     "Alert",
		Content:   "System alert",
		Recipient: "admin",
	}
	repo.notifications[notification.ID] = notification

	result, err := uc.GetNotification(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetNotification() error: %v", err)
	}
	if result.Content != notification.Content {
		t.Errorf("Content = %s, want %s", result.Content, notification.Content)
	}
}

func TestNotificationUseCase_GetNotification_NotFound(t *testing.T) {
	uc, _, _ := setupNotificationUseCase()
	ctx := context.Background()

	_, err := uc.GetNotification(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent notification")
	}
}

func TestNotificationUseCase_ListNotifications(t *testing.T) {
	uc, repo, _ := setupNotificationUseCase()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		n := &Notification{
			ID:        uuid.New(),
			Type:      NotificationTypePaymentSuccess,
			Title:     "Payment",
			Content:   fmt.Sprintf("Payment %d", i),
			Recipient: "user-001",
			Status:    NotificationStatusUnread,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		repo.notifications[n.ID] = n
	}

	nOther := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeVehicleEntry,
		Title:     "Entry",
		Content:   "Other user entry",
		Recipient: "user-002",
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	repo.notifications[nOther.ID] = nOther

	notifications, total, err := uc.ListNotifications(ctx, "user-001", 1, 10)
	if err != nil {
		t.Fatalf("ListNotifications() error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(notifications) != 5 {
		t.Errorf("len(notifications) = %d, want 5", len(notifications))
	}
}

func TestNotificationUseCase_ListNotifications_Pagination(t *testing.T) {
	uc, repo, _ := setupNotificationUseCase()
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		n := &Notification{
			ID:        uuid.New(),
			Type:      NotificationTypePaymentSuccess,
			Title:     "Payment",
			Content:   fmt.Sprintf("Payment %d", i),
			Recipient: "user-001",
			Status:    NotificationStatusUnread,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		}
		repo.notifications[n.ID] = n
	}

	notifications, total, err := uc.ListNotifications(ctx, "user-001", 2, 5)
	if err != nil {
		t.Fatalf("ListNotifications() error: %v", err)
	}
	if total != 15 {
		t.Errorf("total = %d, want 15", total)
	}
	if len(notifications) != 5 {
		t.Errorf("len(notifications) = %d, want 5", len(notifications))
	}
}

func TestNotificationUseCase_ListNotifications_DefaultPagination(t *testing.T) {
	uc, _, _ := setupNotificationUseCase()
	ctx := context.Background()

	notifications, total, err := uc.ListNotifications(ctx, "user-001", 0, 0)
	if err != nil {
		t.Fatalf("ListNotifications() error: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(notifications) != 0 {
		t.Errorf("len(notifications) = %d, want 0", len(notifications))
	}
}

func TestNotificationUseCase_MarkAsRead(t *testing.T) {
	uc, repo, _ := setupNotificationUseCase()
	ctx := context.Background()

	n := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	repo.notifications[n.ID] = n

	if err := uc.MarkAsRead(ctx, n.ID); err != nil {
		t.Fatalf("MarkAsRead() error: %v", err)
	}

	saved, _ := repo.GetByID(ctx, n.ID)
	if saved.Status != NotificationStatusRead {
		t.Errorf("Status = %s, want read", saved.Status)
	}
	if saved.ReadAt == nil {
		t.Error("expected non-nil ReadAt")
	}
}

func TestNotificationUseCase_MarkAsRead_NotFound(t *testing.T) {
	uc, _, _ := setupNotificationUseCase()
	ctx := context.Background()

	err := uc.MarkAsRead(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent notification")
	}
}

func TestNotificationUseCase_MarkAllAsRead(t *testing.T) {
	uc, repo, _ := setupNotificationUseCase()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		n := &Notification{
			ID:        uuid.New(),
			Type:      NotificationTypePaymentSuccess,
			Title:     "Payment",
			Content:   fmt.Sprintf("Payment %d", i),
			Recipient: "user-001",
			Status:    NotificationStatusUnread,
			CreatedAt: time.Now(),
		}
		repo.notifications[n.ID] = n
	}

	nOther := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeVehicleEntry,
		Title:     "Entry",
		Content:   "Other user entry",
		Recipient: "user-002",
		Status:    NotificationStatusUnread,
		CreatedAt: time.Now(),
	}
	repo.notifications[nOther.ID] = nOther

	if err := uc.MarkAllAsRead(ctx, "user-001"); err != nil {
		t.Fatalf("MarkAllAsRead() error: %v", err)
	}

	for _, n := range repo.notifications {
		if n.Recipient == "user-001" {
			if n.Status != NotificationStatusRead {
				t.Errorf("Status for user-001 = %s, want read", n.Status)
			}
		}
	}

	if nOther.Status != NotificationStatusUnread {
		t.Errorf("Status for user-002 = %s, want unread", nOther.Status)
	}
}

func TestNotificationUseCase_SendPaymentSuccess(t *testing.T) {
	uc, repo, notifier := setupNotificationUseCase()
	ctx := context.Background()

	result, err := uc.SendPaymentSuccess(ctx, "order-123", "50.00", "wechat")
	if err != nil {
		t.Fatalf("SendPaymentSuccess() error: %v", err)
	}

	if result.Type != NotificationTypePaymentSuccess {
		t.Errorf("Type = %s, want payment_success", result.Type)
	}
	if result.Recipient != "order-123" {
		t.Errorf("Recipient = %s, want order-123", result.Recipient)
	}

	saved, _ := repo.GetByID(ctx, result.ID)
	if saved == nil {
		t.Error("expected notification to be saved")
	}

	sent := notifier.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}
}

func TestNotificationUseCase_SendVehicleEntry(t *testing.T) {
	uc, _, notifier := setupNotificationUseCase()
	ctx := context.Background()

	result, err := uc.SendVehicleEntry(ctx, "京A12345", "Lot A")
	if err != nil {
		t.Fatalf("SendVehicleEntry() error: %v", err)
	}

	if result.Type != NotificationTypeVehicleEntry {
		t.Errorf("Type = %s, want vehicle_entry", result.Type)
	}
	if result.Recipient != "京A12345" {
		t.Errorf("Recipient = %s, want 京A12345", result.Recipient)
	}

	sent := notifier.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}
}

func TestNotificationUseCase_SendVehicleExit(t *testing.T) {
	uc, _, notifier := setupNotificationUseCase()
	ctx := context.Background()

	result, err := uc.SendVehicleExit(ctx, "京A12345", "Lot A", "2h30m", 25.00)
	if err != nil {
		t.Fatalf("SendVehicleExit() error: %v", err)
	}

	if result.Type != NotificationTypeVehicleExit {
		t.Errorf("Type = %s, want vehicle_exit", result.Type)
	}

	sent := notifier.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}
}

func TestNotificationUseCase_SendMonthlyExpiry(t *testing.T) {
	uc, _, notifier := setupNotificationUseCase()
	ctx := context.Background()

	result, err := uc.SendMonthlyExpiry(ctx, "京A12345", "2026-06-01")
	if err != nil {
		t.Fatalf("SendMonthlyExpiry() error: %v", err)
	}

	if result.Type != NotificationTypeMonthlyExpiry {
		t.Errorf("Type = %s, want monthly_expiry", result.Type)
	}

	sent := notifier.getSent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent notification, got %d", len(sent))
	}
}

func TestCompositeNotifier_SendsToMatchingChannels(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier1 := newMockNotifier()
	notifier2 := newMockNotifier()

	composite := NewCompositeNotifier([]ChannelRoute{
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: notifier1,
		},
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess, NotificationTypeVehicleEntry},
			Notifier: notifier2,
		},
	}, logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	if err := composite.Send(ctx, notification); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if len(notifier1.getSent()) != 1 {
		t.Errorf("notifier1 sent %d, want 1", len(notifier1.getSent()))
	}
	if len(notifier2.getSent()) != 1 {
		t.Errorf("notifier2 sent %d, want 1", len(notifier2.getSent()))
	}
}

func TestCompositeNotifier_SkipsNonMatchingChannels(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier1 := newMockNotifier()
	notifier2 := newMockNotifier()

	composite := NewCompositeNotifier([]ChannelRoute{
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: notifier1,
		},
		{
			Types:    []NotificationType{NotificationTypeVehicleEntry},
			Notifier: notifier2,
		},
	}, logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	if err := composite.Send(ctx, notification); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if len(notifier1.getSent()) != 1 {
		t.Errorf("notifier1 sent %d, want 1", len(notifier1.getSent()))
	}
	if len(notifier2.getSent()) != 0 {
		t.Errorf("notifier2 sent %d, want 0", len(notifier2.getSent()))
	}
}

func TestCompositeNotifier_AllChannelsFail(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)

	composite := NewCompositeNotifier([]ChannelRoute{
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: &failingNotifier{},
		},
	}, logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	err := composite.Send(ctx, notification)
	if err == nil {
		t.Error("expected error when all channels fail")
	}
}

func TestCompositeNotifier_PartialFailure(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	successNotifier := newMockNotifier()

	composite := NewCompositeNotifier([]ChannelRoute{
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: &failingNotifier{},
		},
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: successNotifier,
		},
	}, logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	err := composite.Send(ctx, notification)
	if err != nil {
		t.Errorf("expected no error when at least one channel succeeds, got: %v", err)
	}
	if len(successNotifier.getSent()) != 1 {
		t.Errorf("successNotifier sent %d, want 1", len(successNotifier.getSent()))
	}
}

func TestCompositeNotifier_AddRoute(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier1 := newMockNotifier()

	composite := NewCompositeNotifier([]ChannelRoute{
		{
			Types:    []NotificationType{NotificationTypePaymentSuccess},
			Notifier: notifier1,
		},
	}, logger)

	notifier2 := newMockNotifier()
	composite.AddRoute(ChannelRoute{
		Types:    []NotificationType{NotificationTypeVehicleEntry},
		Notifier: notifier2,
	})

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeVehicleEntry,
		Title:     "Entry",
		Content:   "Car entered",
		Recipient: "user-001",
	}

	if err := composite.Send(ctx, notification); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	if len(notifier1.getSent()) != 0 {
		t.Errorf("notifier1 sent %d, want 0", len(notifier1.getSent()))
	}
	if len(notifier2.getSent()) != 1 {
		t.Errorf("notifier2 sent %d, want 1", len(notifier2.getSent()))
	}
}

func TestInAppNotifier_Send(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier := NewInAppNotifier(logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	if err := notifier.Send(ctx, notification); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestEmailNotifier_Send(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier := NewEmailNotifier(logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	if err := notifier.Send(ctx, notification); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestSMSNotifier_Send(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier := NewSMSNotifier(logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypeSystemAlert,
		Title:     "Alert",
		Content:   "System down",
		Recipient: "admin",
	}

	if err := notifier.Send(ctx, notification); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

func TestWechatNotifier_Send(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	notifier := NewWechatNotifier(logger)

	ctx := context.Background()
	notification := &Notification{
		ID:        uuid.New(),
		Type:      NotificationTypePaymentSuccess,
		Title:     "Payment",
		Content:   "Paid",
		Recipient: "user-001",
	}

	if err := notifier.Send(ctx, notification); err != nil {
		t.Errorf("Send() error: %v", err)
	}
}

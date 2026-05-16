package data

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/notification/biz"
)

type notificationRepo struct {
	mu            sync.RWMutex
	notifications map[uuid.UUID]*biz.Notification
}

func NewNotificationRepo() biz.NotificationRepo {
	return &notificationRepo{
		notifications: make(map[uuid.UUID]*biz.Notification),
	}
}

func (r *notificationRepo) Create(ctx context.Context, notification *biz.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifications[notification.ID] = notification
	return nil
}

func (r *notificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*biz.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.notifications[id]
	if !ok {
		return nil, fmt.Errorf("notification not found")
	}
	return n, nil
}

func (r *notificationRepo) ListByRecipient(ctx context.Context, recipientID string, page, pageSize int) ([]*biz.Notification, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*biz.Notification
	for _, n := range r.notifications {
		if n.Recipient == recipientID {
			result = append(result, n)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	total := int64(len(result))
	offset := (page - 1) * pageSize
	if offset >= len(result) {
		return []*biz.Notification{}, total, nil
	}
	end := offset + pageSize
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], total, nil
}

func (r *notificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.notifications[id]
	if !ok {
		return fmt.Errorf("notification not found")
	}
	n.Status = biz.NotificationStatusRead
	now := time.Now()
	n.ReadAt = &now
	return nil
}

func (r *notificationRepo) MarkAllAsRead(ctx context.Context, recipientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, n := range r.notifications {
		if n.Recipient == recipientID && n.Status == biz.NotificationStatusUnread {
			n.Status = biz.NotificationStatusRead
			n.ReadAt = &now
		}
	}
	return nil
}

func (r *notificationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.notifications, id)
	return nil
}

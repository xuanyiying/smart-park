package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/notification/biz"
)

type NotificationService struct {
	uc  *biz.NotificationUseCase
	log *log.Helper
}

func NewNotificationService(uc *biz.NotificationUseCase, logger log.Logger) *NotificationService {
	return &NotificationService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

type SendNotificationRequest struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Recipient string `json:"recipient"`
}

type NotificationResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Recipient string `json:"recipient"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	ReadAt    string `json:"read_at,omitempty"`
}

type ListNotificationsRequest struct {
	RecipientID string `json:"recipient_id"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

type ListNotificationsResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    []*NotificationResponse `json:"data"`
	Total   int64                  `json:"total"`
}

type MarkReadRequest struct {
	ID string `json:"id"`
}

type MarkAllReadRequest struct {
	RecipientID string `json:"recipient_id"`
}

type SimpleResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SingleNotificationResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    *NotificationResponse `json:"data,omitempty"`
}

func (s *NotificationService) SendNotification(ctx context.Context, req *SendNotificationRequest) (*SingleNotificationResponse, error) {
	notification := &biz.Notification{
		ID:        uuid.New(),
		Type:      biz.NotificationType(req.Type),
		Title:     req.Title,
		Content:   req.Content,
		Recipient: req.Recipient,
		Status:    biz.NotificationStatusUnread,
		CreatedAt: time.Now(),
	}

	result, err := s.uc.SendNotification(ctx, notification)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendNotification failed: %v", err)
		return &SingleNotificationResponse{Code: 500, Message: "failed to send notification"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(result),
	}, nil
}

func (s *NotificationService) GetNotification(ctx context.Context, id string) (*SingleNotificationResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return &SingleNotificationResponse{Code: 400, Message: "invalid notification id"}, nil
	}

	notification, err := s.uc.GetNotification(ctx, uid)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetNotification failed: %v", err)
		return &SingleNotificationResponse{Code: 404, Message: "notification not found"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(notification),
	}, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, req *ListNotificationsRequest) (*ListNotificationsResponse, error) {
	notifications, total, err := s.uc.ListNotifications(ctx, req.RecipientID, req.Page, req.PageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListNotifications failed: %v", err)
		return &ListNotificationsResponse{Code: 500, Message: "failed to list notifications"}, nil
	}

	data := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		data[i] = toNotificationResponse(n)
	}

	return &ListNotificationsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
	}, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, req *MarkReadRequest) (*SimpleResponse, error) {
	uid, err := uuid.Parse(req.ID)
	if err != nil {
		return &SimpleResponse{Code: 400, Message: "invalid notification id"}, nil
	}

	if err := s.uc.MarkAsRead(ctx, uid); err != nil {
		s.log.WithContext(ctx).Errorf("MarkAsRead failed: %v", err)
		return &SimpleResponse{Code: 500, Message: "failed to mark as read"}, nil
	}

	return &SimpleResponse{Code: 0, Message: "success"}, nil
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, req *MarkAllReadRequest) (*SimpleResponse, error) {
	if err := s.uc.MarkAllAsRead(ctx, req.RecipientID); err != nil {
		s.log.WithContext(ctx).Errorf("MarkAllAsRead failed: %v", err)
		return &SimpleResponse{Code: 500, Message: "failed to mark all as read"}, nil
	}

	return &SimpleResponse{Code: 0, Message: "success"}, nil
}

func (s *NotificationService) SendPaymentSuccess(ctx context.Context, orderID, amount, method string) (*SingleNotificationResponse, error) {
	result, err := s.uc.SendPaymentSuccess(ctx, orderID, amount, method)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendPaymentSuccess failed: %v", err)
		return &SingleNotificationResponse{Code: 500, Message: "failed to send payment notification"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(result),
	}, nil
}

func (s *NotificationService) SendVehicleEntry(ctx context.Context, plateNumber, lotName string) (*SingleNotificationResponse, error) {
	result, err := s.uc.SendVehicleEntry(ctx, plateNumber, lotName)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendVehicleEntry failed: %v", err)
		return &SingleNotificationResponse{Code: 500, Message: "failed to send entry notification"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(result),
	}, nil
}

func (s *NotificationService) SendVehicleExit(ctx context.Context, plateNumber, lotName, duration string, fee float64) (*SingleNotificationResponse, error) {
	result, err := s.uc.SendVehicleExit(ctx, plateNumber, lotName, duration, fee)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendVehicleExit failed: %v", err)
		return &SingleNotificationResponse{Code: 500, Message: "failed to send exit notification"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(result),
	}, nil
}

func (s *NotificationService) SendMonthlyExpiry(ctx context.Context, plateNumber, expiryDate string) (*SingleNotificationResponse, error) {
	result, err := s.uc.SendMonthlyExpiry(ctx, plateNumber, expiryDate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendMonthlyExpiry failed: %v", err)
		return &SingleNotificationResponse{Code: 500, Message: "failed to send monthly expiry notification"}, nil
	}

	return &SingleNotificationResponse{
		Code:    0,
		Message: "success",
		Data:    toNotificationResponse(result),
	}, nil
}

func toNotificationResponse(n *biz.Notification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:        n.ID.String(),
		Type:      string(n.Type),
		Title:     n.Title,
		Content:   n.Content,
		Recipient: n.Recipient,
		Status:    string(n.Status),
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
	if n.ReadAt != nil {
		resp.ReadAt = n.ReadAt.Format(time.RFC3339)
	}
	return resp
}

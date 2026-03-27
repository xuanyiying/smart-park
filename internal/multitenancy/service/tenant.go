// Package service provides gRPC service implementation for the multitenancy service.
package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/multitenancy/v1"
	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

// TenantService implements the TenantService gRPC service.
type TenantService struct {
	v1.UnimplementedTenantServiceServer

	uc  *biz.TenantUseCase
	log *log.Helper
}

// NewTenantService creates a new TenantService.
func NewTenantService(uc *biz.TenantUseCase, logger log.Logger) *TenantService {
	return &TenantService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CreateTenant creates a new tenant.
func (s *TenantService) CreateTenant(ctx context.Context, req *v1.CreateTenantRequest) (*v1.CreateTenantResponse, error) {
	var config *biz.TenantConfig
	if req.Config != nil {
		config = &biz.TenantConfig{
			MaxParkingLots: int(req.Config.MaxParkingLots),
			MaxDevices:     int(req.Config.MaxDevices),
			MaxUsers:       int(req.Config.MaxUsers),
			StorageQuota:   req.Config.StorageQuota,
			Features:       req.Config.Features,
			CustomDomain:   req.Config.CustomDomain,
			Timezone:       req.Config.Timezone,
			Currency:       req.Config.Currency,
			Language:       req.Config.Language,
		}
	}

	tenant, err := s.uc.CreateTenant(ctx, req.Name, req.Code, config)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateTenant failed: %v", err)
		return &v1.CreateTenantResponse{Code: 500, Message: "创建租户失败"}, nil
	}

	return &v1.CreateTenantResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoTenant(tenant),
	}, nil
}

// GetTenant retrieves a tenant by ID.
func (s *TenantService) GetTenant(ctx context.Context, req *v1.GetTenantRequest) (*v1.GetTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.GetTenantResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	tenant, err := s.uc.GetTenant(ctx, tenantID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetTenant failed: %v", err)
		return &v1.GetTenantResponse{Code: 404, Message: "租户不存在"}, nil
	}

	return &v1.GetTenantResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoTenant(tenant),
	}, nil
}

// GetTenantByCode retrieves a tenant by code.
func (s *TenantService) GetTenantByCode(ctx context.Context, req *v1.GetTenantByCodeRequest) (*v1.GetTenantByCodeResponse, error) {
	tenant, err := s.uc.GetTenantByCode(ctx, req.Code)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetTenantByCode failed: %v", err)
		return &v1.GetTenantByCodeResponse{Code: 404, Message: "租户不存在"}, nil
	}

	return &v1.GetTenantByCodeResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoTenant(tenant),
	}, nil
}

// UpdateTenant updates a tenant.
func (s *TenantService) UpdateTenant(ctx context.Context, req *v1.UpdateTenantRequest) (*v1.UpdateTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.UpdateTenantResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	var config *biz.TenantConfig
	if req.Config != nil {
		config = &biz.TenantConfig{
			MaxParkingLots: int(req.Config.MaxParkingLots),
			MaxDevices:     int(req.Config.MaxDevices),
			MaxUsers:       int(req.Config.MaxUsers),
			StorageQuota:   req.Config.StorageQuota,
			Features:       req.Config.Features,
			CustomDomain:   req.Config.CustomDomain,
			Timezone:       req.Config.Timezone,
			Currency:       req.Config.Currency,
			Language:       req.Config.Language,
		}
	}

	tenant, err := s.uc.UpdateTenant(ctx, tenantID, req.Name, config)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateTenant failed: %v", err)
		return &v1.UpdateTenantResponse{Code: 500, Message: "更新租户失败"}, nil
	}

	return &v1.UpdateTenantResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoTenant(tenant),
	}, nil
}

// DeleteTenant deletes a tenant.
func (s *TenantService) DeleteTenant(ctx context.Context, req *v1.DeleteTenantRequest) (*v1.DeleteTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.DeleteTenantResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	if err := s.uc.DeleteTenant(ctx, tenantID); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteTenant failed: %v", err)
		return &v1.DeleteTenantResponse{Code: 500, Message: "删除租户失败"}, nil
	}

	return &v1.DeleteTenantResponse{Code: 0, Message: "success"}, nil
}

// ListTenants lists all tenants with pagination.
func (s *TenantService) ListTenants(ctx context.Context, req *v1.ListTenantsRequest) (*v1.ListTenantsResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	tenants, total, err := s.uc.ListTenants(ctx, page, pageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListTenants failed: %v", err)
		return &v1.ListTenantsResponse{Code: 500, Message: "获取租户列表失败"}, nil
	}

	data := make([]*v1.Tenant, len(tenants))
	for i, tenant := range tenants {
		data[i] = toProtoTenant(tenant)
	}

	return &v1.ListTenantsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
	}, nil
}

// EnableTenant enables a tenant.
func (s *TenantService) EnableTenant(ctx context.Context, req *v1.EnableTenantRequest) (*v1.EnableTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.EnableTenantResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	if err := s.uc.EnableTenant(ctx, tenantID); err != nil {
		s.log.WithContext(ctx).Errorf("EnableTenant failed: %v", err)
		return &v1.EnableTenantResponse{Code: 500, Message: "启用租户失败"}, nil
	}

	return &v1.EnableTenantResponse{Code: 0, Message: "success"}, nil
}

// DisableTenant disables a tenant.
func (s *TenantService) DisableTenant(ctx context.Context, req *v1.DisableTenantRequest) (*v1.DisableTenantResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.DisableTenantResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	if err := s.uc.DisableTenant(ctx, tenantID); err != nil {
		s.log.WithContext(ctx).Errorf("DisableTenant failed: %v", err)
		return &v1.DisableTenantResponse{Code: 500, Message: "禁用租户失败"}, nil
	}

	return &v1.DisableTenantResponse{Code: 0, Message: "success"}, nil
}

// CheckFeature checks if a tenant has a specific feature.
func (s *TenantService) CheckFeature(ctx context.Context, req *v1.CheckFeatureRequest) (*v1.CheckFeatureResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.CheckFeatureResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	if err := s.uc.CheckFeature(ctx, tenantID, req.Feature); err != nil {
		return &v1.CheckFeatureResponse{
			Code:       0,
			Message:    "success",
			HasFeature: false,
		}, nil
	}

	return &v1.CheckFeatureResponse{
		Code:       0,
		Message:    "success",
		HasFeature: true,
	}, nil
}

// CheckQuota checks if tenant has exceeded resource quota.
func (s *TenantService) CheckQuota(ctx context.Context, req *v1.CheckQuotaRequest) (*v1.CheckQuotaResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return &v1.CheckQuotaResponse{Code: 400, Message: "无效的租户ID"}, nil
	}

	tenant, err := s.uc.GetTenant(ctx, tenantID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CheckQuota failed: %v", err)
		return &v1.CheckQuotaResponse{Code: 404, Message: "租户不存在"}, nil
	}

	var limit int
	switch req.ResourceType {
	case "parking_lots":
		limit = tenant.Config.MaxParkingLots
	case "devices":
		limit = tenant.Config.MaxDevices
	case "users":
		limit = tenant.Config.MaxUsers
	default:
		return &v1.CheckQuotaResponse{Code: 400, Message: "无效的资源类型"}, nil
	}

	allowed := int(req.CurrentCount) < limit
	remaining := limit - int(req.CurrentCount)
	if remaining < 0 {
		remaining = 0
	}

	return &v1.CheckQuotaResponse{
		Code:      0,
		Message:   "success",
		Allowed:   allowed,
		Limit:     int32(limit),
		Remaining: int32(remaining),
	}, nil
}

// Helper function to convert biz type to proto type
func toProtoTenant(t *biz.Tenant) *v1.Tenant {
	tenant := &v1.Tenant{
		Id:     t.ID.String(),
		Name:   t.Name,
		Code:   t.Code,
		Status: t.Status,
		Config: &v1.TenantConfig{
			MaxParkingLots: int32(t.Config.MaxParkingLots),
			MaxDevices:     int32(t.Config.MaxDevices),
			MaxUsers:       int32(t.Config.MaxUsers),
			StorageQuota:   t.Config.StorageQuota,
			Features:       t.Config.Features,
			CustomDomain:   t.Config.CustomDomain,
			Timezone:       t.Config.Timezone,
			Currency:       t.Config.Currency,
			Language:       t.Config.Language,
		},
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
	if t.ExpiredAt != nil {
		tenant.ExpiredAt = t.ExpiredAt.Format(time.RFC3339)
	}
	return tenant
}

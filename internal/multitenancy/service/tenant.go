package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

// TenantService implements tenant service interface
type TenantService struct {
	uc     *biz.TenantUseCase
	logger *log.Helper
}

// NewTenantService creates a new tenant service
func NewTenantService(uc *biz.TenantUseCase, logger log.Logger) *TenantService {
	return &TenantService{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
}

// CreateTenantRequest represents create tenant request
type CreateTenantRequest struct {
	Name   string
	Code   string
	Config *biz.TenantConfig
}

// CreateTenantResponse represents create tenant response
type CreateTenantResponse struct {
	Tenant *biz.Tenant
}

// CreateTenant creates a new tenant
func (s *TenantService) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*CreateTenantResponse, error) {
	s.logger.WithContext(ctx).Infof("CreateTenant called: %s", req.Name)

	tenant, err := s.uc.CreateTenant(ctx, req.Name, req.Code, req.Config)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("CreateTenant failed: %v", err)
		return nil, err
	}

	return &CreateTenantResponse{Tenant: tenant}, nil
}

// GetTenantRequest represents get tenant request
type GetTenantRequest struct {
	ID uuid.UUID
}

// GetTenantResponse represents get tenant response
type GetTenantResponse struct {
	Tenant *biz.Tenant
}

// GetTenant retrieves a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, req *GetTenantRequest) (*GetTenantResponse, error) {
	s.logger.WithContext(ctx).Infof("GetTenant called: %s", req.ID)

	tenant, err := s.uc.GetTenant(ctx, req.ID)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetTenant failed: %v", err)
		return nil, err
	}

	return &GetTenantResponse{Tenant: tenant}, nil
}

// GetTenantByCodeRequest represents get tenant by code request
type GetTenantByCodeRequest struct {
	Code string
}

// GetTenantByCodeResponse represents get tenant by code response
type GetTenantByCodeResponse struct {
	Tenant *biz.Tenant
}

// GetTenantByCode retrieves a tenant by code
func (s *TenantService) GetTenantByCode(ctx context.Context, req *GetTenantByCodeRequest) (*GetTenantByCodeResponse, error) {
	s.logger.WithContext(ctx).Infof("GetTenantByCode called: %s", req.Code)

	tenant, err := s.uc.GetTenantByCode(ctx, req.Code)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetTenantByCode failed: %v", err)
		return nil, err
	}

	return &GetTenantByCodeResponse{Tenant: tenant}, nil
}

// UpdateTenantRequest represents update tenant request
type UpdateTenantRequest struct {
	ID     uuid.UUID
	Name   string
	Config *biz.TenantConfig
}

// UpdateTenantResponse represents update tenant response
type UpdateTenantResponse struct {
	Tenant *biz.Tenant
}

// UpdateTenant updates a tenant
func (s *TenantService) UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*UpdateTenantResponse, error) {
	s.logger.WithContext(ctx).Infof("UpdateTenant called: %s", req.ID)

	tenant, err := s.uc.UpdateTenant(ctx, req.ID, req.Name, req.Config)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("UpdateTenant failed: %v", err)
		return nil, err
	}

	return &UpdateTenantResponse{Tenant: tenant}, nil
}

// DisableTenantRequest represents disable tenant request
type DisableTenantRequest struct {
	ID uuid.UUID
}

// DisableTenant disables a tenant
func (s *TenantService) DisableTenant(ctx context.Context, req *DisableTenantRequest) error {
	s.logger.WithContext(ctx).Infof("DisableTenant called: %s", req.ID)
	return s.uc.DisableTenant(ctx, req.ID)
}

// EnableTenantRequest represents enable tenant request
type EnableTenantRequest struct {
	ID uuid.UUID
}

// EnableTenant enables a tenant
func (s *TenantService) EnableTenant(ctx context.Context, req *EnableTenantRequest) error {
	s.logger.WithContext(ctx).Infof("EnableTenant called: %s", req.ID)
	return s.uc.EnableTenant(ctx, req.ID)
}

// DeleteTenantRequest represents delete tenant request
type DeleteTenantRequest struct {
	ID uuid.UUID
}

// DeleteTenant deletes a tenant
func (s *TenantService) DeleteTenant(ctx context.Context, req *DeleteTenantRequest) error {
	s.logger.WithContext(ctx).Infof("DeleteTenant called: %s", req.ID)
	return s.uc.DeleteTenant(ctx, req.ID)
}

// ListTenantsRequest represents list tenants request
type ListTenantsRequest struct {
	Page     int
	PageSize int
}

// ListTenantsResponse represents list tenants response
type ListTenantsResponse struct {
	Tenants []*biz.Tenant
	Total   int64
}

// ListTenants lists all tenants with pagination
func (s *TenantService) ListTenants(ctx context.Context, req *ListTenantsRequest) (*ListTenantsResponse, error) {
	s.logger.WithContext(ctx).Infof("ListTenants called: page=%d, pageSize=%d", req.Page, req.PageSize)

	tenants, total, err := s.uc.ListTenants(ctx, req.Page, req.PageSize)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("ListTenants failed: %v", err)
		return nil, err
	}

	return &ListTenantsResponse{
		Tenants: tenants,
		Total:   total,
	}, nil
}

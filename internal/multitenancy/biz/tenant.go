package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID        uuid.UUID
	Name      string
	Code      string
	Status    string
	Config    TenantConfig
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiredAt *time.Time
}

// TenantConfig holds tenant-specific configuration
type TenantConfig struct {
	MaxParkingLots int
	MaxDevices     int
	MaxUsers       int
	StorageQuota   int64
	Features       []string
	CustomDomain   string
	Timezone       string
	Currency       string
	Language       string
}

// TenantInfo is a lightweight version of Tenant for context
type TenantInfo struct {
	ID     uuid.UUID
	Code   string
	Name   string
	Config TenantConfig
}

// IsValid checks if tenant is valid and active
func (t *Tenant) IsValid() bool {
	if t == nil {
		return false
	}
	if t.Status != "active" {
		return false
	}
	if t.ExpiredAt != nil && time.Now().After(*t.ExpiredAt) {
		return false
	}
	return true
}

// HasFeature checks if tenant has a specific feature enabled
func (t *Tenant) HasFeature(feature string) bool {
	if t == nil || t.Config.Features == nil {
		return false
	}
	for _, f := range t.Config.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// ToInfo converts Tenant to TenantInfo
func (t *Tenant) ToInfo() *TenantInfo {
	if t == nil {
		return nil
	}
	return &TenantInfo{
		ID:     t.ID,
		Code:   t.Code,
		Name:   t.Name,
		Config: t.Config,
	}
}

// TenantRepo interface for tenant data access
type TenantRepo interface {
	Create(ctx context.Context, tenant *Tenant) error
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetByCode(ctx context.Context, code string) (*Tenant, error)
	List(ctx context.Context, page, pageSize int) ([]*Tenant, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// TenantUseCase handles tenant business logic
type TenantUseCase struct {
	repo   TenantRepo
	logger *log.Helper
}

// NewTenantUseCase creates a new tenant use case
func NewTenantUseCase(repo TenantRepo, logger log.Logger) *TenantUseCase {
	return &TenantUseCase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// CreateTenant creates a new tenant
func (uc *TenantUseCase) CreateTenant(ctx context.Context, name, code string, config *TenantConfig) (*Tenant, error) {
	uc.logger.WithContext(ctx).Infof("Creating tenant: %s, code: %s", name, code)

	// Check if code already exists
	existing, _ := uc.repo.GetByCode(ctx, code)
	if existing != nil {
		return nil, ErrDuplicateTenantCode
	}

	if config == nil {
		defaultConfig := DefaultTenantConfig()
		config = &defaultConfig
	}

	tenant := &Tenant{
		ID:        uuid.New(),
		Name:      name,
		Code:      code,
		Status:    "active",
		Config:    *config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.repo.Create(ctx, tenant); err != nil {
		uc.logger.WithContext(ctx).Errorf("Failed to create tenant: %v", err)
		return nil, err
	}

	return tenant, nil
}

// GetTenant retrieves a tenant by ID
func (uc *TenantUseCase) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return uc.repo.GetByID(ctx, id)
}

// GetTenantByCode retrieves a tenant by code
func (uc *TenantUseCase) GetTenantByCode(ctx context.Context, code string) (*Tenant, error) {
	return uc.repo.GetByCode(ctx, code)
}

// UpdateTenant updates tenant information
func (uc *TenantUseCase) UpdateTenant(ctx context.Context, id uuid.UUID, name string, config *TenantConfig) (*Tenant, error) {
	uc.logger.WithContext(ctx).Infof("Updating tenant: %s", id)

	tenant, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		tenant.Name = name
	}
	if config != nil {
		tenant.Config = *config
	}
	tenant.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, tenant); err != nil {
		uc.logger.WithContext(ctx).Errorf("Failed to update tenant: %v", err)
		return nil, err
	}

	return tenant, nil
}

// DisableTenant disables a tenant
func (uc *TenantUseCase) DisableTenant(ctx context.Context, id uuid.UUID) error {
	uc.logger.WithContext(ctx).Infof("Disabling tenant: %s", id)
	return uc.repo.UpdateStatus(ctx, id, "disabled")
}

// EnableTenant enables a tenant
func (uc *TenantUseCase) EnableTenant(ctx context.Context, id uuid.UUID) error {
	uc.logger.WithContext(ctx).Infof("Enabling tenant: %s", id)
	return uc.repo.UpdateStatus(ctx, id, "active")
}

// DeleteTenant deletes a tenant
func (uc *TenantUseCase) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	uc.logger.WithContext(ctx).Infof("Deleting tenant: %s", id)
	return uc.repo.Delete(ctx, id)
}

// ListTenants lists all tenants with pagination
func (uc *TenantUseCase) ListTenants(ctx context.Context, page, pageSize int) ([]*Tenant, int64, error) {
	return uc.repo.List(ctx, page, pageSize)
}

// CheckFeature checks if a tenant has access to a feature
func (uc *TenantUseCase) CheckFeature(ctx context.Context, tenantID uuid.UUID, feature string) error {
	tenant, err := uc.repo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	if !tenant.HasFeature(feature) {
		return ErrFeatureNotAvailable
	}

	return nil
}

// CheckQuota checks if tenant has exceeded resource quota
func (uc *TenantUseCase) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType string, currentCount int) error {
	tenant, err := uc.repo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	var limit int
	switch resourceType {
	case "parking_lots":
		limit = tenant.Config.MaxParkingLots
	case "devices":
		limit = tenant.Config.MaxDevices
	case "users":
		limit = tenant.Config.MaxUsers
	default:
		return nil
	}

	if currentCount >= limit {
		return ErrQuotaExceeded
	}

	return nil
}

// DefaultTenantConfig returns default tenant configuration
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		MaxParkingLots: 1,
		MaxDevices:     10,
		MaxUsers:       50,
		StorageQuota:   1024 * 1024 * 100, // 100MB
		Features:       []string{"basic"},
		Timezone:       "Asia/Shanghai",
		Currency:       "CNY",
		Language:       "zh-CN",
	}
}

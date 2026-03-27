// Package data provides data access layer for the multitenancy service.
package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
	"github.com/xuanyiying/smart-park/internal/multitenancy/data/ent"
	"github.com/xuanyiying/smart-park/internal/multitenancy/data/ent/tenant"
)

// tenantRepo implements biz.TenantRepo.
type tenantRepo struct {
	data *Data
}

// NewTenantRepo creates a new TenantRepo.
func NewTenantRepo(data *Data) biz.TenantRepo {
	return &tenantRepo{data: data}
}

// Create creates a new tenant.
func (r *tenantRepo) Create(ctx context.Context, t *biz.Tenant) error {
	builder := r.data.db.Tenant.Create().
		SetID(t.ID).
		SetName(t.Name).
		SetCode(t.Code).
		SetStatus(tenant.Status(t.Status)).
		SetMaxParkingLots(t.Config.MaxParkingLots).
		SetMaxDevices(t.Config.MaxDevices).
		SetMaxUsers(t.Config.MaxUsers).
		SetStorageQuota(t.Config.StorageQuota).
		SetFeatures(t.Config.Features).
		SetTimezone(t.Config.Timezone).
		SetCurrency(t.Config.Currency).
		SetLanguage(t.Config.Language)

	if t.Config.CustomDomain != "" {
		builder.SetCustomDomain(t.Config.CustomDomain)
	}
	if t.ExpiredAt != nil {
		builder.SetExpiredAt(*t.ExpiredAt)
	}

	_, err := builder.Save(ctx)
	return err
}

// Update updates a tenant.
func (r *tenantRepo) Update(ctx context.Context, t *biz.Tenant) error {
	builder := r.data.db.Tenant.UpdateOneID(t.ID).
		SetName(t.Name).
		SetStatus(tenant.Status(t.Status)).
		SetMaxParkingLots(t.Config.MaxParkingLots).
		SetMaxDevices(t.Config.MaxDevices).
		SetMaxUsers(t.Config.MaxUsers).
		SetStorageQuota(t.Config.StorageQuota).
		SetFeatures(t.Config.Features).
		SetTimezone(t.Config.Timezone).
		SetCurrency(t.Config.Currency).
		SetLanguage(t.Config.Language)

	if t.Config.CustomDomain != "" {
		builder.SetCustomDomain(t.Config.CustomDomain)
	}
	if t.ExpiredAt != nil {
		builder.SetExpiredAt(*t.ExpiredAt)
	} else {
		builder.ClearExpiredAt()
	}

	_, err := builder.Save(ctx)
	return err
}

// Delete deletes a tenant.
func (r *tenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.data.db.Tenant.DeleteOneID(id).Exec(ctx)
}

// GetByID retrieves a tenant by ID.
func (r *tenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*biz.Tenant, error) {
	t, err := r.data.db.Tenant.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrTenantNotFound
		}
		return nil, err
	}
	return toBizTenant(t), nil
}

// GetByCode retrieves a tenant by code.
func (r *tenantRepo) GetByCode(ctx context.Context, code string) (*biz.Tenant, error) {
	t, err := r.data.db.Tenant.Query().
		Where(tenant.Code(code)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrTenantNotFound
		}
		return nil, err
	}
	return toBizTenant(t), nil
}

// List retrieves a list of tenants with pagination.
func (r *tenantRepo) List(ctx context.Context, page, pageSize int) ([]*biz.Tenant, int64, error) {
	query := r.data.db.Tenant.Query()

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	tenants, err := query.
		Order(ent.Desc(tenant.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Tenant, len(tenants))
	for i, t := range tenants {
		result[i] = toBizTenant(t)
	}
	return result, int64(total), nil
}

// UpdateStatus updates tenant status.
func (r *tenantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.data.db.Tenant.UpdateOneID(id).
		SetStatus(tenant.Status(status)).
		Save(ctx)
	return err
}

// Helper function to convert ent type to biz type
func toBizTenant(t *ent.Tenant) *biz.Tenant {
	return &biz.Tenant{
		ID:     t.ID,
		Name:   t.Name,
		Code:   t.Code,
		Status: string(t.Status),
		Config: biz.TenantConfig{
			MaxParkingLots: t.MaxParkingLots,
			MaxDevices:     t.MaxDevices,
			MaxUsers:       t.MaxUsers,
			StorageQuota:   t.StorageQuota,
			Features:       t.Features,
			CustomDomain:   t.CustomDomain,
			Timezone:       t.Timezone,
			Currency:       t.Currency,
			Language:       t.Language,
		},
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		ExpiredAt: t.ExpiredAt,
	}
}

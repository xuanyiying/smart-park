package biz

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type mockTenantRepo struct {
	tenants    map[uuid.UUID]*Tenant
	codeIndex  map[string]*Tenant
	createErr  error
	updateErr  error
	getByIDErr error
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants:   make(map[uuid.UUID]*Tenant),
		codeIndex: make(map[string]*Tenant),
	}
}

func (m *mockTenantRepo) Create(ctx context.Context, tenant *Tenant) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.tenants[tenant.ID] = tenant
	m.codeIndex[tenant.Code] = tenant
	return nil
}

func (m *mockTenantRepo) Update(ctx context.Context, tenant *Tenant) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.tenants[tenant.ID] = tenant
	m.codeIndex[tenant.Code] = tenant
	return nil
}

func (m *mockTenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if t, ok := m.tenants[id]; ok {
		delete(m.codeIndex, t.Code)
		delete(m.tenants, id)
	}
	return nil
}

func (m *mockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	return m.tenants[id], nil
}

func (m *mockTenantRepo) GetByCode(ctx context.Context, code string) (*Tenant, error) {
	return m.codeIndex[code], nil
}

func (m *mockTenantRepo) List(ctx context.Context, page, pageSize int) ([]*Tenant, int64, error) {
	var result []*Tenant
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result, int64(len(result)), nil
}

func (m *mockTenantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if t, ok := m.tenants[id]; ok {
		t.Status = status
	}
	return nil
}

func TestTenant_IsValid(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)
	pastTime := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name   string
		tenant *Tenant
		expect bool
	}{
		{
			name:   "nil tenant",
			tenant: nil,
			expect: false,
		},
		{
			name: "active tenant no expiry",
			tenant: &Tenant{
				Status: "active",
			},
			expect: true,
		},
		{
			name: "disabled tenant",
			tenant: &Tenant{
				Status: "disabled",
			},
			expect: false,
		},
		{
			name: "active tenant with future expiry",
			tenant: &Tenant{
				Status:    "active",
				ExpiredAt: &futureTime,
			},
			expect: true,
		},
		{
			name: "active tenant with past expiry",
			tenant: &Tenant{
				Status:    "active",
				ExpiredAt: &pastTime,
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.IsValid()
			if got != tt.expect {
				t.Errorf("Tenant.IsValid() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestTenant_HasFeature(t *testing.T) {
	tests := []struct {
		name    string
		tenant  *Tenant
		feature string
		expect  bool
	}{
		{
			name:    "nil tenant",
			tenant:  nil,
			feature: "basic",
			expect:  false,
		},
		{
			name: "nil features",
			tenant: &Tenant{
				Config: TenantConfig{Features: nil},
			},
			feature: "basic",
			expect:  false,
		},
		{
			name: "feature present",
			tenant: &Tenant{
				Config: TenantConfig{Features: []string{"basic", "advanced"}},
			},
			feature: "advanced",
			expect:  true,
		},
		{
			name: "feature absent",
			tenant: &Tenant{
				Config: TenantConfig{Features: []string{"basic"}},
			},
			feature: "premium",
			expect:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.HasFeature(tt.feature)
			if got != tt.expect {
				t.Errorf("Tenant.HasFeature() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestTenant_ToInfo(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
		expect bool
	}{
		{
			name:   "nil tenant returns nil",
			tenant: nil,
			expect: false,
		},
		{
			name: "valid tenant returns info",
			tenant: &Tenant{
				ID:     uuid.New(),
				Code:   "test",
				Name:   "Test Tenant",
				Config: DefaultTenantConfig(),
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tt.tenant.ToInfo()
			if (info != nil) != tt.expect {
				t.Errorf("Tenant.ToInfo() returned %v, expect non-nil = %v", info, tt.expect)
			}
		})
	}
}

func TestTenantUseCase_CreateTenant(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)

	tests := []struct {
		name    string
		repo    *mockTenantRepo
		tName   string
		code    string
		config  *TenantConfig
		wantErr error
	}{
		{
			name:   "successful creation",
			repo:   newMockTenantRepo(),
			tName:  "Test Tenant",
			code:   "test001",
			config: nil,
		},
		{
			name: "duplicate code",
			repo: func() *mockTenantRepo {
				r := newMockTenantRepo()
				existing := &Tenant{
					ID:     uuid.New(),
					Name:   "Existing",
					Code:   "dup001",
					Status: "active",
				}
				r.tenants[existing.ID] = existing
				r.codeIndex["dup001"] = existing
				return r
			}(),
			tName:   "New Tenant",
			code:    "dup001",
			config:  nil,
			wantErr: ErrDuplicateTenantCode,
		},
		{
			name:   "custom config",
			repo:   newMockTenantRepo(),
			tName:  "Custom Tenant",
			code:   "custom001",
			config: &TenantConfig{MaxParkingLots: 5, MaxDevices: 50, MaxUsers: 200, Features: []string{"basic", "advanced"}},
		},
		{
			name:    "repo create error",
			repo:    &mockTenantRepo{tenants: make(map[uuid.UUID]*Tenant), codeIndex: make(map[string]*Tenant), createErr: errors.New("db error")},
			tName:   "Fail Tenant",
			code:    "fail001",
			config:  nil,
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewTenantUseCase(tt.repo, logger)
			tenant, err := uc.CreateTenant(context.Background(), tt.tName, tt.code, tt.config)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("CreateTenant() expected error, got nil")
					return
				}
				if tt.wantErr.Error() != err.Error() && !errors.Is(err, tt.wantErr) {
					t.Errorf("CreateTenant() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateTenant() unexpected error: %v", err)
				return
			}
			if tenant.Name != tt.tName {
				t.Errorf("tenant.Name = %s, want %s", tenant.Name, tt.tName)
			}
			if tenant.Code != tt.code {
				t.Errorf("tenant.Code = %s, want %s", tenant.Code, tt.code)
			}
			if tenant.Status != "active" {
				t.Errorf("tenant.Status = %s, want active", tenant.Status)
			}
			if tt.config == nil {
				if tenant.Config.MaxParkingLots != 1 {
					t.Errorf("expected default config, MaxParkingLots = %d", tenant.Config.MaxParkingLots)
				}
			} else {
				if tenant.Config.MaxParkingLots != tt.config.MaxParkingLots {
					t.Errorf("tenant.Config.MaxParkingLots = %d, want %d", tenant.Config.MaxParkingLots, tt.config.MaxParkingLots)
				}
			}
		})
	}
}

func TestTenantUseCase_CheckFeature(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	tenantID := uuid.New()

	repo := newMockTenantRepo()
	repo.tenants[tenantID] = &Tenant{
		ID:     tenantID,
		Code:   "feat001",
		Name:   "Feature Tenant",
		Status: "active",
		Config: TenantConfig{Features: []string{"basic", "charging"}},
	}

	uc := NewTenantUseCase(repo, logger)

	tests := []struct {
		name    string
		feature string
		wantErr error
	}{
		{
			name:    "available feature",
			feature: "charging",
			wantErr: nil,
		},
		{
			name:    "unavailable feature",
			feature: "premium",
			wantErr: ErrFeatureNotAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.CheckFeature(context.Background(), tenantID, tt.feature)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CheckFeature() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("CheckFeature() unexpected error: %v", err)
			}
		})
	}
}

func TestTenantUseCase_CheckQuota(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	tenantID := uuid.New()

	repo := newMockTenantRepo()
	repo.tenants[tenantID] = &Tenant{
		ID:     tenantID,
		Code:   "quota001",
		Name:   "Quota Tenant",
		Status: "active",
		Config: TenantConfig{
			MaxParkingLots: 3,
			MaxDevices:     10,
			MaxUsers:       50,
		},
	}

	uc := NewTenantUseCase(repo, logger)

	tests := []struct {
		name         string
		resourceType string
		currentCount int
		wantErr      error
	}{
		{
			name:         "parking_lots under quota",
			resourceType: "parking_lots",
			currentCount: 2,
			wantErr:      nil,
		},
		{
			name:         "parking_lots at quota",
			resourceType: "parking_lots",
			currentCount: 3,
			wantErr:      ErrQuotaExceeded,
		},
		{
			name:         "devices under quota",
			resourceType: "devices",
			currentCount: 5,
			wantErr:      nil,
		},
		{
			name:         "devices at quota",
			resourceType: "devices",
			currentCount: 10,
			wantErr:      ErrQuotaExceeded,
		},
		{
			name:         "users over quota",
			resourceType: "users",
			currentCount: 55,
			wantErr:      ErrQuotaExceeded,
		},
		{
			name:         "unknown resource type",
			resourceType: "unknown",
			currentCount: 999,
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.CheckQuota(context.Background(), tenantID, tt.resourceType, tt.currentCount)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CheckQuota() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("CheckQuota() unexpected error: %v", err)
			}
		})
	}
}

func TestTenantUseCase_DisableEnableTenant(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	tenantID := uuid.New()

	repo := newMockTenantRepo()
	repo.tenants[tenantID] = &Tenant{
		ID:     tenantID,
		Code:   "toggle001",
		Name:   "Toggle Tenant",
		Status: "active",
	}

	uc := NewTenantUseCase(repo, logger)

	if err := uc.DisableTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("DisableTenant() error: %v", err)
	}
	if repo.tenants[tenantID].Status != "disabled" {
		t.Errorf("status = %s, want disabled", repo.tenants[tenantID].Status)
	}

	if err := uc.EnableTenant(context.Background(), tenantID); err != nil {
		t.Fatalf("EnableTenant() error: %v", err)
	}
	if repo.tenants[tenantID].Status != "active" {
		t.Errorf("status = %s, want active", repo.tenants[tenantID].Status)
	}
}

func TestDefaultTenantConfig(t *testing.T) {
	cfg := DefaultTenantConfig()

	if cfg.MaxParkingLots != 1 {
		t.Errorf("MaxParkingLots = %d, want 1", cfg.MaxParkingLots)
	}
	if cfg.MaxDevices != 10 {
		t.Errorf("MaxDevices = %d, want 10", cfg.MaxDevices)
	}
	if cfg.MaxUsers != 50 {
		t.Errorf("MaxUsers = %d, want 50", cfg.MaxUsers)
	}
	if cfg.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %s, want Asia/Shanghai", cfg.Timezone)
	}
	if cfg.Currency != "CNY" {
		t.Errorf("Currency = %s, want CNY", cfg.Currency)
	}
	if len(cfg.Features) != 1 || cfg.Features[0] != "basic" {
		t.Errorf("Features = %v, want [basic]", cfg.Features)
	}
}

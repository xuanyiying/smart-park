package multitenancy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// IsolationLevel defines the level of tenant isolation
type IsolationLevel int

const (
	// IsolationLevelShared shares database and schema among tenants
	IsolationLevelShared IsolationLevel = iota
	// IsolationLevelSchema uses separate schema per tenant
	IsolationLevelSchema
	// IsolationLevelDatabase uses separate database per tenant
	IsolationLevelDatabase
)

// TenantIsolation provides database isolation strategies
type TenantIsolation struct {
	level IsolationLevel
}

// NewTenantIsolation creates a new tenant isolation manager
func NewTenantIsolation(level IsolationLevel) *TenantIsolation {
	return &TenantIsolation{level: level}
}

// GetTableName returns the table name with tenant isolation
func (ti *TenantIsolation) GetTableName(tenantID uuid.UUID, baseTable string) string {
	switch ti.level {
	case IsolationLevelSchema:
		return fmt.Sprintf("tenant_%s.%s", tenantID.String(), baseTable)
	case IsolationLevelDatabase:
		// Database isolation is handled at connection level
		return baseTable
	default:
		// Shared: add tenant_id column filter
		return baseTable
	}
}

// GetConnectionString returns the database connection string for a tenant
func (ti *TenantIsolation) GetConnectionString(baseConnStr string, tenantID uuid.UUID) string {
	switch ti.level {
	case IsolationLevelDatabase:
		// Append tenant database name to connection string
		return fmt.Sprintf("%s_tenant_%s", baseConnStr, tenantID.String())
	default:
		return baseConnStr
	}
}

// AddTenantFilter adds tenant filter to queries for shared isolation
func AddTenantFilter(ctx context.Context, query string) string {
	tenantID := GetTenantID(ctx)
	if tenantID == nil {
		return query
	}
	// In production, this would use parameterized queries
	// This is a simplified example
	return fmt.Sprintf("%s AND tenant_id = '%s'", query, tenantID.String())
}

// TenantAwareDB wraps database operations with tenant context
type TenantAwareDB struct {
	isolation *TenantIsolation
}

// NewTenantAwareDB creates a new tenant-aware database wrapper
func NewTenantAwareDB(level IsolationLevel) *TenantAwareDB {
	return &TenantAwareDB{
		isolation: NewTenantIsolation(level),
	}
}

// WithTenantScope returns a context with tenant scope applied
func (db *TenantAwareDB) WithTenantScope(ctx context.Context, tenantID uuid.UUID) context.Context {
	return ContextWithTenant(ctx, &TenantInfo{
		ID: tenantID,
	})
}

// IsolationLevel returns the current isolation level
func (db *TenantAwareDB) IsolationLevel() IsolationLevel {
	return db.isolation.level
}

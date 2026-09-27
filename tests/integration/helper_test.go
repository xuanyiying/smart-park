//go:build integration

// Package integration_test runs business flows against a real PostgreSQL
// database. Execute with:
//
//	INTEGRATION=1 go test -tags=integration ./tests/integration
//
// Required infrastructure (defaults in parentheses):
//
//	DB_HOST (127.0.0.1), DB_PORT (5432), DB_USER (postgres),
//	DB_PASSWORD (postgres), DB_NAME (parking_test).
package integration_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	billingbiz "github.com/xuanyiying/smart-park/internal/billing/biz"
	billingdata "github.com/xuanyiying/smart-park/internal/billing/data"
	billingent "github.com/xuanyiying/smart-park/internal/billing/data/ent"
	paymentbiz "github.com/xuanyiying/smart-park/internal/payment/biz"
	paymentdata "github.com/xuanyiying/smart-park/internal/payment/data"
	paymentent "github.com/xuanyiying/smart-park/internal/payment/data/ent"
	chargingbiz "github.com/xuanyiying/smart-park/internal/charging/biz"
	chargingdata "github.com/xuanyiying/smart-park/internal/charging/data"
	chargingent "github.com/xuanyiying/smart-park/internal/charging/data/ent"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

func testLogger() log.Logger {
	return log.NewStdLogger(io.Discard)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func testDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		envOr("DB_HOST", "127.0.0.1"),
		envOr("DB_PORT", "5432"),
		envOr("DB_USER", "postgres"),
		envOr("DB_PASSWORD", "postgres"),
		envOr("DB_NAME", "parking_test"),
	)
}

// newChargingStack builds a ChargingUseCase backed by the real database, with
// tenant scoping applied exactly as cmd/charging/main.go does. The returned
// context carries a fresh per-test tenant so tests never collide.
func newChargingStack(t *testing.T) (*chargingbiz.ChargingUseCase, context.Context) {
	t.Helper()
	ctx := context.Background()

	client, err := chargingent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	tenant.ApplyTenantScoping(client, chargingdata.TenantScopes(), chargingdata.TenantTypes())

	dataLayer, cleanup, err := chargingdata.NewData(client, testLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	repo := chargingdata.NewChargingRepo(dataLayer)
	cfg := &chargingbiz.Config{
		DefaultServiceFee:     0.5,
		MaxSessionDuration:    24 * time.Hour,
		DefaultPeakLoadKWh:    7.0,
		DefaultOffPeakLoadKWh: 3.5,
	}
	uc := chargingbiz.NewChargingUseCaseWithConfig(repo, cfg, testLogger())
	return uc, withTestTenant(ctx)
}

func newBillingStack(t *testing.T) (*billingbiz.BillingUseCase, billingbiz.BillingRuleRepo, context.Context) {
	t.Helper()
	ctx := context.Background()

	client, err := billingent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	tenant.ApplyTenantScoping(client, billingdata.TenantScopes(), billingdata.TenantTypes())

	dataLayer, cleanup, err := billingdata.NewData(client, testLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	repo := billingdata.NewBillingRuleRepo(dataLayer)
	uc := billingbiz.NewBillingUseCase(repo, testLogger(), nil)
	return uc, repo, withTestTenant(ctx)
}

func newPaymentStack(t *testing.T) (paymentbiz.OrderRepo, context.Context) {
	t.Helper()
	ctx := context.Background()

	client, err := paymentent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	dataLayer, cleanup, err := paymentdata.NewData(client, testLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	return paymentdata.NewOrderRepo(dataLayer), ctx
}

func withTestTenant(ctx context.Context) context.Context {
	id := uuid.New()
	return tenant.WithTenant(ctx, &tenant.TenantInfo{
		ID:   id,
		Code: "itest-" + id.String()[:8],
		Name: "集成测试租户",
	})
}

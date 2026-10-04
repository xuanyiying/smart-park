//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	vehiclebiz "github.com/xuanyiying/smart-park/internal/vehicle/biz"
	vehicledata "github.com/xuanyiying/smart-park/internal/vehicle/data"
	vehicleent "github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/pkg/multitenancy"
)

func newVehicleStack(t *testing.T) (vehiclebiz.VehicleRepo, *vehiclebiz.RecordQueryUseCase, context.Context) {
	t.Helper()
	ctx := context.Background()

	client, err := vehicleent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))

	dataLayer, cleanup, err := vehicledata.NewData(client, testDSN(), testLogger())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	repo := vehicledata.NewVehicleRepo(dataLayer)
	tenantID := uuid.New()
	ctx = multitenancy.ContextWithTenant(ctx, &multitenancy.TenantInfo{ID: tenantID})
	return repo, vehiclebiz.NewRecordQueryUseCase(repo), ctx
}

// TestVehicleRecordStatusRoundTrip 回归断点 A：支付服务回写的 exit_status
// 必须真实落库，使出场时"已支付直接抬杆"路径可以命中。
func TestVehicleRecordStatusRoundTrip(t *testing.T) {
	repo, uc, ctx := newVehicleStack(t)

	recordID := uuid.New()
	plate := "京D99999"
	require.NoError(t, repo.CreateParkingRecord(ctx, &vehiclebiz.ParkingRecord{
		ID:          recordID,
		LotID:       uuid.New(),
		EntryLaneID: uuid.New(),
		EntryTime:   time.Now(),
		PlateNumber: &plate,
	}))

	// 初始为 unpaid
	got, err := repo.GetParkingRecord(ctx, recordID)
	require.NoError(t, err)
	require.Equal(t, "unpaid", got.ExitStatus)

	// 支付结算后回写 paid
	require.NoError(t, uc.UpdateRecordStatus(ctx, recordID.String(), "paid"))
	got, err = repo.GetParkingRecord(ctx, recordID)
	require.NoError(t, err)
	require.Equal(t, "paid", got.ExitStatus)

	// 重复回写幂等；退款状态可继续流转
	require.NoError(t, uc.UpdateRecordStatus(ctx, recordID.String(), "paid"))
	require.NoError(t, uc.UpdateRecordStatus(ctx, recordID.String(), "refunded"))
	got, err = repo.GetParkingRecord(ctx, recordID)
	require.NoError(t, err)
	require.Equal(t, "refunded", got.ExitStatus)

	// 非法状态拒绝
	require.Error(t, uc.UpdateRecordStatus(ctx, recordID.String(), "whatever"))
}

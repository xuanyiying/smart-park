package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// recordStatusMock overrides the generic mock's record methods so
// UpdateRecordStatus can be exercised end-to-end within the biz layer.
type recordStatusMock struct {
	*MockVehicleRepo
	records map[uuid.UUID]*ParkingRecord
	updates int
}

func (m *recordStatusMock) GetParkingRecord(ctx context.Context, recordID uuid.UUID) (*ParkingRecord, error) {
	if rec, ok := m.records[recordID]; ok {
		cp := *rec
		return &cp, nil
	}
	return nil, nil
}

func (m *recordStatusMock) UpdateParkingRecord(ctx context.Context, rec *ParkingRecord) error {
	m.updates++
	stored := *rec
	m.records[rec.ID] = &stored
	return nil
}

func TestRecordQueryUseCaseUpdateRecordStatus(t *testing.T) {
	store := &recordStatusMock{
		MockVehicleRepo: NewMockVehicleRepo(),
		records:         make(map[uuid.UUID]*ParkingRecord),
	}
	uc := NewRecordQueryUseCase(store)

	recordID := uuid.New()
	store.records[recordID] = &ParkingRecord{ID: recordID, ExitStatus: ExitStatusUnpaid}

	// 正常回写支付状态
	require.NoError(t, uc.UpdateRecordStatus(context.Background(), recordID.String(), ExitStatusPaid))
	require.Equal(t, ExitStatusPaid, store.records[recordID].ExitStatus)

	// 重复回写同一状态：幂等，不再产生更新
	updatesBefore := store.updates
	require.NoError(t, uc.UpdateRecordStatus(context.Background(), recordID.String(), ExitStatusPaid))
	require.Equal(t, updatesBefore, store.updates)

	// 非法状态被拒绝
	require.Error(t, uc.UpdateRecordStatus(context.Background(), recordID.String(), "hacked"))

	// 非法 UUID 被拒绝
	require.Error(t, uc.UpdateRecordStatus(context.Background(), "not-a-uuid", ExitStatusPaid))

	// 不存在的记录返回错误而非静默成功
	require.Error(t, uc.UpdateRecordStatus(context.Background(), uuid.New().String(), ExitStatusPaid))
}

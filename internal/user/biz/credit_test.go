package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockUserRepo is a mock implementation of UserRepo for testing
type MockUserRepo struct {
	users          map[string]*User
	paymentRecords map[string][]*PaymentRecord
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users:          make(map[string]*User),
		paymentRecords: make(map[string][]*PaymentRecord),
	}
}

func (m *MockUserRepo) GetUserByOpenID(ctx context.Context, openID string) (*User, error) {
	return nil, nil
}

func (m *MockUserRepo) CreateUser(ctx context.Context, user *User) error {
	m.users[user.ID.String()] = user
	return nil
}

func (m *MockUserRepo) GetUserByID(ctx context.Context, userID string) (*User, error) {
	user, exists := m.users[userID]
	if !exists {
		// Create a default user if not found
		user = &User{
			ID:            uuid.New(),
			OpenID:        "test-openid",
			CreditScore:   100,
			CreditLevel:   "A",
			AutoPayEnabled: false,
		}
		m.users[userID] = user
	}
	return user, nil
}

func (m *MockUserRepo) UpdateUser(ctx context.Context, user *User) error {
	m.users[user.ID.String()] = user
	return nil
}

func (m *MockUserRepo) BindVehicle(ctx context.Context, userVehicle *UserVehicle) error {
	return nil
}

func (m *MockUserRepo) UnbindVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) error {
	return nil
}

func (m *MockUserRepo) ListUserVehicles(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*UserVehicle, int64, error) {
	return nil, 0, nil
}

func (m *MockUserRepo) GetUserPaymentRecords(ctx context.Context, userID string) ([]*PaymentRecord, error) {
	return m.paymentRecords[userID], nil
}

func TestCreditManager_CalculateCreditScore(t *testing.T) {
	mockRepo := NewMockUserRepo()
	creditManager := NewCreditManager(mockRepo, nil)

	// Test case 1: New user with no payment records
	userID := uuid.New().String()
	score, err := creditManager.CalculateCreditScore(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, 100, score)

	// Test case 2: User with good payment history
	userID2 := uuid.New().String()
	mockRepo.paymentRecords[userID2] = []*PaymentRecord{
		{IsLate: false},
		{IsLate: false},
		{IsLate: false},
	}
	score, err = creditManager.CalculateCreditScore(context.Background(), userID2)
	assert.NoError(t, err)
	assert.Equal(t, 110, score) // 100 + 10 for good payment history

	// Test case 3: User with some late payments
	userID3 := uuid.New().String()
	mockRepo.paymentRecords[userID3] = []*PaymentRecord{
		{IsLate: false},
		{IsLate: true},
		{IsLate: false},
	}
	score, err = creditManager.CalculateCreditScore(context.Background(), userID3)
	assert.NoError(t, err)
	assert.Equal(t, 95, score) // 100 - 10 + 5 for average payment history

	// Test case 4: User with many late payments
	userID4 := uuid.New().String()
	mockRepo.paymentRecords[userID4] = []*PaymentRecord{
		{IsLate: true},
		{IsLate: true},
		{IsLate: true},
	}
	score, err = creditManager.CalculateCreditScore(context.Background(), userID4)
	assert.NoError(t, err)
	assert.Equal(t, 70, score) // 100 - 30
}

func TestCreditManager_CalculateCreditLevel(t *testing.T) {
	creditManager := NewCreditManager(nil, nil)

	// Test credit level calculation
	testCases := []struct {
		score int
		expectedLevel string
	}{
		{95, "A+"},
		{85, "A"},
		{75, "B+"},
		{65, "B"},
		{55, "C"},
		{45, "D"},
	}

	for _, tc := range testCases {
		level := creditManager.CalculateCreditLevel(tc.score)
		assert.Equal(t, tc.expectedLevel, level)
	}
}

func TestCreditManager_CheckCreditEligibility(t *testing.T) {
	mockRepo := NewMockUserRepo()
	creditManager := NewCreditManager(mockRepo, nil)

	// Test case 1: User with high credit score
	userID1 := uuid.New().String()
	user1 := &User{
		ID:          uuid.MustParse(userID1),
		CreditScore: 70,
	}
	mockRepo.users[userID1] = user1

	eligible, err := creditManager.CheckCreditEligibility(context.Background(), userID1)
	assert.NoError(t, err)
	assert.True(t, eligible)

	// Test case 2: User with low credit score
	userID2 := uuid.New().String()
	user2 := &User{
		ID:          uuid.MustParse(userID2),
		CreditScore: 50,
	}
	mockRepo.users[userID2] = user2

	eligible, err = creditManager.CheckCreditEligibility(context.Background(), userID2)
	assert.NoError(t, err)
	assert.False(t, eligible)
}

package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// MockBillingRuleRepo is a mock implementation of BillingRuleRepo for testing.
type MockBillingRuleRepo struct {
	Rules map[uuid.UUID]*BillingRule
}

func NewMockBillingRuleRepo() *MockBillingRuleRepo {
	return &MockBillingRuleRepo{
		Rules: make(map[uuid.UUID]*BillingRule),
	}
}

func (m *MockBillingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	var rules []*BillingRule
	for _, rule := range m.Rules {
		if rule.LotID == lotID {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (m *MockBillingRuleRepo) GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*BillingRule, error) {
	return m.Rules[ruleID], nil
}

func (m *MockBillingRuleRepo) CreateBillingRule(ctx context.Context, rule *BillingRule) error {
	m.Rules[rule.ID] = rule
	return nil
}

func (m *MockBillingRuleRepo) UpdateBillingRule(ctx context.Context, rule *BillingRule) error {
	m.Rules[rule.ID] = rule
	return nil
}

func (m *MockBillingRuleRepo) DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error {
	delete(m.Rules, ruleID)
	return nil
}

func (m *MockBillingRuleRepo) ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*BillingRule, int64, error) {
	rules, _ := m.GetRulesByLotID(ctx, lotID)
	return rules, int64(len(rules)), nil
}

func TestBillingUseCase_CalculateFee(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockBillingRuleRepo()

	lotID := uuid.New()

	// Add test billing rules
	mockRepo.Rules[uuid.New()] = &BillingRule{
		ID:       uuid.New(),
		LotID:    lotID,
		RuleName: "Standard Rate",
		RuleType: "time",
		Priority: 1,
		IsActive: true,
	}

	uc := NewBillingUseCase(mockRepo, logger)

	req := &v1.CalculateFeeRequest{
		RecordId:    uuid.New().String(),
		LotId:       lotID.String(),
		PlateNumber: "京A12345",
		EntryTime:   1700000000,
		ExitTime:    1700007200, // 2 hours later
		VehicleType: "temporary",
	}

	data, err := uc.CalculateFee(context.Background(), req)
	if err != nil {
		t.Fatalf("CalculateFee failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.BaseAmount <= 0 {
		t.Error("Expected positive base amount")
	}
}

func TestBillingUseCase_CreateBillingRule(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockBillingRuleRepo()

	uc := NewBillingUseCase(mockRepo, logger)

	req := &v1.CreateBillingRuleRequest{
		LotId:         uuid.New().String(),
		RuleName:      "VIP Discount",
		RuleType:      "vip",
		ConditionsJson: "{}",
		ActionsJson:    "{}",
		Priority:       10,
		IsActive:       true,
	}

	rule, err := uc.CreateBillingRule(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateBillingRule failed: %v", err)
	}

	if rule == nil {
		t.Fatal("Expected non-nil rule")
	}

	if rule.RuleName != "VIP Discount" {
		t.Errorf("Expected rule name 'VIP Discount', got %s", rule.RuleName)
	}
}

func TestBillingUseCase_GetBillingRules(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockBillingRuleRepo()

	lotID := uuid.New()

	// Add test rules
	mockRepo.Rules[uuid.New()] = &BillingRule{
		ID:       uuid.New(),
		LotID:    lotID,
		RuleName: "Rule 1",
		RuleType: "time",
		IsActive: true,
	}
	mockRepo.Rules[uuid.New()] = &BillingRule{
		ID:       uuid.New(),
		LotID:    lotID,
		RuleName: "Rule 2",
		RuleType: "monthly",
		IsActive: true,
	}

	uc := NewBillingUseCase(mockRepo, logger)

	req := &v1.GetBillingRulesRequest{
		LotId: lotID.String(),
	}

	rules, err := uc.GetBillingRules(context.Background(), req)
	if err != nil {
		t.Fatalf("GetBillingRules failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}
}

func TestBillingUseCase_DeleteBillingRule(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockBillingRuleRepo()

	ruleID := uuid.New()
	mockRepo.Rules[ruleID] = &BillingRule{
		ID:       ruleID,
		LotID:    uuid.New(),
		RuleName: "Test Rule",
	}

	uc := NewBillingUseCase(mockRepo, logger)

	req := &v1.DeleteBillingRuleRequest{
		Id: ruleID.String(),
	}

	err := uc.DeleteBillingRule(context.Background(), req)
	if err != nil {
		t.Fatalf("DeleteBillingRule failed: %v", err)
	}

	if _, exists := mockRepo.Rules[ruleID]; exists {
		t.Error("Expected rule to be deleted")
	}
}

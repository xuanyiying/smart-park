package biz

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// fakeBillingRuleRepo is an in-memory BillingRuleRepo for use-case tests.
type fakeBillingRuleRepo struct {
	rules      map[uuid.UUID]*BillingRule
	byLot      map[uuid.UUID][]*BillingRule
	createErr  error
	updateErr  error
	deleteErr  error
	getErr     error
	seedCalled bool
}

func newFakeBillingRuleRepo() *fakeBillingRuleRepo {
	return &fakeBillingRuleRepo{
		rules: make(map[uuid.UUID]*BillingRule),
		byLot: make(map[uuid.UUID][]*BillingRule),
	}
}

func (r *fakeBillingRuleRepo) add(rule *BillingRule) {
	r.rules[rule.ID] = rule
	r.byLot[rule.LotID] = append(r.byLot[rule.LotID], rule)
}

func (r *fakeBillingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	out := make([]*BillingRule, len(r.byLot[lotID]))
	copy(out, r.byLot[lotID])
	return out, nil
}

func (r *fakeBillingRuleRepo) GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*BillingRule, error) {
	return r.rules[ruleID], nil
}

func (r *fakeBillingRuleRepo) CreateBillingRule(ctx context.Context, rule *BillingRule) error {
	if r.createErr != nil {
		return r.createErr
	}
	rule.CreatedAt = time.Now()
	r.add(rule)
	return nil
}

func (r *fakeBillingRuleRepo) UpdateBillingRule(ctx context.Context, rule *BillingRule) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.rules[rule.ID] = rule
	return nil
}

func (r *fakeBillingRuleRepo) DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.rules, ruleID)
	return nil
}

func (r *fakeBillingRuleRepo) ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*BillingRule, int64, error) {
	rules := r.byLot[lotID]
	return rules, int64(len(rules)), nil
}

func (r *fakeBillingRuleRepo) SeedData(ctx context.Context, lotID uuid.UUID) error {
	r.seedCalled = true
	return nil
}

func (r *fakeBillingRuleRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newTestBillingUseCase(repo BillingRuleRepo, holidays ...string) *BillingUseCase {
	return NewBillingUseCase(repo, log.NewStdLogger(io.Discard), holidays)
}

func baseRule(lotID uuid.UUID, priority int) *BillingRule {
	return &BillingRule{
		ID:         uuid.New(),
		LotID:      lotID,
		RuleName:   "基础计费-小型车",
		RuleType:   "base",
		Conditions: `{"type":"vehicle_type","value":"small"}`,
		Actions:    `[{"type":"fixed","amount":5},{"type":"per_hour","amount":2},{"type":"max_daily","amount":50}]`,
		Priority:   priority,
		IsActive:   true,
	}
}

func feeRequest(lotID uuid.UUID, entry, exit time.Time, vehicleType string) *v1.CalculateFeeRequest {
	return &v1.CalculateFeeRequest{
		RecordId:    uuid.New().String(),
		LotId:       lotID.String(),
		PlateNumber: "京A12345",
		EntryTime:   entry.Unix(),
		ExitTime:    exit.Unix(),
		VehicleType: vehicleType,
	}
}

// TestCalculateFeeAppliesBaseRule drives the full use case: a small car parked for
// two hours must be charged 5 + 2*2 = 9 元 through the configured rule, not the
// hard-coded fallback (which the old seed/engine mismatch always produced).
func TestCalculateFeeAppliesBaseRule(t *testing.T) {
	lotID := uuid.New()
	repo := newFakeBillingRuleRepo()
	repo.add(baseRule(lotID, 1))
	uc := newTestBillingUseCase(repo)

	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	exit := entry.Add(2 * time.Hour)

	bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, exit, "small"))
	if err != nil {
		t.Fatalf("CalculateFee failed: %v", err)
	}
	if bill.FinalAmount != 900 {
		t.Errorf("final amount = %d cents, want 900 (5元 + 2小时×2元)", bill.FinalAmount)
	}
	if bill.BaseAmount != 900 {
		t.Errorf("base amount = %d cents, want 900", bill.BaseAmount)
	}
	if len(bill.AppliedRules) != 1 {
		t.Fatalf("applied rules = %d, want 1", len(bill.AppliedRules))
	}
	if bill.AppliedRules[0].Amount != 900 {
		t.Errorf("applied rule amount = %d cents, want 900", bill.AppliedRules[0].Amount)
	}
}

// TestCalculateFeeFallsBackToDefaultTariff pins the fallback behaviour when no
// rule matches: 1 小时内 5 元，超出 2 元/小时。
func TestCalculateFeeFallsBackToDefaultTariff(t *testing.T) {
	lotID := uuid.New()
	repo := newFakeBillingRuleRepo()
	repo.add(baseRule(lotID, 1)) // only matches "small"
	uc := newTestBillingUseCase(repo)

	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, entry.Add(30*time.Minute), "large"))
	if err != nil {
		t.Fatalf("CalculateFee failed: %v", err)
	}
	if bill.FinalAmount != 500 {
		t.Errorf("final amount = %d cents, want 500 (default tariff)", bill.FinalAmount)
	}
	if len(bill.AppliedRules) != 0 {
		t.Errorf("no rule should be applied for an unmatched vehicle type, got %d", len(bill.AppliedRules))
	}
}

// TestCalculateFeeDiscountAndOverride covers the rule-type switch: discounts
// accumulate, override replaces, and the monthly type exempts monthly vehicles.
func TestCalculateFeeDiscountAndOverride(t *testing.T) {
	lotID := uuid.New()
	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	exit := entry.Add(2 * time.Hour)

	t.Run("discount accumulates", func(t *testing.T) {
		repo := newFakeBillingRuleRepo()
		repo.add(baseRule(lotID, 1))
		repo.add(&BillingRule{
			ID: uuid.New(), LotID: lotID, RuleName: "夜间优惠", RuleType: "discount",
			Conditions: `{"type":"time_range","value":{"start":22,"end":6}}`,
			Actions:    `[{"type":"fixed","amount":2}]`,
			Priority:   2, IsActive: true,
		})
		// 入场 23:00，命中跨零点夜间优惠：900 - 200 = 700。
		nightEntry := time.Date(2026, 10, 1, 23, 0, 0, 0, time.Local)
		uc := newTestBillingUseCase(repo)
		bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, nightEntry, nightEntry.Add(2*time.Hour), "small"))
		if err != nil {
			t.Fatalf("CalculateFee failed: %v", err)
		}
		if bill.DiscountAmount != 200 {
			t.Errorf("discount = %d cents, want 200", bill.DiscountAmount)
		}
		if bill.FinalAmount != 700 {
			t.Errorf("final = %d cents, want 700", bill.FinalAmount)
		}
	})

	t.Run("override replaces", func(t *testing.T) {
		repo := newFakeBillingRuleRepo()
		repo.add(baseRule(lotID, 1))
		repo.add(&BillingRule{
			ID: uuid.New(), LotID: lotID, RuleName: "活动统一价", RuleType: "override",
			Conditions: `{"type":"vehicle_type","value":"small"}`,
			Actions:    `[{"type":"fixed","amount":3}]`,
			Priority:   9, IsActive: true,
		})
		uc := newTestBillingUseCase(repo)
		bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, exit, "small"))
		if err != nil {
			t.Fatalf("CalculateFee failed: %v", err)
		}
		if bill.FinalAmount != 300 {
			t.Errorf("override final = %d cents, want 300", bill.FinalAmount)
		}
		if bill.DiscountAmount != 0 {
			t.Errorf("override must clear discounts, got %d", bill.DiscountAmount)
		}
	})

	t.Run("monthly exempts", func(t *testing.T) {
		repo := newFakeBillingRuleRepo()
		repo.add(&BillingRule{
			ID: uuid.New(), LotID: lotID, RuleName: "月卡", RuleType: "monthly",
			Conditions: `{"type":"vehicle_type","value":"monthly"}`,
			Actions:    `[{"type":"fixed","amount":0}]`,
			Priority:   5, IsActive: true,
		})
		uc := newTestBillingUseCase(repo)
		bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, exit, "monthly"))
		if err != nil {
			t.Fatalf("CalculateFee failed: %v", err)
		}
		if bill.FinalAmount != 0 {
			t.Errorf("monthly vehicle must be exempt, got %d cents", bill.FinalAmount)
		}
	})

	t.Run("inactive rules are skipped", func(t *testing.T) {
		repo := newFakeBillingRuleRepo()
		inactive := baseRule(lotID, 1)
		inactive.IsActive = false
		repo.add(inactive)
		uc := newTestBillingUseCase(repo)
		bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, exit, "small"))
		if err != nil {
			t.Fatalf("CalculateFee failed: %v", err)
		}
		if len(bill.AppliedRules) != 0 {
			t.Errorf("inactive rules must not apply, got %d", len(bill.AppliedRules))
		}
	})
}

func TestCalculateFeeRejectsInvalidLotID(t *testing.T) {
	uc := newTestBillingUseCase(newFakeBillingRuleRepo())
	req := feeRequest(uuid.New(), time.Now(), time.Now(), "small")
	req.LotId = "not-a-uuid"
	if _, err := uc.CalculateFee(context.Background(), req); err == nil {
		t.Fatal("expected an error for a malformed lot id")
	}
}

func TestCalculateFeePropagatesRepoError(t *testing.T) {
	repo := newFakeBillingRuleRepo()
	repo.getErr = errors.New("db down")
	uc := newTestBillingUseCase(repo)

	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	if _, err := uc.CalculateFee(context.Background(), feeRequest(uuid.New(), entry, entry.Add(time.Hour), "small")); err == nil {
		t.Fatal("expected the repository error to surface")
	}
}

// TestCalculateFeeSkipsMalformedRules proves a rule whose JSON cannot be parsed is
// skipped (with a warning) instead of poisoning the whole calculation.
func TestCalculateFeeSkipsMalformedRules(t *testing.T) {
	lotID := uuid.New()
	repo := newFakeBillingRuleRepo()
	broken := baseRule(lotID, 1)
	broken.Conditions = `{"type":` // malformed
	repo.add(broken)
	repo.add(baseRule(lotID, 2))
	uc := newTestBillingUseCase(repo)

	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	bill, err := uc.CalculateFee(context.Background(), feeRequest(lotID, entry, entry.Add(2*time.Hour), "small"))
	if err != nil {
		t.Fatalf("CalculateFee failed: %v", err)
	}
	if len(bill.AppliedRules) != 1 {
		t.Errorf("only the well-formed rule should apply, got %d", len(bill.AppliedRules))
	}
}

// TestRuleCacheServesAndInvalidates covers the TTL cache: a second calculation
// must not hit the repository, and a rule mutation must invalidate immediately.
func TestRuleCacheServesAndInvalidates(t *testing.T) {
	lotID := uuid.New()
	repo := &countingBillingRuleRepo{fakeBillingRuleRepo: newFakeBillingRuleRepo()}
	repo.add(baseRule(lotID, 1))
	uc := newTestBillingUseCase(repo)

	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)
	req := feeRequest(lotID, entry, entry.Add(time.Hour), "small")

	if _, err := uc.CalculateFee(context.Background(), req); err != nil {
		t.Fatalf("first CalculateFee failed: %v", err)
	}
	first := repo.getCalls
	if _, err := uc.CalculateFee(context.Background(), req); err != nil {
		t.Fatalf("second CalculateFee failed: %v", err)
	}
	if repo.getCalls != first {
		t.Errorf("second calculation re-read the rules table: %d -> %d calls", first, repo.getCalls)
	}

	// A mutation must take effect immediately, not after the TTL.
	newRule := baseRule(lotID, 2)
	newRule.RuleName = "更新后的规则"
	if _, err := uc.CreateBillingRule(context.Background(), &v1.CreateBillingRuleRequest{
		LotId: lotID.String(), RuleName: newRule.RuleName, RuleType: "base",
		ConditionsJson: newRule.Conditions, ActionsJson: newRule.Actions,
		Priority: 2, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateBillingRule failed: %v", err)
	}
	if _, err := uc.CalculateFee(context.Background(), req); err != nil {
		t.Fatalf("CalculateFee after create failed: %v", err)
	}
	if repo.getCalls == first {
		t.Error("cache was not invalidated after a rule create")
	}
}

type countingBillingRuleRepo struct {
	*fakeBillingRuleRepo
	getCalls int
}

func (r *countingBillingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	r.getCalls++
	return r.fakeBillingRuleRepo.GetRulesByLotID(ctx, lotID)
}

func TestBillingRuleCRUD(t *testing.T) {
	lotID := uuid.New()
	repo := newFakeBillingRuleRepo()
	uc := newTestBillingUseCase(repo)

	created, err := uc.CreateBillingRule(context.Background(), &v1.CreateBillingRuleRequest{
		LotId: lotID.String(), RuleName: "规则A", RuleType: "base",
		ConditionsJson: `{"type":"vehicle_type","value":"small"}`,
		ActionsJson:    `[{"type":"fixed","amount":5}]`,
		Priority:       3, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateBillingRule failed: %v", err)
	}
	if created.Priority != 3 || !created.IsActive || created.RuleName != "规则A" {
		t.Errorf("created rule not mapped back correctly: %+v", created)
	}

	if err := uc.UpdateBillingRule(context.Background(), &v1.UpdateBillingRuleRequest{
		Id: created.Id, RuleName: "规则A-改", RuleType: "base",
		ConditionsJson: created.ConditionsJson, ActionsJson: created.ActionsJson,
		Priority: 7, IsActive: true,
	}); err != nil {
		t.Fatalf("UpdateBillingRule failed: %v", err)
	}
	if repo.rules[uuid.MustParse(created.Id)].Priority != 7 {
		t.Error("update did not persist")
	}

	rules, err := uc.GetBillingRules(context.Background(), &v1.GetBillingRulesRequest{LotId: lotID.String()})
	if err != nil {
		t.Fatalf("GetBillingRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("GetBillingRules returned %d rules, want 1", len(rules))
	}

	if err := uc.DeleteBillingRule(context.Background(), &v1.DeleteBillingRuleRequest{Id: created.Id}); err != nil {
		t.Fatalf("DeleteBillingRule failed: %v", err)
	}
	if len(repo.rules) != 0 {
		t.Error("delete did not remove the rule")
	}
}

// TestGetBillingRulesRejectsInvalidLotID locks in M8: a malformed lot id must
// return an error rather than a silently empty list that hides caller bugs.
func TestGetBillingRulesRejectsInvalidLotID(t *testing.T) {
	uc := newTestBillingUseCase(newFakeBillingRuleRepo())
	if _, err := uc.GetBillingRules(context.Background(), &v1.GetBillingRulesRequest{LotId: "nope"}); err == nil {
		t.Fatal("expected an error for an invalid lot id")
	}
}

func TestBillingRuleCRUDInvalidIDs(t *testing.T) {
	uc := newTestBillingUseCase(newFakeBillingRuleRepo())
	ctx := context.Background()

	if _, err := uc.CreateBillingRule(ctx, &v1.CreateBillingRuleRequest{LotId: "bad"}); err == nil {
		t.Error("create must reject an invalid lot id")
	}
	if err := uc.UpdateBillingRule(ctx, &v1.UpdateBillingRuleRequest{Id: "bad"}); err == nil {
		t.Error("update must reject an invalid rule id")
	}
	if err := uc.DeleteBillingRule(ctx, &v1.DeleteBillingRuleRequest{Id: "bad"}); err == nil {
		t.Error("delete must reject an invalid rule id")
	}
}

func TestBillingRuleCRUDRepoErrors(t *testing.T) {
	repo := newFakeBillingRuleRepo()
	repo.createErr = errors.New("create failed")
	repo.updateErr = errors.New("update failed")
	repo.deleteErr = errors.New("delete failed")
	uc := newTestBillingUseCase(repo)
	ctx := context.Background()
	lotID := uuid.New()

	if _, err := uc.CreateBillingRule(ctx, &v1.CreateBillingRuleRequest{LotId: lotID.String(), RuleName: "x", RuleType: "base"}); err == nil {
		t.Error("expected the create error to surface")
	}
	if err := uc.UpdateBillingRule(ctx, &v1.UpdateBillingRuleRequest{Id: uuid.New().String()}); err == nil {
		t.Error("expected the update error to surface")
	}
	if err := uc.DeleteBillingRule(ctx, &v1.DeleteBillingRuleRequest{Id: uuid.New().String()}); err == nil {
		t.Error("expected the delete error to surface")
	}
}

// TestTestBillingRule covers the dry-run endpoint operators use to preview a rule.
func TestTestBillingRule(t *testing.T) {
	uc := newTestBillingUseCase(newFakeBillingRuleRepo())
	entry := time.Date(2026, 10, 1, 10, 0, 0, 0, time.Local)

	resp, err := uc.TestBillingRule(context.Background(), &v1.TestBillingRuleRequest{
		VehicleType:    "small",
		EntryTime:      entry.Unix(),
		ExitTime:       entry.Add(2 * time.Hour).Unix(),
		IsHoliday:      false,
		ConditionsJson: `{"type":"vehicle_type","value":"small"}`,
		ActionsJson:    `[{"type":"fixed","amount":5},{"type":"per_hour","amount":2}]`,
	})
	if err != nil {
		t.Fatalf("TestBillingRule failed: %v", err)
	}
	if !resp.ConditionMet {
		t.Error("condition should be met")
	}
	if resp.CalculatedFee != 900 {
		t.Errorf("fee = %d cents, want 900", resp.CalculatedFee)
	}
	if len(resp.AppliedActions) != 2 {
		t.Errorf("applied actions = %v, want 2 entries", resp.AppliedActions)
	}
	if resp.Duration != 2*3600 {
		t.Errorf("duration = %v seconds, want 7200", resp.Duration)
	}

	// Unmet condition: no actions applied, zero fee, no error.
	resp, err = uc.TestBillingRule(context.Background(), &v1.TestBillingRuleRequest{
		VehicleType:    "large",
		EntryTime:      entry.Unix(),
		ExitTime:       entry.Add(time.Hour).Unix(),
		ConditionsJson: `{"type":"vehicle_type","value":"small"}`,
		ActionsJson:    `[{"type":"fixed","amount":5}]`,
	})
	if err != nil {
		t.Fatalf("TestBillingRule failed: %v", err)
	}
	if resp.ConditionMet || resp.CalculatedFee != 0 || len(resp.AppliedActions) != 0 {
		t.Errorf("unmet condition must produce zero: %+v", resp)
	}

	// Malformed JSON must error, not silently succeed.
	if _, err := uc.TestBillingRule(context.Background(), &v1.TestBillingRuleRequest{
		ConditionsJson: `{"type":`,
	}); err == nil {
		t.Error("malformed conditions must error")
	}
	if _, err := uc.TestBillingRule(context.Background(), &v1.TestBillingRuleRequest{
		VehicleType:    "small",
		ConditionsJson: `{"type":"vehicle_type","value":"small"}`,
		ActionsJson:    `[{"type":`,
	}); err == nil {
		t.Error("malformed actions must error")
	}
}

// TestNewBillingUseCaseInvalidHolidays documents that a broken holiday config
// disables holiday conditions (with a warning) instead of failing startup.
func TestNewBillingUseCaseInvalidHolidays(t *testing.T) {
	uc := NewBillingUseCase(newFakeBillingRuleRepo(), log.NewStdLogger(io.Discard), []string{"not-a-date"})
	if uc.holidays.IsHoliday(time.Now()) {
		t.Error("an invalid calendar must never report a holiday")
	}
}

func TestSortRulesByPriorityDescending(t *testing.T) {
	lotID := uuid.New()
	low := baseRule(lotID, 1)
	high := baseRule(lotID, 9)
	mid := baseRule(lotID, 5)
	rules := []*BillingRule{low, high, mid}
	sortRulesByPriority(rules)
	if rules[0] != high || rules[1] != mid || rules[2] != low {
		t.Error("rules must be ordered by descending priority")
	}
}

func TestToFloatCoercions(t *testing.T) {
	cases := map[interface{}]float64{
		float64(1.5): 1.5,
		float32(2.5): 2.5,
		int(3):       3,
		int32(4):     4,
		int64(5):     5,
	}
	for in, want := range cases {
		got, ok := toFloat(in)
		if !ok || got != want {
			t.Errorf("toFloat(%v) = %v, %v; want %v, true", in, got, ok, want)
		}
	}
	if _, ok := toFloat("nope"); ok {
		t.Error("toFloat must reject unsupported types")
	}
	if _, ok := toFloat(nil); ok {
		t.Error("toFloat must reject nil")
	}
}

func TestYuanToCents(t *testing.T) {
	cases := map[float64]int64{
		0:     0,
		5:     500,
		9.99:  999,
		12.34: 1234,
		0.01:  1,
		2.675: 268, // math.Round 半远离奇进偶此处为 267.5 → 268
	}
	for yuan, want := range cases {
		if got := yuanToCents(yuan); got != want {
			t.Errorf("yuanToCents(%v) = %d, want %d", yuan, got, want)
		}
	}
}

func TestNewLogger(t *testing.T) {
	if NewLogger(log.NewStdLogger(io.Discard)) == nil {
		t.Error("NewLogger must return a helper")
	}
}

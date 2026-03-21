// Package biz provides business logic for the billing service.
package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// BillingRule represents a billing rule entity.
type BillingRule struct {
	ID         uuid.UUID
	LotID      uuid.UUID
	RuleName   string
	RuleType   string
	Conditions string
	Actions    string
	RuleConfig map[string]interface{}
	Priority   int
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BillingRuleRepo defines the repository interface for billing rule operations.
type BillingRuleRepo interface {
	GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error)
	GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*BillingRule, error)
	CreateBillingRule(ctx context.Context, rule *BillingRule) error
	UpdateBillingRule(ctx context.Context, rule *BillingRule) error
	DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error
	ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*BillingRule, int64, error)
}

// BillingUseCase implements billing business logic.
type BillingUseCase struct {
	repo BillingRuleRepo
	log  *log.Helper
}

// NewBillingUseCase creates a new BillingUseCase.
func NewBillingUseCase(repo BillingRuleRepo, logger log.Logger) *BillingUseCase {
	return &BillingUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CalculateFee calculates the parking fee.
func (uc *BillingUseCase) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.BillData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	// Get billing rules
	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get billing rules: %v", err)
		return nil, err
	}

	entryTime := time.Unix(req.EntryTime, 0)
	exitTime := time.Unix(req.ExitTime, 0)
	duration := exitTime.Sub(entryTime)

	// Calculate base amount (default: 2 yuan per hour)
	hours := duration.Hours()
	baseAmount := hours * 2

	// Apply rules
	var discountAmount float64
	var appliedRules []*v1.AppliedRule

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		// Apply rule logic based on type
		switch rule.RuleType {
		case "time":
			// Time-based pricing
			amount := applyTimeRule(rule, hours)
			if amount > 0 {
				appliedRules = append(appliedRules, &v1.AppliedRule{
					RuleId:   rule.ID.String(),
					RuleName: rule.RuleName,
					Amount:   amount,
				})
				baseAmount = amount
			}
		case "vip":
			if req.VehicleType == "vip" {
				discount := baseAmount * 0.5
				discountAmount += discount
				appliedRules = append(appliedRules, &v1.AppliedRule{
					RuleId:   rule.ID.String(),
					RuleName: rule.RuleName,
					Amount:   -discount,
				})
			}
		case "monthly":
			if req.VehicleType == "monthly" {
				discountAmount = baseAmount
				appliedRules = append(appliedRules, &v1.AppliedRule{
					RuleId:   rule.ID.String(),
					RuleName: rule.RuleName,
					Amount:   -discountAmount,
				})
			}
		}
	}

	finalAmount := baseAmount - discountAmount
	if finalAmount < 0 {
		finalAmount = 0
	}

	return &v1.BillData{
		RecordId:       req.RecordId,
		BaseAmount:     baseAmount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		AppliedRules:   appliedRules,
	}, nil
}

// applyTimeRule applies a time-based billing rule.
func applyTimeRule(rule *BillingRule, hours float64) float64 {
	// Simplified time rule calculation
	// In production, parse Conditions and Actions JSON
	if hours < 1 {
		return 5 // Minimum charge
	}
	return hours * 2
}

// CreateBillingRule creates a new billing rule.
func (uc *BillingUseCase) CreateBillingRule(ctx context.Context, req *v1.CreateBillingRuleRequest) (*v1.BillingRule, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	rule := &BillingRule{
		ID:         uuid.New(),
		LotID:      lotID,
		RuleName:   req.RuleName,
		RuleType:   req.RuleType,
		Conditions: req.ConditionsJson,
		Actions:    req.ActionsJson,
		Priority:   int(req.Priority),
		IsActive:   req.IsActive,
	}

	if err := uc.repo.CreateBillingRule(ctx, rule); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create billing rule: %v", err)
		return nil, err
	}

	return &v1.BillingRule{
		Id:            rule.ID.String(),
		LotId:         rule.LotID.String(),
		RuleName:      rule.RuleName,
		RuleType:      rule.RuleType,
		ConditionsJson: rule.Conditions,
		ActionsJson:    rule.Actions,
		Priority:       int32(rule.Priority),
		IsActive:       rule.IsActive,
		CreatedAt:      rule.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateBillingRule updates a billing rule.
func (uc *BillingUseCase) UpdateBillingRule(ctx context.Context, req *v1.UpdateBillingRuleRequest) error {
	ruleID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	rule := &BillingRule{
		ID:         ruleID,
		RuleName:   req.RuleName,
		RuleType:   req.RuleType,
		Conditions: req.ConditionsJson,
		Actions:    req.ActionsJson,
		Priority:   int(req.Priority),
		IsActive:   req.IsActive,
	}

	return uc.repo.UpdateBillingRule(ctx, rule)
}

// DeleteBillingRule deletes a billing rule.
func (uc *BillingUseCase) DeleteBillingRule(ctx context.Context, req *v1.DeleteBillingRuleRequest) error {
	ruleID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	return uc.repo.DeleteBillingRule(ctx, ruleID)
}

// GetBillingRules retrieves billing rules for a parking lot.
func (uc *BillingUseCase) GetBillingRules(ctx context.Context, req *v1.GetBillingRulesRequest) ([]*v1.BillingRule, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		return nil, err
	}

	var result []*v1.BillingRule
	for _, rule := range rules {
		result = append(result, &v1.BillingRule{
			Id:            rule.ID.String(),
			LotId:         rule.LotID.String(),
			RuleName:      rule.RuleName,
			RuleType:      rule.RuleType,
			ConditionsJson: rule.Conditions,
			ActionsJson:    rule.Actions,
			Priority:       int32(rule.Priority),
			IsActive:       rule.IsActive,
			CreatedAt:      rule.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

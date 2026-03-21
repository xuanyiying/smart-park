// Package service provides gRPC service implementation for the billing service.
package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
)

// BillingService implements the BillingService gRPC service.
type BillingService struct {
	v1.UnimplementedBillingServiceServer

	uc  *biz.BillingUseCase
	log *log.Helper
}

// NewBillingService creates a new BillingService.
func NewBillingService(uc *biz.BillingUseCase, logger log.Logger) *BillingService {
	return &BillingService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CalculateFee handles calculate fee request.
func (s *BillingService) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.CalculateFeeResponse, error) {
	data, err := s.uc.CalculateFee(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CalculateFee failed: %v", err)
		return &v1.CalculateFeeResponse{
			Code:    500,
			Message: "计算费用失败",
		}, nil
	}

	return &v1.CalculateFeeResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// CreateBillingRule handles create billing rule request.
func (s *BillingService) CreateBillingRule(ctx context.Context, req *v1.CreateBillingRuleRequest) (*v1.CreateBillingRuleResponse, error) {
	rule, err := s.uc.CreateBillingRule(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateBillingRule failed: %v", err)
		return &v1.CreateBillingRuleResponse{
			Code:    500,
			Message: "创建计费规则失败",
		}, nil
	}

	return &v1.CreateBillingRuleResponse{
		Code:    0,
		Message: "success",
		Data:    rule,
	}, nil
}

// UpdateBillingRule handles update billing rule request.
func (s *BillingService) UpdateBillingRule(ctx context.Context, req *v1.UpdateBillingRuleRequest) (*v1.UpdateBillingRuleResponse, error) {
	if err := s.uc.UpdateBillingRule(ctx, req); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateBillingRule failed: %v", err)
		return &v1.UpdateBillingRuleResponse{
			Code:    500,
			Message: "更新计费规则失败",
		}, nil
	}

	return &v1.UpdateBillingRuleResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// DeleteBillingRule handles delete billing rule request.
func (s *BillingService) DeleteBillingRule(ctx context.Context, req *v1.DeleteBillingRuleRequest) (*v1.DeleteBillingRuleResponse, error) {
	if err := s.uc.DeleteBillingRule(ctx, req); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteBillingRule failed: %v", err)
		return &v1.DeleteBillingRuleResponse{
			Code:    500,
			Message: "删除计费规则失败",
		}, nil
	}

	return &v1.DeleteBillingRuleResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// GetBillingRules handles get billing rules request.
func (s *BillingService) GetBillingRules(ctx context.Context, req *v1.GetBillingRulesRequest) (*v1.GetBillingRulesResponse, error) {
	rules, err := s.uc.GetBillingRules(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetBillingRules failed: %v", err)
		return &v1.GetBillingRulesResponse{
			Code:    500,
			Message: "获取计费规则失败",
		}, nil
	}

	return &v1.GetBillingRulesResponse{
		Code:    0,
		Message: "success",
		Data:    rules,
	}, nil
}

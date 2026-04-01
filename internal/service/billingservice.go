
package service

import(
	"context"

	pb "github.com/xuanyiying/smart-park/api/billing/v1"
)

type BillingServiceService struct {
	pb.UnimplementedBillingServiceServer
}

func NewBillingServiceService() pb.BillingServiceServer {
	return &BillingServiceService{}
}

func (s *BillingServiceService) CalculateFee(ctx context.Context, req *pb.CalculateFeeRequest) (*pb.CalculateFeeResponse, error) {
	return &pb.CalculateFeeResponse{}, nil
}
func (s *BillingServiceService) CreateBillingRule(ctx context.Context, req *pb.CreateBillingRuleRequest) (*pb.CreateBillingRuleResponse, error) {
	return &pb.CreateBillingRuleResponse{}, nil
}
func (s *BillingServiceService) UpdateBillingRule(ctx context.Context, req *pb.UpdateBillingRuleRequest) (*pb.UpdateBillingRuleResponse, error) {
	return &pb.UpdateBillingRuleResponse{}, nil
}
func (s *BillingServiceService) DeleteBillingRule(ctx context.Context, req *pb.DeleteBillingRuleRequest) (*pb.DeleteBillingRuleResponse, error) {
	return &pb.DeleteBillingRuleResponse{}, nil
}
func (s *BillingServiceService) GetBillingRules(ctx context.Context, req *pb.GetBillingRulesRequest) (*pb.GetBillingRulesResponse, error) {
	return &pb.GetBillingRulesResponse{}, nil
}
func (s *BillingServiceService) AdjustPrice(ctx context.Context, req *pb.AdjustPriceRequest) (*pb.AdjustPriceResponse, error) {
	return &pb.AdjustPriceResponse{}, nil
}

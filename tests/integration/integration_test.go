package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/pkg/cache"
)

type IntegrationTestSuite struct {
	suite.Suite
	vehicleRepo  biz.VehicleRepo
	billingRepo  biz.BillingRepo
	paymentRepo  biz.OrderRepo
	vehicleCache *cache.VehicleCache
	billingCache *cache.BillingCache
}

func (s *IntegrationTestSuite) SetupSuite() {
	// 初始化测试环境
	// 实际实现中需要连接测试数据库和 Redis
}

func (s *IntegrationTestSuite) TearDownSuite() {
	// 清理测试环境
}

func (s *IntegrationTestSuite) TestVehicleEntryAndExit() {
	ctx := context.Background()
	lotID := uuid.New()
	plateNumber := "京A12345"

	// 测试车辆入场
	entryReq := &biz.EntryRequest{
		LotID:       lotID,
		PlateNumber: plateNumber,
		EntryTime:   time.Now(),
		DeviceID:    "test-device-001",
	}

	vehicle, err := s.vehicleRepo.CreateVehicle(ctx, &biz.Vehicle{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		LotID:       lotID,
		EntryTime:   entryReq.EntryTime,
		Status:      "parked",
	})

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), vehicle)
	assert.Equal(s.T(), plateNumber, vehicle.PlateNumber)

	// 测试缓存
	cachedVehicle, err := s.vehicleCache.GetVehicle(ctx, plateNumber)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), vehicle.ID, cachedVehicle.ID)

	// 测试车辆出场
	exitTime := time.Now().Add(2 * time.Hour)
	exitVehicle, err := s.vehicleRepo.UpdateVehicle(ctx, &biz.Vehicle{
		ID:        vehicle.ID,
		ExitTime:  &exitTime,
		Status:    "exited",
	})

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), exitVehicle.ExitTime)

	// 测试缓存失效
	err = s.vehicleCache.InvalidateVehicle(ctx, plateNumber)
	assert.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestBillingCalculation() {
	ctx := context.Background()
	lotID := uuid.New()

	// 创建计费规则
	rule := &biz.BillingRule{
		ID:           uuid.New(),
		LotID:        lotID,
		RuleType:     "time",
		BaseFee:      5.0,
		BaseDuration: 30,
		UnitFee:      2.0,
		UnitDuration: 30,
		MaxFee:       100.0,
		FreeDuration: 15,
	}

	err := s.billingRepo.CreateBillingRule(ctx, rule)
	assert.NoError(s.T(), err)

	// 测试缓存
	cachedRule, err := s.billingCache.GetOrLoadBillingRule(ctx, lotID.String(), "time", func() (*biz.BillingRule, error) {
		return s.billingRepo.GetBillingRule(ctx, lotID, "time")
	})

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), rule.ID, cachedRule.ID)

	// 测试计费计算
	entryTime := time.Now()
	exitTime := entryTime.Add(90 * time.Minute)

	amount, err := calculateFee(rule, entryTime, exitTime)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 9.0, amount)
}

func (s *IntegrationTestSuite) TestPaymentFlow() {
	ctx := context.Background()
	recordID := uuid.New()

	// 创建支付订单
	order := &biz.Order{
		ID:          uuid.New(),
		RecordID:    recordID,
		Amount:      15.0,
		FinalAmount: 15.0,
		Status:      "pending",
	}

	err := s.paymentRepo.CreateOrder(ctx, order)
	assert.NoError(s.T(), err)

	// 模拟支付成功
	now := time.Now()
	order.Status = "paid"
	order.PayTime = &now
	order.PayMethod = "wechat"
	order.TransactionID = "test_transaction_123"
	order.PaidAmount = 15.0

	err = s.paymentRepo.UpdateOrder(ctx, order)
	assert.NoError(s.T(), err)

	// 查询订单
	savedOrder, err := s.paymentRepo.GetOrder(ctx, order.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "paid", savedOrder.Status)
	assert.Equal(s.T(), "wechat", savedOrder.PayMethod)
}

func (s *IntegrationTestSuite) TestCacheInvalidation() {
	ctx := context.Background()
	plateNumber := "京B67890"

	// 创建车辆并缓存
	vehicle := &biz.Vehicle{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		LotID:       uuid.New(),
		Status:      "parked",
	}

	err := s.vehicleCache.SetVehicle(ctx, vehicle)
	assert.NoError(s.T(), err)

	// 验证缓存存在
	cached, err := s.vehicleCache.GetVehicle(ctx, plateNumber)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), vehicle.ID, cached.ID)

	// 失效缓存
	err = s.vehicleCache.InvalidateVehicle(ctx, plateNumber)
	assert.NoError(s.T(), err)

	// 验证缓存已失效
	_, err = s.vehicleCache.GetVehicle(ctx, plateNumber)
	assert.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestConcurrentVehicleEntry() {
	ctx := context.Background()
	lotID := uuid.New()
	plateNumber := "京C11111"

	// 模拟并发入场
	done := make(chan bool, 2)

	go func() {
		acquired, err := s.vehicleCache.AcquireVehicleLock(ctx, plateNumber)
		if assert.NoError(s.T(), err) && acquired {
			defer s.vehicleCache.ReleaseVehicleLock(ctx, plateNumber)
			time.Sleep(100 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		acquired, err := s.vehicleCache.AcquireVehicleLock(ctx, plateNumber)
		if assert.NoError(s.T(), err) && acquired {
			defer s.vehicleCache.ReleaseVehicleLock(ctx, plateNumber)
			time.Sleep(100 * time.Millisecond)
		}
		done <- true
	}()

	<-done
	<-done
}

func calculateFee(rule *biz.BillingRule, entryTime, exitTime time.Time) (float64, error) {
	duration := exitTime.Sub(entryTime).Minutes()

	if duration <= float64(rule.FreeDuration) {
		return 0, nil
	}

	duration -= float64(rule.FreeDuration)

	if duration <= float64(rule.BaseDuration) {
		return rule.BaseFee, nil
	}

	duration -= float64(rule.BaseDuration)
	units := int(duration / float64(rule.UnitDuration))
	if duration/float64(rule.UnitDuration) > float64(units) {
		units++
	}

	total := rule.BaseFee + float64(units)*rule.UnitFee

	if total > rule.MaxFee {
		return rule.MaxFee, nil
	}

	return total, nil
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

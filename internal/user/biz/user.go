package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/user/v1"
	"github.com/xuanyiying/smart-park/internal/user/client/payment"
	"github.com/xuanyiying/smart-park/internal/user/client/vehicle"
	"github.com/xuanyiying/smart-park/internal/user/wechat"
	"github.com/xuanyiying/smart-park/pkg/auth"
)

type User struct {
	ID        uuid.UUID
	OpenID    string
	Nickname  string
	Avatar    string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserVehicle struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	PlateNumber string
	OwnerName   string
	OwnerPhone  string
	CreatedAt   time.Time
}

type UserRepo interface {
	GetUserByOpenID(ctx context.Context, openID string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
	BindVehicle(ctx context.Context, userVehicle *UserVehicle) error
	UnbindVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) error
	ListUserVehicles(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*UserVehicle, int64, error)
}

type UserUseCase struct {
	userRepo      UserRepo
	vehicleClient vehicle.Client
	paymentClient payment.Client
	jwtManager    *auth.JWTManager
	wechatClient  *wechat.Client
	log           *log.Helper
}

func NewUserUseCase(userRepo UserRepo, vehicleClient vehicle.Client, paymentClient payment.Client, jwtManager *auth.JWTManager, wechatClient *wechat.Client, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		userRepo:      userRepo,
		vehicleClient: vehicleClient,
		paymentClient: paymentClient,
		jwtManager:    jwtManager,
		wechatClient:  wechatClient,
		log:           log.NewHelper(logger),
	}
}

func (uc *UserUseCase) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginData, error) {
	openID, err := uc.getOpenIDFromWechat(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	user, err := uc.userRepo.GetUserByOpenID(ctx, openID)
	if err != nil {
		user = &User{
			ID:     uuid.New(),
			OpenID: openID,
		}
		if err := uc.userRepo.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	token, err := uc.jwtManager.GenerateToken(user.ID.String(), user.OpenID)
	if err != nil {
		return nil, err
	}

	return &v1.LoginData{
		Token:     token,
		OpenId:    openID,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

func (uc *UserUseCase) getOpenIDFromWechat(ctx context.Context, code string) (string, error) {
	if uc.wechatClient != nil {
		openID, err := uc.wechatClient.GetOpenID(ctx, code)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("failed to get openid from wechat: %v", err)
			return "", err
		}
		return openID, nil
	}

	uc.log.WithContext(ctx).Warn("wechat client not configured, using mock openid")
	return "mock_openid_" + code, nil
}

func (uc *UserUseCase) BindPlate(ctx context.Context, userID string, req *v1.BindPlateRequest) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	userVehicle := &UserVehicle{
		ID:          uuid.New(),
		UserID:      uid,
		PlateNumber: req.PlateNumber,
		OwnerName:   req.OwnerName,
		OwnerPhone:  req.OwnerPhone,
	}

	return uc.userRepo.BindVehicle(ctx, userVehicle)
}

func (uc *UserUseCase) UnbindPlate(ctx context.Context, userID string, plateNumber string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	return uc.userRepo.UnbindVehicle(ctx, uid, plateNumber)
}

func (uc *UserUseCase) ListPlates(ctx context.Context, userID string, page, pageSize int) (*v1.ListPlatesData, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	vehicles, total, err := uc.userRepo.ListUserVehicles(ctx, uid, page, pageSize)
	if err != nil {
		return nil, err
	}

	var plates []*v1.PlateInfo
	for _, v := range vehicles {
		plates = append(plates, &v1.PlateInfo{
			PlateNumber: v.PlateNumber,
			OwnerName:   v.OwnerName,
			OwnerPhone:  v.OwnerPhone,
		})
	}

	return &v1.ListPlatesData{
		Plates: plates,
		Total:  int32(total),
	}, nil
}

func (uc *UserUseCase) ListParkingRecords(ctx context.Context, userID string, page, pageSize int32) (*v1.ListParkingRecordsData, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// 1. 获取用户绑定的车牌
	vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uid, 1, 100)
	if err != nil {
		return nil, err
	}

	if len(vehicles) == 0 {
		return &v1.ListParkingRecordsData{
			Records: []*v1.ParkingRecordInfo{},
			Total:   0,
		}, nil
	}

	// 2. 提取车牌号
	var plateNumbers []string
	for _, v := range vehicles {
		plateNumbers = append(plateNumbers, v.PlateNumber)
	}

	// 3. 调用 vehicle service 查询停车记录
	vehicleData, err := uc.vehicleClient.ListParkingRecords(ctx, plateNumbers, page, pageSize)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to list parking records from vehicle service: %v", err)
		return nil, err
	}

	// 4. 转换为 user service 的响应格式
	var records []*v1.ParkingRecordInfo
	for _, r := range vehicleData.Records {
		// 计算金额（这里简化处理，实际应该从订单获取）
		var amount float64
		if r.ExitStatus == "paid" {
			amount = 0 // 已支付，显示0或实际金额
		}

		records = append(records, &v1.ParkingRecordInfo{
			RecordId:    r.RecordId,
			PlateNumber: r.PlateNumber,
			LotName:     "", // 需要从lot_id查询名称
			EntryTime:   r.EntryTime,
			ExitTime:    r.ExitTime,
			Duration:    r.ParkingDuration,
			Amount:      amount,
			Status:      r.RecordStatus,
		})
	}

	return &v1.ListParkingRecordsData{
		Records: records,
		Total:   vehicleData.Total,
	}, nil
}

func (uc *UserUseCase) GetParkingRecord(ctx context.Context, userID, recordID string) (*v1.ParkingRecordInfo, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// 1. 获取用户绑定的车牌
	vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uid, 1, 100)
	if err != nil {
		return nil, err
	}

	// 2. 调用 vehicle service 查询记录
	record, err := uc.vehicleClient.GetParkingRecord(ctx, recordID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get parking record from vehicle service: %v", err)
		return nil, err
	}

	// 3. 验证记录是否属于该用户的车牌
	userPlateMap := make(map[string]bool)
	for _, v := range vehicles {
		userPlateMap[v.PlateNumber] = true
	}

	if !userPlateMap[record.PlateNumber] {
		return nil, fmt.Errorf("record not found or access denied")
	}

	// 计算金额（这里简化处理，实际应该从订单获取）
	var amount float64
	if record.ExitStatus == "paid" {
		amount = 0 // 已支付，显示0或实际金额
	}

	return &v1.ParkingRecordInfo{
		RecordId:    record.RecordId,
		PlateNumber: record.PlateNumber,
		LotName:     "", // 需要从lot_id查询名称
		EntryTime:   record.EntryTime,
		ExitTime:    record.ExitTime,
		Duration:    record.ParkingDuration,
		Amount:      amount,
		Status:      record.RecordStatus,
	}, nil
}

func (uc *UserUseCase) ScanPay(ctx context.Context, userID string, req *v1.ScanPayRequest) (*v1.ScanPayData, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uid, 1, 100)
	if err != nil {
		return nil, err
	}

	plateMap := make(map[string]bool)
	for _, v := range vehicles {
		plateMap[v.PlateNumber] = true
	}

	record, err := uc.vehicleClient.GetParkingRecord(ctx, req.RecordId)
	if err != nil {
		return nil, fmt.Errorf("failed to get parking record: %w", err)
	}

	if !plateMap[record.PlateNumber] {
		return nil, fmt.Errorf("record not found or access denied")
	}

	payData, err := uc.paymentClient.CreatePayment(ctx, req.RecordId, 0, req.PayMethod, req.OpenId)
	if err != nil {
		return nil, err
	}

	return &v1.ScanPayData{
		OrderId: payData.OrderId,
		Amount:  payData.Amount,
		PayUrl:  payData.PayUrl,
		QrCode:  payData.QrCode,
	}, nil
}

func (uc *UserUseCase) GetMonthlyCard(ctx context.Context, userID, plateNumber string) (*v1.MonthlyCardInfo, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uid, 1, 100)
	if err != nil {
		return nil, err
	}

	plateMap := make(map[string]bool)
	for _, v := range vehicles {
		plateMap[v.PlateNumber] = true
	}

	if !plateMap[plateNumber] {
		return nil, fmt.Errorf("plate number not bound to user")
	}

	vehicleInfo, err := uc.vehicleClient.GetVehicleInfo(ctx, plateNumber)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get vehicle info: %v", err)
		return &v1.MonthlyCardInfo{
			PlateNumber:   plateNumber,
			IsValid:       false,
			DaysRemaining: 0,
		}, nil
	}

	var isValid bool
	var daysRemaining int32
	var validUntil string

	if vehicleInfo.MonthlyValidUntil != "" {
		validUntilTime, err := time.Parse("2006-01-02", vehicleInfo.MonthlyValidUntil)
		if err == nil {
			validUntil = vehicleInfo.MonthlyValidUntil
			now := time.Now()
			if validUntilTime.After(now) {
				isValid = true
				daysRemaining = int32(validUntilTime.Sub(now).Hours() / 24)
			}
		}
	}

	return &v1.MonthlyCardInfo{
		PlateNumber:   plateNumber,
		ValidUntil:    validUntil,
		DaysRemaining: daysRemaining,
		IsValid:       isValid,
	}, nil
}

func (uc *UserUseCase) PurchaseMonthlyCard(ctx context.Context, userID string, req *v1.PurchaseMonthlyCardRequest) (*v1.PurchaseMonthlyCardData, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uid, 1, 100)
	if err != nil {
		return nil, err
	}

	plateMap := make(map[string]bool)
	for _, v := range vehicles {
		plateMap[v.PlateNumber] = true
	}

	if !plateMap[req.PlateNumber] {
		return nil, fmt.Errorf("plate number not bound to user")
	}

	monthlyPrice := 300.0
	amount := monthlyPrice * float64(req.Months)

	payData, err := uc.paymentClient.CreateMonthlyCardPayment(ctx, req.PlateNumber, req.Months, amount, req.PayMethod, req.OpenId)
	if err != nil {
		return nil, err
	}

	return &v1.PurchaseMonthlyCardData{
		OrderId: payData.OrderId,
		Amount:  payData.Amount,
		PayUrl:  payData.PayUrl,
		QrCode:  payData.QrCode,
	}, nil
}

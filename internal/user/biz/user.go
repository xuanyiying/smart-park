package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/user/v1"
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
	userRepo     UserRepo
	jwtManager   *auth.JWTManager
	wechatClient *wechat.Client
	log          *log.Helper
}

func NewUserUseCase(userRepo UserRepo, jwtManager *auth.JWTManager, wechatClient *wechat.Client, logger log.Logger) *UserUseCase {
	return &UserUseCase{
		userRepo:     userRepo,
		jwtManager:   jwtManager,
		wechatClient: wechatClient,
		log:          log.NewHelper(logger),
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

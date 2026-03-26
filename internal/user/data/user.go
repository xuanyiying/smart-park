package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/user/biz"
	"github.com/xuanyiying/smart-park/internal/user/data/ent"
	"github.com/xuanyiying/smart-park/internal/user/data/ent/user"
	"github.com/xuanyiying/smart-park/internal/user/data/ent/uservehicle"
)

type userRepo struct {
	data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{data: data}
}

func (r *userRepo) GetUserByOpenID(ctx context.Context, openID string) (*biz.User, error) {
	u, err := r.data.db.User.Query().
		Where(user.OpenID(openID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.User{
		ID:        u.ID,
		OpenID:    u.OpenID,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
	}, nil
}

func (r *userRepo) CreateUser(ctx context.Context, user *biz.User) error {
	_, err := r.data.db.User.Create().
		SetID(user.ID).
		SetOpenID(user.OpenID).
		Save(ctx)
	return err
}

func (r *userRepo) GetUserByID(ctx context.Context, userID uuid.UUID) (*biz.User, error) {
	u, err := r.data.db.User.Get(ctx, userID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.User{
		ID:        u.ID,
		OpenID:    u.OpenID,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
	}, nil
}

func (r *userRepo) BindVehicle(ctx context.Context, userVehicle *biz.UserVehicle) error {
	_, err := r.data.db.UserVehicle.Create().
		SetID(userVehicle.ID).
		SetUserID(userVehicle.UserID).
		SetPlateNumber(userVehicle.PlateNumber).
		SetOwnerName(userVehicle.OwnerName).
		SetOwnerPhone(userVehicle.OwnerPhone).
		Save(ctx)
	return err
}

func (r *userRepo) UnbindVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) error {
	_, err := r.data.db.UserVehicle.Delete().
		Where(uservehicle.UserID(userID), uservehicle.PlateNumber(plateNumber)).
		Exec(ctx)
	return err
}

func (r *userRepo) ListUserVehicles(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*biz.UserVehicle, int64, error) {
	query := r.data.db.UserVehicle.Query().
		Where(uservehicle.UserID(userID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	vehicles, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)

	if err != nil {
		return nil, 0, err
	}

	var result []*biz.UserVehicle
	for _, v := range vehicles {
		result = append(result, &biz.UserVehicle{
			ID:          v.ID,
			UserID:      v.UserID,
			PlateNumber: v.PlateNumber,
			OwnerName:   v.OwnerName,
			OwnerPhone:  v.OwnerPhone,
			CreatedAt:   v.CreatedAt,
		})
	}

	return result, int64(total), nil
}

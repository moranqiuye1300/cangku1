package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"short-video-platform/user-service/internal/model"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, user *model.User) error {
	role := user.Role
	if role == "" {
		role = model.RoleUser
	}
	record := &model.UserRecord{
		Username:      user.Username,
		PasswordHash:  user.Password,
		Nickname:      user.Nickname,
		Avatar:        user.Avatar,
		Role:          role,
		OAuthProvider: user.OAuthProvider,
		OAuthID:       user.OAuthID,
	}
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		if isDuplicate(err) {
			return ErrAlreadyExists
		}
		return err
	}
	created := model.RecordToUser(record)
	user.ID = created.ID
	user.CreatedAt = created.CreatedAt
	return nil
}

func (r *MySQLRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	numID, err := model.ParseUserID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	var record model.UserRecord
	if err := r.db.WithContext(ctx).First(&record, numID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.RecordToUser(&record), nil
}

func (r *MySQLRepository) GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error) {
	var record model.UserRecord
	if err := r.db.WithContext(ctx).
		Where("oauth_provider = ? AND oauth_id = ?", provider, oauthID).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.RecordToUser(&record), nil
}

func (r *MySQLRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var record model.UserRecord
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.RecordToUser(&record), nil
}

func (r *MySQLRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.UserRecord{}).Count(&count).Error
	return count, err
}

func (r *MySQLRepository) UpdateAvatar(ctx context.Context, id, avatarURL string) error {
	numID, err := model.ParseUserID(id)
	if err != nil {
		return ErrNotFound
	}
	res := r.db.WithContext(ctx).Model(&model.UserRecord{}).
		Where("id = ?", numID).
		Update("avatar", avatarURL)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MySQLRepository) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.UserRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []model.UserRecord
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("id asc").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	out := make([]model.User, 0, len(records))
	for i := range records {
		u := model.RecordToUser(&records[i])
		u.Password = ""
		out = append(out, *u)
	}
	return out, int(total), nil
}

func (r *MySQLRepository) UpdateRole(ctx context.Context, id, role string) error {
	numID, err := model.ParseUserID(id)
	if err != nil {
		return ErrNotFound
	}
	res := r.db.WithContext(ctx).Model(&model.UserRecord{}).
		Where("id = ?", numID).
		Update("role", role)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "UNIQUE constraint failed")
}

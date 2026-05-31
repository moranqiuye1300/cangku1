package repository

import (
	"context"
	"errors"

	"short-video-platform/user-service/internal/model"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("username already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByOAuth(ctx context.Context, provider, oauthID string) (*model.User, error)
	Count(ctx context.Context) (int64, error)
	UpdateAvatar(ctx context.Context, id, avatarURL string) error
	List(ctx context.Context, page, pageSize int) ([]model.User, int, error)
	UpdateRole(ctx context.Context, id, role string) error
}

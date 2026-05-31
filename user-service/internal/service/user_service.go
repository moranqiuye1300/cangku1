package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
	"short-video-platform/user-service/internal/jwtutil"
	"short-video-platform/user-service/internal/model"
	"short-video-platform/user-service/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type UserService struct {
	repo        repository.UserRepository
	videoClient videopb.VideoServiceClient
}

func NewUserService(repo repository.UserRepository, videoClient videopb.VideoServiceClient) *UserService {
	return &UserService{repo: repo, videoClient: videoClient}
}

func (s *UserService) Register(ctx context.Context, username, password, nickname string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Username: username,
		Password: string(hash),
		Nickname: nickname,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	copy := *u
	copy.Password = ""
	return &copy, nil
}

func (s *UserService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return nil, "", ErrInvalidCredentials
	}
	token, err := jwtutil.SignWithRole(u.ID, u.Username, u.Role)
	if err != nil {
		return nil, "", err
	}
	copy := *u
	copy.Password = ""
	return &copy, token, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u.Password = ""
	return u, nil
}

func (s *UserService) OAuthLogin(ctx context.Context, provider, oauthID, nickname, avatar string) (*model.User, string, error) {
	provider = strings.TrimSpace(provider)
	oauthID = strings.TrimSpace(oauthID)
	if provider == "" || oauthID == "" {
		return nil, "", errors.New("provider and oauth_id required")
	}
	if u, err := s.repo.GetByOAuth(ctx, provider, oauthID); err == nil {
		token, err := auth.SignWithRole(u.ID, u.Username, u.Role)
		if err != nil {
			return nil, "", err
		}
		u.Password = ""
		return u, token, nil
	} else if err != repository.ErrNotFound {
		return nil, "", err
	}
	randPass := randomSecret()
	hash, err := bcrypt.GenerateFromPassword([]byte(randPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	username := fmt.Sprintf("%s_%s", provider, oauthID)
	if nickname == "" {
		nickname = username
	}
	u := &model.User{
		Username:      username,
		Password:      string(hash),
		Nickname:      nickname,
		Avatar:        avatar,
		OAuthProvider: provider,
		OAuthID:       oauthID,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, "", err
	}
	token, err := auth.SignWithRole(u.ID, u.Username, u.Role)
	if err != nil {
		return nil, "", err
	}
	u.Password = ""
	return u, token, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, userID, avatarURL string) (*model.User, error) {
	avatarURL = strings.TrimSpace(avatarURL)
	if userID == "" || avatarURL == "" {
		return nil, errors.New("user_id and avatar_url required")
	}
	if err := s.repo.UpdateAvatar(ctx, userID, avatarURL); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, userID)
}

func (s *UserService) ListUsers(ctx context.Context, page, pageSize int32) ([]model.User, int, error) {
	return s.repo.List(ctx, int(page), int(pageSize))
}

func (s *UserService) SetUserRole(ctx context.Context, userID, role string) (*model.User, error) {
	role = strings.TrimSpace(role)
	switch role {
	case model.RoleUser, model.RoleReviewer, model.RoleAdmin:
	default:
		return nil, fmt.Errorf("invalid role")
	}
	if _, err := s.repo.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateRole(ctx, userID, role); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, userID)
}

func randomSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *UserService) GetUserVideos(ctx context.Context, userID string, page, pageSize int32) (*videopb.ListVideosByUserResponse, error) {
	if _, err := s.repo.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	return s.videoClient.ListVideosByUser(ctx, &videopb.ListVideosByUserRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
}

func SeedUsers(ctx context.Context, repo repository.UserRepository) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	count, err := repo.Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		seed := []model.User{
			{Username: "admin", Password: string(hash), Nickname: "管理员", Role: model.RoleAdmin},
			{Username: "alice", Password: string(hash), Nickname: "Alice"},
			{Username: "bob", Password: string(hash), Nickname: "Bob"},
		}
		for i := range seed {
			if err := repo.Create(ctx, &seed[i]); err != nil {
				return err
			}
		}
		return nil
	}

	if _, err := repo.GetByUsername(ctx, "admin"); err == repository.ErrNotFound {
		admin := model.User{
			Username: "admin",
			Password: string(hash),
			Nickname: "管理员",
			Role:     model.RoleAdmin,
		}
		if err := repo.Create(ctx, &admin); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// Legacy helper kept for docs; seed uses SeedUsers now.
func DefaultSeedUsers() []model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	return []model.User{
		{ID: "u1", Username: "alice", Password: string(hash), Nickname: "Alice", CreatedAt: time.Now().Unix()},
		{ID: "u2", Username: "bob", Password: string(hash), Nickname: "Bob", CreatedAt: time.Now().Unix()},
	}
}

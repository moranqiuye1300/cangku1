package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"short-video-platform/gen/userpb"
	"short-video-platform/pkg/auth"
	"short-video-platform/user-service/internal/model"
	"short-video-platform/user-service/internal/repository"
	"short-video-platform/user-service/internal/service"
)

type UserGRPCServer struct {
	userpb.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewUserGRPCServer(svc *service.UserService) *UserGRPCServer {
	return &UserGRPCServer{svc: svc}
}

func (s *UserGRPCServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	u, err := s.svc.Register(ctx, req.GetUsername(), req.GetPassword(), req.GetNickname())
	if err != nil {
		if err == repository.ErrAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.RegisterResponse{User: toProto(u)}, nil
}

func (s *UserGRPCServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	u, token, err := s.svc.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return nil, status.Error(codes.Unauthenticated, "invalid username or password")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.LoginResponse{User: toProto(u), Token: token}, nil
}

func (s *UserGRPCServer) GetUserInfo(ctx context.Context, req *userpb.GetUserInfoRequest) (*userpb.GetUserInfoResponse, error) {
	u, err := s.svc.GetByID(ctx, req.GetUserId())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.GetUserInfoResponse{User: toProto(u)}, nil
}

func (s *UserGRPCServer) OAuthLogin(ctx context.Context, req *userpb.OAuthLoginRequest) (*userpb.OAuthLoginResponse, error) {
	u, token, err := s.svc.OAuthLogin(ctx, req.GetProvider(), req.GetOauthId(), req.GetNickname(), req.GetAvatar())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.OAuthLoginResponse{User: toProto(u), Token: token}, nil
}

func (s *UserGRPCServer) GetUserVideos(ctx context.Context, req *userpb.GetUserVideosRequest) (*userpb.GetUserVideosResponse, error) {
	resp, err := s.svc.GetUserVideos(ctx, req.GetUserId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.GetUserVideosResponse{Videos: resp.GetVideos(), Total: resp.GetTotal()}, nil
}

func (s *UserGRPCServer) UpdateAvatar(ctx context.Context, req *userpb.UpdateAvatarRequest) (*userpb.UpdateAvatarResponse, error) {
	if req.GetUserId() == "" || strings.TrimSpace(req.GetAvatarUrl()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and avatar_url required")
	}
	callerID := auth.UserIDFromContext(ctx)
	if callerID == "" || callerID != req.GetUserId() {
		return nil, status.Error(codes.PermissionDenied, "can only update own avatar")
	}
	u, err := s.svc.UpdateAvatar(ctx, req.GetUserId(), req.GetAvatarUrl())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.UpdateAvatarResponse{User: toProto(u)}, nil
}

func (s *UserGRPCServer) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	list, total, err := s.svc.ListUsers(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	users := make([]*userpb.User, 0, len(list))
	for i := range list {
		users = append(users, toProto(&list[i]))
	}
	return &userpb.ListUsersResponse{Users: users, Total: int32(total)}, nil
}

func (s *UserGRPCServer) SetUserRole(ctx context.Context, req *userpb.SetUserRoleRequest) (*userpb.SetUserRoleResponse, error) {
	if req.GetUserId() == "" || req.GetRole() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and role required")
	}
	u, err := s.svc.SetUserRole(ctx, req.GetUserId(), req.GetRole())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		if strings.Contains(err.Error(), "invalid role") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userpb.SetUserRoleResponse{User: toProto(u)}, nil
}

func toProto(u *model.User) *userpb.User {
	role := u.Role
	if role == "" {
		role = model.RoleUser
	}
	return &userpb.User{
		Id:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		Role:      role,
	}
}

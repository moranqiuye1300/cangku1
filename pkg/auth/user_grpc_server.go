package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserAuthRules defines user-service gRPC auth: internal key + optional JWT/role.
type UserAuthRules struct {
	InternalOnly  map[string]bool
	RequireJWT    map[string]bool
	RequireAdmin  map[string]bool
}

func DefaultUserAuthRules() UserAuthRules {
	return UserAuthRules{
		InternalOnly: map[string]bool{
			"/user.v1.UserService/Register":       true,
			"/user.v1.UserService/Login":          true,
			"/user.v1.UserService/GetUserInfo":    true,
			"/user.v1.UserService/OAuthLogin":     true,
			"/user.v1.UserService/GetUserVideos":  true,
		},
		RequireJWT: map[string]bool{
			"/user.v1.UserService/UpdateAvatar": true,
		},
		RequireAdmin: map[string]bool{
			"/user.v1.UserService/ListUsers":   true,
			"/user.v1.UserService/SetUserRole": true,
		},
	}
}

func UserUnaryServerInterceptor(rules UserAuthRules) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if InternalKeyFromIncoming(ctx) != InternalKey() {
			return nil, status.Error(codes.Unauthenticated, "invalid internal key")
		}
		if rules.InternalOnly[info.FullMethod] {
			return handler(ctx, req)
		}
		if rules.RequireAdmin[info.FullMethod] {
			token := TokenFromIncoming(ctx)
			claims, err := ParseToken(token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
			}
			if claims.Role != RoleAdmin {
				return nil, status.Error(codes.PermissionDenied, "admin role required")
			}
			ctx = withClaims(ctx, claims)
			return handler(ctx, req)
		}
		if rules.RequireJWT[info.FullMethod] {
			token := TokenFromIncoming(ctx)
			claims, err := ParseToken(token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
			}
			ctx = withClaims(ctx, claims)
			return handler(ctx, req)
		}
		return nil, status.Error(codes.PermissionDenied, "unknown method")
	}
}

func withClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, ctxUserIDKey{}, claims.UserID)
	ctx = context.WithValue(ctx, ctxUsernameKey{}, claims.Username)
	ctx = context.WithValue(ctx, ctxRoleKey{}, claims.Role)
	return ctx
}

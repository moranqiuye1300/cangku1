package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VideoAuthRules 定义 video-service 方法鉴权策略。
type VideoAuthRules struct {
	Public          map[string]bool
	Internal        map[string]bool
	RequireJWT      map[string]bool
	RequireAdmin    map[string]bool
	RequireReviewer map[string]bool
}

func DefaultVideoAuthRules() VideoAuthRules {
	return VideoAuthRules{
		Public: map[string]bool{
			"/video.v1.VideoService/GetVideoList":         true,
			"/video.v1.VideoService/GetRecommendedFeed":   true,
			"/video.v1.VideoService/GetVideoInfo":         true,
			"/video.v1.VideoService/ListVideosByUser":     true,
			"/video.v1.VideoService/SearchVideos":          true,
			"/video.v1.VideoService/ListBarrages":           true,
			"/video.v1.VideoService/GetVideoEngagement":   true,
			"/video.v1.VideoService/ListComments":           true,
			"/video.v1.VideoService/Health":                 true,
		},
		Internal: map[string]bool{
			"/video.v1.VideoService/UpdateTranscodeResult": true,
		},
		RequireJWT: map[string]bool{
			"/video.v1.VideoService/CreateVideo":              true,
			"/video.v1.VideoService/PostBarrage":              true,
			"/video.v1.VideoService/ToggleLike":               true,
			"/video.v1.VideoService/ToggleFavorite":           true,
			"/video.v1.VideoService/PostComment":              true,
			"/video.v1.VideoService/ListUserLikedVideos":      true,
			"/video.v1.VideoService/ListUserFavoriteVideos":   true,
		},
		RequireAdmin: map[string]bool{
			"/video.v1.VideoService/AdminListVideos":           true,
			"/video.v1.VideoService/AdminSoftDeleteVideo":      true,
			"/video.v1.VideoService/AdminRestoreVideo":         true,
			"/video.v1.VideoService/AdminPermanentDeleteVideo": true,
			"/video.v1.VideoService/ListRecycleBin":            true,
			"/video.v1.VideoService/ListAuditLogs":             true,
		},
		RequireReviewer: map[string]bool{
			"/video.v1.VideoService/ReviewerListPending":    true,
			"/video.v1.VideoService/ReviewerApproveSource":  true,
			"/video.v1.VideoService/ReviewerRejectSource":   true,
			"/video.v1.VideoService/ReviewerApprovePublish": true,
			"/video.v1.VideoService/ReviewerRejectPublish":  true,
		},
	}
}

func UnaryServerInterceptor(rules VideoAuthRules) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if rules.Public[info.FullMethod] {
			return handler(ctx, req)
		}
		if rules.Internal[info.FullMethod] {
			if InternalKeyFromIncoming(ctx) != InternalKey() {
				return nil, status.Error(codes.Unauthenticated, "invalid internal key")
			}
			return handler(ctx, req)
		}
		if rules.RequireJWT[info.FullMethod] || rules.RequireAdmin[info.FullMethod] || rules.RequireReviewer[info.FullMethod] {
			token := TokenFromIncoming(ctx)
			claims, err := ParseToken(token)
			if err != nil {
				return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
			}
			ctx = withClaims(ctx, claims)
			role := claims.Role
			if role == "" {
				role = RoleUser
			}
			if rules.RequireAdmin[info.FullMethod] && role != RoleAdmin {
				return nil, status.Error(codes.PermissionDenied, "admin role required")
			}
			if rules.RequireReviewer[info.FullMethod] && role != RoleReviewer && role != RoleAdmin {
				return nil, status.Error(codes.PermissionDenied, "reviewer role required")
			}
			return handler(ctx, req)
		}
		return handler(ctx, req)
	}
}

type ctxUserIDKey struct{}
type ctxUsernameKey struct{}
type ctxRoleKey struct{}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserIDKey{}).(string)
	return v
}

func UsernameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUsernameKey{}).(string)
	return v
}

func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxRoleKey{}).(string)
	if v == "" {
		return RoleUser
	}
	return v
}

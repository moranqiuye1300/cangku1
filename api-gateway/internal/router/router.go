package router

import (
	"short-video-platform/api-gateway/internal/handler"
	"short-video-platform/api-gateway/internal/middleware"
	"short-video-platform/gen/userpb"
	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"

	"github.com/gin-gonic/gin"
)

type Options struct {
	UserClient  userpb.UserServiceClient
	VideoClient videopb.VideoServiceClient
}

func Setup(opts Options) *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 64 << 20
	h := handler.New(opts.UserClient, opts.VideoClient)

	r.GET("/api/health", h.Health)
	r.GET("/api/v1/health", h.Health)

	// Protected raw uploads; public transcoded assets
	r.OPTIONS("/api/v1/media/uploads/*filepath", h.ServeMediaUploadOptions)
	r.GET("/api/v1/media/uploads/*filepath", middleware.OptionalJWT(), h.ServeMediaUpload)
	r.GET("/media/transcoded/*filepath", h.ServeMediaPublic)
	r.GET("/media/avatars/*filepath", h.ServeMediaPublic)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimit())
	{
		v1.POST("/auth/register", h.Register)
		v1.POST("/auth/login", h.Login)
		v1.GET("/auth/oauth/github/url", h.OAuthGitHubURL)
		v1.GET("/auth/oauth/github/callback", h.OAuthGitHubCallback)
		v1.GET("/auth/oauth/mock", h.OAuthMock)

		v1.POST("/ai/ask", h.AIAsk)

		v1.GET("/users/:id", h.GetUser)
		v1.GET("/users/:id/videos", h.GetUserVideos)
		v1.GET("/users/:id/likes", middleware.JWTAuth(), h.GetUserLikedVideos)
		v1.GET("/users/:id/favorites", middleware.JWTAuth(), h.GetUserFavoriteVideos)
		v1.POST("/users/me/avatar", middleware.JWTAuth(), h.UploadAvatar)

		v1.GET("/videos/search", h.SearchVideos)
		v1.GET("/videos/feed", middleware.OptionalJWT(), h.GetRecommendedFeed)
		v1.GET("/videos", h.ListVideos)
		v1.GET("/videos/:id/engagement", middleware.OptionalJWT(), h.GetVideoEngagement)
		v1.GET("/videos/:id/comments", h.ListComments)
		v1.POST("/videos/:id/comments", middleware.JWTAuth(), h.PostComment)
		v1.POST("/videos/:id/like", middleware.JWTAuth(), h.ToggleLike)
		v1.POST("/videos/:id/favorite", middleware.JWTAuth(), h.ToggleFavorite)
		v1.GET("/videos/:id/barrages", h.ListBarrages)
		v1.POST("/videos/:id/barrages", middleware.JWTAuth(), h.PostBarrage)
		v1.GET("/videos/:id", h.GetVideo)
		v1.POST("/videos/upload/init", middleware.JWTAuth(), h.InitChunkUpload)
		v1.POST("/videos/upload/chunk", middleware.JWTAuth(), h.UploadChunk)
		v1.POST("/videos/upload/complete", middleware.JWTAuth(), h.CompleteChunkUpload)
		v1.POST("/videos/upload", middleware.JWTAuth(), h.UploadVideo)

		admin := v1.Group("/admin")
		admin.POST("/login", h.AdminLogin)
		adminAuth := admin.Group("")
		adminAuth.Use(middleware.JWTAuth(), middleware.RequireRole(auth.RoleAdmin))
		{
			adminAuth.GET("/users", h.AdminListUsers)
			adminAuth.PATCH("/users/:id/role", h.AdminSetUserRole)
			adminAuth.GET("/videos", h.AdminListVideos)
			adminAuth.DELETE("/videos/:id", h.AdminDeleteVideo)
			adminAuth.POST("/videos/:id/restore", h.AdminRestoreVideo)
			adminAuth.DELETE("/videos/:id/permanent", h.AdminPermanentDeleteVideo)
			adminAuth.GET("/recycle-bin", h.AdminListRecycleBin)
			adminAuth.GET("/audit-logs", h.AdminListAuditLogs)
		}

		reviewer := v1.Group("/reviewer")
		reviewer.Use(middleware.JWTAuth(), middleware.RequireRole(auth.RoleReviewer, auth.RoleAdmin))
		{
			reviewer.GET("/videos", h.ReviewerListVideos)
			reviewer.POST("/videos/:id/approve-source", h.ReviewerApproveSource)
			reviewer.POST("/videos/:id/reject-source", h.ReviewerRejectSource)
			reviewer.POST("/videos/:id/approve-publish", h.ReviewerApprovePublish)
			reviewer.POST("/videos/:id/reject-publish", h.ReviewerRejectPublish)
			reviewer.POST("/videos/:id/reject", h.ReviewerDeleteVideo)
		}
	}

	return r
}

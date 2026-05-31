package handler

import (
	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/gen/videopb"
)

func (h *Handler) GetVideoEngagement(c *gin.Context) {
	videoID := c.Param("id")
	viewerID, _ := c.Get("user_id")
	uid, _ := viewerID.(string)
	resp, err := h.videoClient.GetVideoEngagement(h.ctx(c), &videopb.GetVideoEngagementRequest{
		VideoId:      videoID,
		ViewerUserId: uid,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"engagement": resp.GetEngagement()})
}

func (h *Handler) ToggleLike(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	resp, err := h.videoClient.ToggleLike(h.ctx(c), &videopb.ToggleLikeRequest{
		VideoId: videoID,
		UserId:  uid,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"liked": resp.GetLiked(), "like_count": resp.GetLikeCount()})
}

func (h *Handler) ToggleFavorite(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	resp, err := h.videoClient.ToggleFavorite(h.ctx(c), &videopb.ToggleFavoriteRequest{
		VideoId: videoID,
		UserId:  uid,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"favorited": resp.GetFavorited(), "favorite_count": resp.GetFavoriteCount()})
}

func (h *Handler) ListComments(c *gin.Context) {
	videoID := c.Param("id")
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListComments(h.ctx(c), &videopb.ListCommentsRequest{
		VideoId:  videoID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{
		"comments":  resp.GetComments(),
		"total":     resp.GetTotal(),
		"page":      page,
		"page_size": pageSize,
	})
}

type postCommentReq struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

func (h *Handler) PostComment(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	uid, _ := userID.(string)
	uname, _ := username.(string)
	if uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	var req postCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	resp, err := h.videoClient.PostComment(h.ctx(c), &videopb.PostCommentRequest{
		VideoId:  videoID,
		UserId:   uid,
		Username: uname,
		Content:  req.Content,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"comment": resp.GetComment()})
}

func (h *Handler) assertSelfUser(c *gin.Context) (string, bool) {
	userID := c.Param("id")
	uid, _ := c.Get("user_id")
	self, _ := uid.(string)
	if self == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return "", false
	}
	if self != userID {
		response.Fail(c, 403, 40300, "forbidden")
		return "", false
	}
	return self, true
}

func (h *Handler) GetUserLikedVideos(c *gin.Context) {
	userID, ok := h.assertSelfUser(c)
	if !ok {
		return
	}
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListUserLikedVideos(h.ctx(c), &videopb.ListVideosByUserRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{
		"videos":    resp.GetVideos(),
		"total":     resp.GetTotal(),
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *Handler) GetUserFavoriteVideos(c *gin.Context) {
	userID, ok := h.assertSelfUser(c)
	if !ok {
		return
	}
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListUserFavoriteVideos(h.ctx(c), &videopb.ListVideosByUserRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{
		"videos":    resp.GetVideos(),
		"total":     resp.GetTotal(),
		"page":      page,
		"page_size": pageSize,
	})
}

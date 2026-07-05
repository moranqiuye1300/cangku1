package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/gen/userpb"
	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
)

func (h *Handler) AdminLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	resp, err := h.userClient.Login(h.ctx(c), &userpb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	role := resp.GetUser().GetRole()
	if role == "" {
		role = auth.RoleUser
	}
	if role != auth.RoleAdmin && role != auth.RoleReviewer {
		response.Fail(c, 403, 40300, "not an admin or reviewer account")
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser(), "token": resp.GetToken()})
}

func (h *Handler) AdminListUsers(c *gin.Context) {
	page, pageSize := pageQuery(c)
	resp, err := h.userClient.ListUsers(h.ctx(c), &userpb.ListUsersRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"users": resp.GetUsers(), "total": resp.GetTotal()})
}

type setRoleReq struct {
	Role string `json:"role" binding:"required"`
}

func (h *Handler) AdminSetUserRole(c *gin.Context) {
	var req setRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	userID := c.Param("id")
	resp, err := h.userClient.SetUserRole(h.ctx(c), &userpb.SetUserRoleRequest{
		UserId: userID,
		Role:   req.Role,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser()})
}

func (h *Handler) AdminListVideos(c *gin.Context) {
	page, pageSize := pageQuery(c)
	includeDeleted := c.Query("include_deleted") == "1"
	resp, err := h.videoClient.AdminListVideos(h.ctx(c), &videopb.AdminListVideosRequest{
		Page:           page,
		PageSize:       pageSize,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"videos": resp.GetVideos(), "total": resp.GetTotal()})
}

type deleteVideoReq struct {
	Reason string `json:"reason"`
}

func (h *Handler) AdminDeleteVideo(c *gin.Context) {
	var req deleteVideoReq
	_ = c.ShouldBindJSON(&req)
	videoID := c.Param("id")
	resp, err := h.videoClient.AdminSoftDeleteVideo(h.ctx(c), &videopb.AdminSoftDeleteVideoRequest{
		VideoId:          videoID,
		OperatorId:       c.GetString("user_id"),
		OperatorUsername: c.GetString("username"),
		Reason:           strings.TrimSpace(req.Reason),
		Ip:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"video": resp.GetVideo()})
}

func (h *Handler) AdminRestoreVideo(c *gin.Context) {
	videoID := c.Param("id")
	resp, err := h.videoClient.AdminRestoreVideo(h.ctx(c), &videopb.AdminRestoreVideoRequest{
		VideoId:          videoID,
		OperatorId:       c.GetString("user_id"),
		OperatorUsername: c.GetString("username"),
		Ip:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"video": resp.GetVideo()})
}

func (h *Handler) AdminPermanentDeleteVideo(c *gin.Context) {
	videoID := c.Param("id")
	_, err := h.videoClient.AdminPermanentDeleteVideo(h.ctx(c), &videopb.AdminPermanentDeleteVideoRequest{
		VideoId:          videoID,
		OperatorId:       c.GetString("user_id"),
		OperatorUsername: c.GetString("username"),
		Ip:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) AdminListRecycleBin(c *gin.Context) {
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListRecycleBin(h.ctx(c), &videopb.ListRecycleBinRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"videos": resp.GetVideos(), "total": resp.GetTotal()})
}

func (h *Handler) AdminListAuditLogs(c *gin.Context) {
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListAuditLogs(h.ctx(c), &videopb.ListAuditLogsRequest{
		Page:       page,
		PageSize:   pageSize,
		TargetType: c.Query("target_type"),
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"logs": resp.GetLogs(), "total": resp.GetTotal()})
}

func (h *Handler) ReviewerListVideos(c *gin.Context) {
	stage := c.DefaultQuery("stage", "source")
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ReviewerListPending(h.ctx(c), &videopb.ReviewerListPendingRequest{
		Stage:    stage,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"videos": resp.GetVideos(), "total": resp.GetTotal()})
}

func (h *Handler) reviewerAction(
	c *gin.Context,
	call func(context.Context, *videopb.ReviewerReviewActionRequest, ...grpc.CallOption) (*videopb.ReviewerReviewActionResponse, error),
) {
	var req deleteVideoReq
	_ = c.ShouldBindJSON(&req)
	videoID := c.Param("id")
	resp, err := call(h.ctx(c), &videopb.ReviewerReviewActionRequest{
		VideoId:          videoID,
		OperatorId:       c.GetString("user_id"),
		OperatorUsername: c.GetString("username"),
		Reason:           strings.TrimSpace(req.Reason),
		Ip:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"video": resp.GetVideo()})
}

func (h *Handler) ReviewerApproveSource(c *gin.Context) {
	h.reviewerAction(c, h.videoClient.ReviewerApproveSource)
}

func (h *Handler) ReviewerRejectSource(c *gin.Context) {
	var req deleteVideoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		response.Fail(c, 400, 40001, "reason required for review rejection")
		return
	}
	h.reviewerAction(c, h.videoClient.ReviewerRejectSource)
}

func (h *Handler) ReviewerApprovePublish(c *gin.Context) {
	h.reviewerAction(c, h.videoClient.ReviewerApprovePublish)
}

func (h *Handler) ReviewerRejectPublish(c *gin.Context) {
	var req deleteVideoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		response.Fail(c, 400, 40001, "reason required for review rejection")
		return
	}
	h.reviewerAction(c, h.videoClient.ReviewerRejectPublish)
}

func (h *Handler) ReviewerDeleteVideo(c *gin.Context) {
	h.ReviewerRejectPublish(c)
}

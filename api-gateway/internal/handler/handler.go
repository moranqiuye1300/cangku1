package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/api-gateway/internal/storage"
	"short-video-platform/gen/userpb"
	"short-video-platform/gen/videopb"
)

type Handler struct {
	userClient  userpb.UserServiceClient
	videoClient videopb.VideoServiceClient
}

func New(userClient userpb.UserServiceClient, videoClient videopb.VideoServiceClient) *Handler {
	return &Handler{userClient: userClient, videoClient: videoClient}
}

func (h *Handler) Health(c *gin.Context) {
	response.OK(c, gin.H{
		"service": "short-video-platform",
		"stage":   "4",
		"time":    time.Now().Format(time.RFC3339),
	})
}

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=64"`
	Nickname string `json:"nickname" binding:"max=32"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	resp, err := h.userClient.Register(h.ctx(c), &userpb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser()})
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
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
	response.OK(c, gin.H{"user": resp.GetUser(), "token": resp.GetToken()})
}

func (h *Handler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	resp, err := h.userClient.GetUserInfo(h.ctx(c), &userpb.GetUserInfoRequest{UserId: userID})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser()})
}

func (h *Handler) GetUserVideos(c *gin.Context) {
	userID := c.Param("id")
	page, pageSize := pageQuery(c)
	resp, err := h.userClient.GetUserVideos(h.ctx(c), &userpb.GetUserVideosRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"videos": resp.GetVideos(), "total": resp.GetTotal(), "page": page, "page_size": pageSize})
}

func (h *Handler) SearchVideos(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		response.Fail(c, 400, 40001, "q is required")
		return
	}
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.SearchVideos(h.ctx(c), &videopb.SearchVideosRequest{
		Keyword:  keyword,
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
		"keyword":   keyword,
	})
}

func (h *Handler) ListVideos(c *gin.Context) {
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.GetVideoList(h.ctx(c), &videopb.GetVideoListRequest{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"videos": resp.GetVideos(), "total": resp.GetTotal(), "page": page, "page_size": pageSize})
}

func (h *Handler) GetRecommendedFeed(c *gin.Context) {
	page, pageSize := pageQuery(c)
	viewerID, _ := c.Get("user_id")
	viewer, _ := viewerID.(string)
	resp, err := h.videoClient.GetRecommendedFeed(h.ctx(c), &videopb.GetRecommendedFeedRequest{
		ViewerUserId: viewer,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{
		"videos":       resp.GetVideos(),
		"total":        resp.GetTotal(),
		"page":         page,
		"page_size":    pageSize,
		"personalized": resp.GetPersonalized(),
	})
}

func (h *Handler) GetVideo(c *gin.Context) {
	videoID := c.Param("id")
	resp, err := h.videoClient.GetVideoInfo(h.ctx(c), &videopb.GetVideoInfoRequest{VideoId: videoID})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"video": resp.GetVideo()})
}

type initChunkUploadReq struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size" binding:"required"`
	ChunkSize   int64  `json:"chunk_size"`
}

type completeChunkUploadReq struct {
	SessionID   string `json:"session_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) InitChunkUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	var req initChunkUploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	if req.Size <= 0 {
		response.Fail(c, 400, 40001, "size must be positive")
		return
	}
	const maxVideoBytes = 200 << 20
	if req.Size > maxVideoBytes {
		response.Fail(c, 400, 40001, "video file too large (max 200MB)")
		return
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}
	session, err := storage.CreateUploadSession(storage.MediaRoot(), uid, "", req.Filename, req.ContentType, req.Size, req.ChunkSize)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	response.OK(c, gin.H{"session_id": session.ID, "chunk_count": session.ChunkCount, "chunk_size": session.ChunkSize})
}

func (h *Handler) UploadChunk(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	sessionID := strings.TrimSpace(c.PostForm("session_id"))
	chunkIndexStr := strings.TrimSpace(c.PostForm("chunk_index"))
	if sessionID == "" || chunkIndexStr == "" {
		response.Fail(c, 400, 40001, "session_id and chunk_index are required")
		return
	}
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		response.Fail(c, 400, 40001, "invalid chunk_index")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 400, 40001, "file is required")
		return
	}
	if file.Size <= 0 {
		response.Fail(c, 400, 40001, "empty file")
		return
	}
	src, err := file.Open()
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	defer src.Close()
	if err := storage.SaveUploadChunk(storage.MediaRoot(), sessionID, chunkIndex, src, file.Size); err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	status, err := storage.GetUploadSessionStatus(storage.MediaRoot(), sessionID)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	if status.UserID != "" && status.UserID != uid {
		response.Fail(c, 403, 40300, "session does not belong to user")
		return
	}
	response.OK(c, gin.H{"session_id": sessionID, "chunk_index": chunkIndex, "uploaded_chunks": status.Uploaded, "total_chunks": status.ChunkCount})
}

func (h *Handler) CompleteChunkUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	var req completeChunkUploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	relPath, err := storage.CompleteUploadFromChunks(storage.MediaRoot(), req.SessionID, uid)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	resp, err := h.videoClient.CreateVideo(h.ctx(c), &videopb.CreateVideoRequest{
		UserId:      uid,
		Title:       req.Title,
		Description: req.Description,
		SourcePath:  relPath,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"video": resp.GetVideo(), "source_path": relPath})
}

func (h *Handler) UploadVideo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}

	title := c.PostForm("title")
	if title == "" {
		response.Fail(c, 400, 40001, "title is required")
		return
	}
	description := c.PostForm("description")

	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 400, 40001, "file is required")
		return
	}
	if file.Size <= 0 {
		response.Fail(c, 400, 40001, "empty file")
		return
	}

	const maxVideoBytes = 200 << 20
	if file.Size > maxVideoBytes {
		response.Fail(c, 400, 40001, "video file too large (max 200MB)")
		return
	}

	mediaRoot := storage.MediaRoot()
	tempID := fmt.Sprintf("temp-%d", time.Now().UnixNano())
	src, err := file.Open()
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	defer src.Close()

	relPath, err := storage.SaveUpload(mediaRoot, uid, tempID, file.Filename, src)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}

	resp, err := h.videoClient.CreateVideo(h.ctx(c), &videopb.CreateVideoRequest{
		UserId:      uid,
		Title:       title,
		Description: description,
		SourcePath:  relPath,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}

	response.OK(c, gin.H{"video": resp.GetVideo()})
}

func pageQuery(c *gin.Context) (int32, int32) {
	page := parseInt32(c.DefaultQuery("page", "1"))
	pageSize := parseInt32(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return page, pageSize
}

func parseInt32(s string) int32 {
	var n int32
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int32(ch-'0')
	}
	return n
}

func (h *Handler) writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	switch st.Code() {
	case codes.NotFound:
		response.Fail(c, 404, 40400, st.Message())
	case codes.AlreadyExists:
		response.Fail(c, 409, 40900, st.Message())
	case codes.Unauthenticated:
		response.Fail(c, 401, 40100, st.Message())
	case codes.PermissionDenied:
		response.Fail(c, 403, 40300, st.Message())
	case codes.InvalidArgument:
		response.Fail(c, 400, 40000, st.Message())
	case codes.Unavailable:
		response.Fail(c, 503, 50300, st.Message())
	case codes.ResourceExhausted:
		response.Fail(c, 429, 42900, st.Message())
	default:
		response.Fail(c, 500, 50000, st.Message())
	}
}

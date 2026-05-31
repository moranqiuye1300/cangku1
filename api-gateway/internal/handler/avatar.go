package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/api-gateway/internal/storage"
	"short-video-platform/gen/userpb"
)

func (h *Handler) UploadAvatar(c *gin.Context) {
	uid, _ := c.Get("user_id")
	userID, _ := uid.(string)
	if userID == "" {
		response.Fail(c, 401, 40100, "unauthorized")
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
	if file.Size > 2<<20 {
		response.Fail(c, 400, 40001, "image must be <= 2MB")
		return
	}

	contentType := strings.ToLower(file.Header.Get("Content-Type"))
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		response.Fail(c, 400, 40001, "only image files are allowed")
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	defer src.Close()

	avatarURL, err := storage.SaveAvatar(storage.MediaRoot(), userID, file.Filename, src)
	if err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}

	resp, err := h.userClient.UpdateAvatar(h.ctx(c), &userpb.UpdateAvatarRequest{
		UserId:    userID,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser()})
}

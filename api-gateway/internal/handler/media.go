package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/api-gateway/internal/storage"
	"short-video-platform/pkg/auth"
)

// ServeMediaUpload serves raw source files only to owner or reviewer/admin.
func (h *Handler) ServeMediaUpload(c *gin.Context) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	rel = filepath.ToSlash(filepath.Clean("uploads/" + rel))
	if rel == "uploads/." || strings.Contains(rel, "..") || !strings.HasPrefix(rel, "uploads/") {
		response.Fail(c, 400, 40001, "invalid path")
		return
	}

	role, _ := c.Get("role")
	roleStr, _ := role.(string)
	if roleStr == "" {
		roleStr = auth.RoleUser
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)

	if !auth.CanAccessUploadPath(roleStr, uid, rel) {
		response.Fail(c, 403, 40300, "forbidden")
		return
	}

	abs := filepath.Join(storage.MediaRoot(), filepath.FromSlash(rel))
	c.File(abs)
}

// ServeMediaPublic serves transcoded HLS and covers (published assets).
func (h *Handler) ServeMediaPublic(c *gin.Context) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || strings.Contains(rel, "..") {
		response.Fail(c, 400, 40001, "invalid path")
		return
	}
	if strings.HasPrefix(rel, "uploads/") {
		response.Fail(c, 403, 40300, "forbidden")
		return
	}
	abs := filepath.Join(storage.MediaRoot(), filepath.FromSlash(rel))
	if _, err := filepath.Abs(abs); err != nil {
		response.Fail(c, 400, 40001, "invalid path")
		return
	}
	c.File(abs)
}

func (h *Handler) ServeMediaUploadOptions(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

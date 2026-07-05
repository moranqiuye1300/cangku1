package handler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"short-video-platform/pkg/auth"
)

func (h *Handler) ctx(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if token, ok := bearerToken(c); ok {
		ctx = auth.WithBearer(ctx, token)
	}
	return ctx
}

func bearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), true
}

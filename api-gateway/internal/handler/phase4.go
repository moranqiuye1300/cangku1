package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
)

type postBarrageReq struct {
	Content string `json:"content" binding:"required,min=1,max=100"`
	TimeMs  int32  `json:"time_ms"`
}

func (h *Handler) ListBarrages(c *gin.Context) {
	videoID := c.Param("id")
	resp, err := h.videoClient.ListBarrages(h.ctx(c), &videopb.ListBarragesRequest{VideoId: videoID})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"barrages": resp.GetBarrages()})
}

func (h *Handler) PostBarrage(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	uid, _ := userID.(string)
	uname, _ := username.(string)
	if uid == "" {
		response.Fail(c, 401, 40100, "unauthorized")
		return
	}
	var req postBarrageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	resp, err := h.videoClient.PostBarrage(h.ctx(c), &videopb.PostBarrageRequest{
		VideoId:  videoID,
		UserId:   uid,
		Username: uname,
		Content:  req.Content,
		TimeMs:   req.TimeMs,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"barrage": resp.GetBarrage()})
}

type aiAskReq struct {
	Question string `json:"question" binding:"required,min=2,max=500"`
}

func (h *Handler) AIAsk(c *gin.Context) {
	var req aiAskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, 40001, err.Error())
		return
	}
	base := strings.TrimRight(os.Getenv("AI_SERVICE_URL"), "/")
	if base == "" {
		base = "http://ai-service:8090"
	}
	payload, _ := json.Marshal(map[string]string{"question": req.Question})
	httpReq, err := http.NewRequest(http.MethodPost, base+"/rag/ask", bytes.NewReader(payload))
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		response.Fail(c, 503, 50300, "ai service unavailable")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		response.Fail(c, 503, 50300, string(body))
		return
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		response.OK(c, gin.H{"raw": string(body)})
		return
	}
	response.OK(c, data)
}

func (h *Handler) ctx(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		ctx = auth.WithBearer(ctx, strings.TrimPrefix(authHeader, "Bearer "))
	}
	return ctx
}

package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/gen/videopb"
)

type postBarrageReq struct {
	Content string `json:"content" binding:"required,min=1,max=100"`
	TimeMs  int32  `json:"time_ms"`
}

func (h *Handler) ListBarrages(c *gin.Context) {
	videoID := c.Param("id")
	page, pageSize := pageQuery(c)
	resp, err := h.videoClient.ListBarrages(h.ctx(c), &videopb.ListBarragesRequest{
		VideoId:  videoID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"barrages": resp.GetBarrages(), "total": resp.GetTotal()})
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
	Question        string   `json:"question" binding:"required,min=2,max=500"`
	ContextVideoIDs []string `json:"context_video_ids"` // from Feed personalized list
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

	payload := map[string]any{
		"question":          req.Question,
		"context_video_ids": req.ContextVideoIDs,
	}
	bodyBytes, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest(http.MethodPost, base+"/rag/ask", bytes.NewReader(bodyBytes))
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
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

package aitag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type generateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type generateResponse struct {
	Tags []string `json:"tags"`
}

func NewFromEnv() *Client {
	base := strings.TrimRight(os.Getenv("AI_SERVICE_URL"), "/")
	if base == "" {
		base = "http://ai-service:8000"
	}
	return &Client{
		baseURL: base,
		http:    &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Client) Generate(ctx context.Context, title, description string) ([]string, error) {
	body, _ := json.Marshal(generateRequest{Title: title, Description: description})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/tags/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai tag status %d", resp.StatusCode)
	}
	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Tags, nil
}

package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/gen/userpb"
)

func (h *Handler) OAuthGitHubURL(c *gin.Context) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirect := os.Getenv("GITHUB_REDIRECT_URL")
	if clientID == "" || redirect == "" {
		response.OK(c, gin.H{
			"provider": "github",
			"mock":     true,
			"login_url": "/api/v1/auth/oauth/mock?provider=github&oauth_id=demo_user",
		})
		return
	}
	u := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user",
		url.QueryEscape(clientID),
		url.QueryEscape(redirect),
	)
	response.OK(c, gin.H{"provider": "github", "url": u})
}

func (h *Handler) OAuthGitHubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Fail(c, 400, 40001, "code required")
		return
	}
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		response.Fail(c, 503, 50300, "github oauth not configured, use /api/v1/auth/oauth/mock")
		return
	}
	token, err := exchangeGitHubToken(code, clientID, clientSecret)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	profile, err := fetchGitHubUser(token)
	if err != nil {
		response.Fail(c, 500, 50000, err.Error())
		return
	}
	h.finishOAuth(c, "github", fmt.Sprintf("%d", profile.ID), profile.Name, profile.AvatarURL)
}

func (h *Handler) OAuthMock(c *gin.Context) {
	provider := c.DefaultQuery("provider", "github")
	oauthID := c.DefaultQuery("oauth_id", "demo_user")
	nickname := c.DefaultQuery("nickname", "Demo User")
	avatar := c.DefaultQuery("avatar", "")
	h.finishOAuth(c, provider, oauthID, nickname, avatar)
}

func (h *Handler) finishOAuth(c *gin.Context, provider, oauthID, nickname, avatar string) {
	resp, err := h.userClient.OAuthLogin(h.ctx(c), &userpb.OAuthLoginRequest{
		Provider: provider,
		OauthId:  oauthID,
		Nickname: nickname,
		Avatar:   avatar,
	})
	if err != nil {
		h.writeGRPCError(c, err)
		return
	}
	response.OK(c, gin.H{"user": resp.GetUser(), "token": resp.GetToken()})
}

type gitHubTokenResp struct {
	AccessToken string `json:"access_token"`
}

type gitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func exchangeGitHubToken(code, clientID, clientSecret string) (string, error) {
	body := url.Values{}
	body.Set("client_id", clientID)
	body.Set("client_secret", clientSecret)
	body.Set("code", code)
	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(body.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out gitHubTokenResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("github token empty")
	}
	return out.AccessToken, nil
}

func fetchGitHubUser(token string) (*gitHubUser, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var u gitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	if u.Name == "" {
		u.Name = u.Login
	}
	return &u, nil
}

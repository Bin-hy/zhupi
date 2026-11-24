package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const GitHubIssuer = "https://github.com"

// ExternalUser 是任一登录提供商返回的规范化用户信息。
type ExternalUser struct {
	Subject       string
	Handle        string
	Name          string
	Email         string
	EmailVerified bool
	AvatarURL     string
}

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	APIURL       string
}

type GitHubClient struct {
	conf       *oauth2.Config
	apiURL     string
	httpClient *http.Client
}

func NewGitHubClient(cfg GitHubConfig, hc *http.Client) *GitHubClient {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = "https://github.com/login/oauth/authorize"
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://github.com/login/oauth/access_token"
	}
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.github.com/user"
	}
	return &GitHubClient{
		conf: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL},
			Scopes:       []string{"read:user"},
		},
		apiURL:     apiURL,
		httpClient: hc,
	}
}

func (g *GitHubClient) Configured() bool {
	return g.conf.ClientID != "" && g.conf.ClientSecret != ""
}

func (g *GitHubClient) AuthCodeURL(state string) string {
	return g.conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (g *GitHubClient) Exchange(ctx context.Context, code string) (ExternalUser, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, g.httpClient)
	token, err := g.conf.Exchange(ctx, code)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("GitHub 换取 access token 失败: %w", err)
	}
	return g.fetchUser(ctx, token.AccessToken)
}

func (g *GitHubClient) fetchUser(ctx context.Context, accessToken string) (ExternalUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.apiURL, nil)
	if err != nil {
		return ExternalUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "annotation-service")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("请求 GitHub 用户信息失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ExternalUser{}, fmt.Errorf("GitHub 用户信息返回 %d", resp.StatusCode)
	}
	var payload struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ExternalUser{}, fmt.Errorf("解析 GitHub 用户信息失败: %w", err)
	}
	if payload.ID == 0 || payload.Login == "" {
		return ExternalUser{}, fmt.Errorf("GitHub 用户信息缺少 id/login")
	}
	name := payload.Name
	if name == "" {
		name = payload.Login
	}
	return ExternalUser{
		Subject:   strconv.FormatInt(payload.ID, 10),
		Handle:    payload.Login,
		Name:      name,
		AvatarURL: payload.AvatarURL,
	}, nil
}

func NormalizeIssuer(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/")
}

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type OIDCClient struct {
	httpClient *http.Client

	mu        sync.Mutex
	providers map[string]*oidc.Provider
}

func NewOIDCClient(hc *http.Client) *OIDCClient {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	return &OIDCClient{httpClient: hc, providers: map[string]*oidc.Provider{}}
}

func (c *OIDCClient) provider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	issuer = NormalizeIssuer(issuer)
	if issuer == "" {
		return nil, fmt.Errorf("issuer 不能为空")
	}
	c.mu.Lock()
	if p, ok := c.providers[issuer]; ok {
		c.mu.Unlock()
		return p, nil
	}
	c.mu.Unlock()

	ctx = oidc.ClientContext(ctx, c.httpClient)
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC Discovery 失败: %w", err)
	}
	c.mu.Lock()
	c.providers[issuer] = p
	c.mu.Unlock()
	return p, nil
}

func (c *OIDCClient) Invalidate(issuer string) {
	c.mu.Lock()
	delete(c.providers, NormalizeIssuer(issuer))
	c.mu.Unlock()
}

// Validate 在保存提供商时做一次 Discovery 校验。
func (c *OIDCClient) Validate(ctx context.Context, issuer string) error {
	_, err := c.provider(ctx, issuer)
	return err
}

func (c *OIDCClient) oauth2Config(p *oidc.Provider, cfg OIDCConfig) *oauth2.Config {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       scopes,
	}
}

func (c *OIDCClient) AuthCodeURL(ctx context.Context, cfg OIDCConfig, state, nonce, verifier string) (string, error) {
	p, err := c.provider(ctx, cfg.Issuer)
	if err != nil {
		return "", err
	}
	conf := c.oauth2Config(p, cfg)
	return conf.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

func (c *OIDCClient) Exchange(ctx context.Context, cfg OIDCConfig, code, verifier, expectedNonce string) (ExternalUser, error) {
	p, err := c.provider(ctx, cfg.Issuer)
	if err != nil {
		return ExternalUser{}, err
	}
	ctx = oidc.ClientContext(ctx, c.httpClient)
	conf := c.oauth2Config(p, cfg)
	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return ExternalUser{}, fmt.Errorf("OIDC 换取 token 失败: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return ExternalUser{}, fmt.Errorf("OIDC 响应缺少 id_token")
	}
	idToken, err := p.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return ExternalUser{}, fmt.Errorf("OIDC id_token 校验失败: %w", err)
	}
	if expectedNonce != "" && idToken.Nonce != expectedNonce {
		return ExternalUser{}, fmt.Errorf("OIDC nonce 不匹配")
	}
	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return ExternalUser{}, fmt.Errorf("解析 OIDC claims 失败: %w", err)
	}
	handle := claims.PreferredUsername
	if handle == "" {
		handle = claims.Email
	}
	name := claims.Name
	if name == "" {
		name = handle
	}
	return ExternalUser{
		Subject:       idToken.Subject,
		Handle:        handle,
		Name:          name,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		AvatarURL:     claims.Picture,
	}, nil
}

func scopesOrDefault(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{oidc.ScopeOpenID, "email", "profile"}
	}
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

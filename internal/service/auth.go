package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/crypto"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

type ClientMeta struct {
	UserAgent string
	IP        string
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      domain.User
	ReturnTo  string
	Mode      domain.FlowMode
	SiteID    int64
}

type AuthService struct {
	cfg        *config.Config
	users      store.UserRepo
	identities store.IdentityRepo
	sessions   store.SessionRepo
	flows      store.AuthFlowRepo
	providers  store.ProviderRepo
	localCreds store.LocalCredentialRepo
	box        *crypto.SecretBox
	github     *authpkg.GitHubClient
	oidc       *authpkg.OIDCClient
	now        func() time.Time
}

func NewAuthService(cfg *config.Config, users store.UserRepo, identities store.IdentityRepo,
	sessions store.SessionRepo, flows store.AuthFlowRepo, providers store.ProviderRepo,
	localCreds store.LocalCredentialRepo, box *crypto.SecretBox,
	github *authpkg.GitHubClient, oidc *authpkg.OIDCClient) *AuthService {
	return &AuthService{
		cfg: cfg, users: users, identities: identities, sessions: sessions,
		flows: flows, providers: providers, localCreds: localCreds,
		box: box, github: github, oidc: oidc,
		now: time.Now,
	}
}

// LocalAdminBootstrap 描述启动时本地管理员初始化的结果。
type LocalAdminBootstrap struct {
	Created            bool
	Reset              bool
	Skipped            bool
	Username           string
	MustChangePassword bool
}

// BootstrapLocalAdmin 在启动时保证存在一个本地超级管理员账号。
// 默认只在没有任何本地凭据时创建；LOCAL_ADMIN_RESET_PASSWORD=1 时强制重置密码。
func (a *AuthService) BootstrapLocalAdmin(ctx context.Context) (LocalAdminBootstrap, error) {
	username := strings.TrimSpace(a.cfg.LocalAdminUsername)
	if username == "" {
		username = "admin"
	}
	password := a.cfg.LocalAdminPassword
	if password == "" {
		password = "admin"
	}
	mustChange := password == "admin"

	count, err := a.localCreds.CountLocalCredentials(ctx)
	if err != nil {
		return LocalAdminBootstrap{}, err
	}
	if count == 0 {
		if password != "admin" {
			if err := authpkg.ValidatePassword(password); err != nil {
				return LocalAdminBootstrap{}, domain.WrapError("local_admin_password_invalid", "本地管理员初始密码不符合要求", 500, err)
			}
		}
		hash, err := authpkg.HashPassword(password)
		if err != nil {
			return LocalAdminBootstrap{}, domain.WrapError("local_admin_password_invalid", "本地管理员初始密码不符合要求", 500, err)
		}
		user, err := a.users.CreateUser(ctx, domain.User{
			DisplayName:  username,
			Status:       domain.UserActive,
			IsSuperadmin: true,
		})
		if err != nil {
			return LocalAdminBootstrap{}, err
		}
		if err := a.localCreds.CreateLocalCredential(ctx, domain.LocalCredential{
			UserID:             user.ID,
			Username:           username,
			PasswordHash:       hash,
			MustChangePassword: mustChange,
		}); err != nil {
			return LocalAdminBootstrap{}, err
		}
		return LocalAdminBootstrap{
			Created: true, Username: username, MustChangePassword: mustChange,
		}, nil
	}

	cred, err := a.localCreds.LocalCredentialByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		// 已存在本地账号但环境变量用户名不同：不创建第二个，避免误建后门账号。
		return LocalAdminBootstrap{Skipped: true}, nil
	}
	if err != nil {
		return LocalAdminBootstrap{}, err
	}
	if !a.cfg.LocalAdminResetPassword {
		return LocalAdminBootstrap{Skipped: true, Username: cred.Username}, nil
	}
	if password != "admin" {
		if err := authpkg.ValidatePassword(password); err != nil {
			return LocalAdminBootstrap{}, domain.WrapError("local_admin_password_invalid", "LOCAL_ADMIN_PASSWORD 不符合要求", 500, err)
		}
	}
	hash, err := authpkg.HashPassword(password)
	if err != nil {
		return LocalAdminBootstrap{}, domain.WrapError("local_admin_password_invalid", "LOCAL_ADMIN_PASSWORD 不符合要求", 500, err)
	}
	if err := a.localCreds.UpdateLocalCredentialPassword(ctx, cred.UserID, hash, mustChange); err != nil {
		return LocalAdminBootstrap{}, err
	}
	if err := a.sessions.RevokeAllSessionsForUser(ctx, cred.UserID); err != nil {
		return LocalAdminBootstrap{}, err
	}
	return LocalAdminBootstrap{
		Reset: true, Username: cred.Username, MustChangePassword: mustChange,
	}, nil
}

// LocalLoginResult 是本地账号密码登录的结果。
type LocalLoginResult struct {
	Token              string
	ExpiresAt          time.Time
	User               domain.User
	MustChangePassword bool
}

// LocalLogin 校验本地管理员账号密码并创建会话。
func (a *AuthService) LocalLogin(ctx context.Context, username, password string, meta ClientMeta) (LocalLoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return LocalLoginResult{}, domain.Unauthorized("用户名或密码错误")
	}
	cred, err := a.localCreds.LocalCredentialByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		authpkg.VerifyDummy(password)
		return LocalLoginResult{}, domain.Unauthorized("用户名或密码错误")
	}
	if err != nil {
		return LocalLoginResult{}, err
	}
	if !authpkg.VerifyPassword(cred.PasswordHash, password) {
		return LocalLoginResult{}, domain.Unauthorized("用户名或密码错误")
	}
	user, err := a.users.UserByID(ctx, cred.UserID)
	if err != nil {
		return LocalLoginResult{}, err
	}
	if user.Status == domain.UserBanned {
		return LocalLoginResult{}, domain.Forbidden("user_banned", "账号已被封禁")
	}
	raw, hash, err := authpkg.NewToken()
	if err != nil {
		return LocalLoginResult{}, err
	}
	now := a.now()
	expires := now.Add(a.cfg.SessionTTL)
	if err := a.sessions.CreateSession(ctx, domain.Session{
		TokenHash:  hash,
		UserID:     user.ID,
		UserAgent:  truncate(meta.UserAgent, 255),
		IP:         truncate(meta.IP, 64),
		CreatedAt:  now,
		ExpiresAt:  expires,
		LastSeenAt: now,
	}); err != nil {
		return LocalLoginResult{}, err
	}
	return LocalLoginResult{
		Token: raw, ExpiresAt: expires, User: user, MustChangePassword: cred.MustChangePassword,
	}, nil
}

// ChangePassword 修改本地账号密码，并吊销除当前会话外的其他会话。
func (a *AuthService) ChangePassword(ctx context.Context, userID, currentSessionID int64, current, next string) error {
	cred, err := a.localCreds.LocalCredentialByUserID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Invalid("local_credential_missing", "当前账号未设置本地密码")
	}
	if err != nil {
		return err
	}
	if !authpkg.VerifyPassword(cred.PasswordHash, current) {
		return domain.Invalid("invalid_current_password", "当前密码不正确")
	}
	if current == next {
		return domain.Invalid("password_unchanged", "新密码不能与当前密码相同")
	}
	if err := authpkg.ValidatePassword(next); err != nil {
		return domain.Invalid("weak_password", err.Error())
	}
	hash, err := authpkg.HashPassword(next)
	if err != nil {
		return err
	}
	if err := a.localCreds.UpdateLocalCredentialPassword(ctx, userID, hash, false); err != nil {
		return err
	}
	if currentSessionID != 0 {
		return a.sessions.RevokeOtherSessionsForUser(ctx, userID, currentSessionID)
	}
	return a.sessions.RevokeAllSessionsForUser(ctx, userID)
}

// ResetLocalPassword 供 CLI 使用：重置本地账号密码并吊销全部会话。
func (a *AuthService) ResetLocalPassword(ctx context.Context, username, password string) (domain.LocalCredential, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.LocalCredential{}, domain.Invalid("validation_error", "需要用户名")
	}
	cred, err := a.localCreds.LocalCredentialByUsername(ctx, username)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.LocalCredential{}, domain.NotFound("本地管理员账号不存在，请先启动一次服务完成初始化")
	}
	if err != nil {
		return domain.LocalCredential{}, err
	}
	if password != "admin" {
		if err := authpkg.ValidatePassword(password); err != nil {
			return domain.LocalCredential{}, domain.Invalid("weak_password", err.Error())
		}
	}
	hash, err := authpkg.HashPassword(password)
	if err != nil {
		return domain.LocalCredential{}, err
	}
	if err := a.localCreds.UpdateLocalCredentialPassword(ctx, cred.UserID, hash, password == "admin"); err != nil {
		return domain.LocalCredential{}, err
	}
	if err := a.sessions.RevokeAllSessionsForUser(ctx, cred.UserID); err != nil {
		return domain.LocalCredential{}, err
	}
	cred.MustChangePassword = password == "admin"
	return cred, nil
}

func (a *AuthService) StartGitHub(ctx context.Context, site domain.Site, returnTo string) (string, error) {
	return a.startGitHub(ctx, site, returnTo, domain.FlowLogin, 0)
}

func (a *AuthService) StartBindGitHub(ctx context.Context, userID int64, site domain.Site, returnTo string) (string, error) {
	if _, err := a.users.UserByID(ctx, userID); err != nil {
		return "", domain.Unauthorized("用户不存在")
	}
	return a.startGitHub(ctx, site, returnTo, domain.FlowBind, userID)
}

func (a *AuthService) startGitHub(ctx context.Context, site domain.Site, returnTo string, mode domain.FlowMode, userID int64) (string, error) {
	if a.github == nil || !a.github.Configured() {
		return "", domain.NewError("github_not_configured", "服务端未配置 GitHub 登录", 503)
	}
	state, hash, err := newState()
	if err != nil {
		return "", err
	}
	flow := domain.AuthFlow{
		StateHash:    hash,
		Mode:         mode,
		ProviderType: domain.ProviderGitHub,
		ProviderID:   0,
		Issuer:       authpkg.GitHubIssuer,
		SiteID:       site.ID,
		UserID:       userID,
		ReturnTo:     safeReturnTo(site, returnTo),
		ExpiresAt:    a.now().Add(10 * time.Minute),
	}
	if err := a.flows.CreateAuthFlow(ctx, flow); err != nil {
		return "", err
	}
	return a.github.AuthCodeURL(state), nil
}

type AdminProviderView struct {
	ID       int64  `json:"id"`
	SiteID   int64  `json:"siteId"`
	SiteName string `json:"siteName"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer"`
}

type AdminAuthProviders struct {
	GitHub    bool                `json:"github"`
	Local     bool                `json:"local"`
	Providers []AdminProviderView `json:"providers"`
}

func (a *AuthService) AdminProviders(ctx context.Context) (AdminAuthProviders, error) {
	list, err := a.providers.ListEnabledWithSite(ctx)
	if err != nil {
		return AdminAuthProviders{}, err
	}
	out := AdminAuthProviders{
		GitHub:    a.github != nil && a.github.Configured(),
		Providers: make([]AdminProviderView, 0, len(list)),
	}
	if a.localCreds != nil {
		if n, err := a.localCreds.CountLocalCredentials(ctx); err == nil && n > 0 {
			out.Local = true
		}
	}
	for _, item := range list {
		out.Providers = append(out.Providers, AdminProviderView{
			ID: item.Provider.ID, SiteID: item.SiteID, SiteName: item.SiteName,
			Name: item.Provider.Name, Issuer: item.Provider.Issuer,
		})
	}
	return out, nil
}

func (a *AuthService) StartAdminGitHub(ctx context.Context, returnTo string) (string, error) {
	if a.github == nil || !a.github.Configured() {
		return "", domain.NewError("github_not_configured", "服务端未配置 GitHub 登录", 503)
	}
	state, hash, err := newState()
	if err != nil {
		return "", err
	}
	flow := domain.AuthFlow{
		StateHash:    hash,
		Mode:         domain.FlowLogin,
		ProviderType: domain.ProviderGitHub,
		Issuer:       authpkg.GitHubIssuer,
		SiteID:       0,
		ReturnTo:     safeAdminReturnTo(a.cfg.PublicBaseURL, returnTo),
		ExpiresAt:    a.now().Add(10 * time.Minute),
	}
	if err := a.flows.CreateAuthFlow(ctx, flow); err != nil {
		return "", err
	}
	return a.github.AuthCodeURL(state), nil
}

func (a *AuthService) StartAdminOIDC(ctx context.Context, providerID int64, returnTo string) (string, error) {
	provider, err := a.providers.ProviderByIDAny(ctx, providerID)
	if err != nil {
		return "", domain.NotFound("登录提供商不存在")
	}
	if !provider.Enabled {
		return "", domain.Forbidden("provider_disabled", "登录提供商已停用")
	}
	cfg, _, err := a.oidcConfig(provider)
	if err != nil {
		return "", err
	}
	state, hash, err := newState()
	if err != nil {
		return "", err
	}
	verifier, err := randomBase64(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomHex(16)
	if err != nil {
		return "", err
	}
	flow := domain.AuthFlow{
		StateHash:    hash,
		Mode:         domain.FlowLogin,
		ProviderType: domain.ProviderOIDC,
		ProviderID:   provider.ID,
		Issuer:       authpkg.NormalizeIssuer(provider.Issuer),
		SiteID:       provider.SiteID,
		ReturnTo:     safeAdminReturnTo(a.cfg.PublicBaseURL, returnTo),
		CodeVerifier: verifier,
		Nonce:        nonce,
		ExpiresAt:    a.now().Add(10 * time.Minute),
	}
	if err := a.flows.CreateAuthFlow(ctx, flow); err != nil {
		return "", err
	}
	return a.oidc.AuthCodeURL(ctx, cfg, state, nonce, verifier)
}

func safeAdminReturnTo(publicBaseURL, raw string) string {
	fallback := strings.TrimRight(publicBaseURL, "/") + "/admin/"
	base, err := url.Parse(publicBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return fallback
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fallback
	}
	if !strings.EqualFold(u.Scheme, base.Scheme) || !strings.EqualFold(u.Host, base.Host) {
		return fallback
	}
	if u.Path != "/admin" && !strings.HasPrefix(u.Path, "/admin/") {
		return fallback
	}
	u.Fragment = ""
	return u.String()
}

func (a *AuthService) StartOIDC(ctx context.Context, providerID int64, site domain.Site, returnTo string) (string, error) {
	return a.startOIDC(ctx, providerID, site, returnTo, domain.FlowLogin, 0)
}

func (a *AuthService) StartBindOIDC(ctx context.Context, userID, providerID int64, site domain.Site, returnTo string) (string, error) {
	if _, err := a.users.UserByID(ctx, userID); err != nil {
		return "", domain.Unauthorized("用户不存在")
	}
	return a.startOIDC(ctx, providerID, site, returnTo, domain.FlowBind, userID)
}

func (a *AuthService) startOIDC(ctx context.Context, providerID int64, site domain.Site, returnTo string, mode domain.FlowMode, userID int64) (string, error) {
	provider, err := a.providers.ProviderByID(ctx, site.ID, providerID)
	if err != nil {
		return "", domain.NotFound("登录提供商不存在")
	}
	if !provider.Enabled {
		return "", domain.Forbidden("provider_disabled", "登录提供商已停用")
	}
	cfg, _, err := a.oidcConfig(provider)
	if err != nil {
		return "", err
	}
	state, hash, err := newState()
	if err != nil {
		return "", err
	}
	verifier, err := randomBase64(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomHex(16)
	if err != nil {
		return "", err
	}
	flow := domain.AuthFlow{
		StateHash:    hash,
		Mode:         mode,
		ProviderType: domain.ProviderOIDC,
		ProviderID:   provider.ID,
		Issuer:       authpkg.NormalizeIssuer(provider.Issuer),
		SiteID:       site.ID,
		UserID:       userID,
		ReturnTo:     safeReturnTo(site, returnTo),
		CodeVerifier: verifier,
		Nonce:        nonce,
		ExpiresAt:    a.now().Add(10 * time.Minute),
	}
	if err := a.flows.CreateAuthFlow(ctx, flow); err != nil {
		return "", err
	}
	return a.oidc.AuthCodeURL(ctx, cfg, state, nonce, verifier)
}

func (a *AuthService) FinishGitHub(ctx context.Context, code, state string, meta ClientMeta) (LoginResult, error) {
	flow, err := a.consumeFlow(ctx, state)
	if err != nil {
		return LoginResult{}, err
	}
	if flow.ProviderType != domain.ProviderGitHub {
		return LoginResult{}, domain.Invalid("invalid_flow", "登录流程类型不匹配")
	}
	ext, err := a.github.Exchange(ctx, code)
	if err != nil {
		return LoginResult{}, domain.NewError("github_exchange_failed", "GitHub 登录失败，请重试", 502)
	}
	return a.completeLogin(ctx, flow, ext, meta)
}

func (a *AuthService) FinishOIDC(ctx context.Context, providerID int64, code, state string, meta ClientMeta) (LoginResult, error) {
	flow, err := a.consumeFlow(ctx, state)
	if err != nil {
		return LoginResult{}, err
	}
	if flow.ProviderType != domain.ProviderOIDC || flow.ProviderID != providerID {
		return LoginResult{}, domain.Invalid("invalid_flow", "登录流程与提供商不匹配")
	}
	provider, err := a.providers.ProviderByID(ctx, flow.SiteID, providerID)
	if err != nil {
		return LoginResult{}, domain.NotFound("登录提供商不存在")
	}
	cfg, _, err := a.oidcConfig(provider)
	if err != nil {
		return LoginResult{}, err
	}
	ext, err := a.oidc.Exchange(ctx, cfg, code, flow.CodeVerifier, flow.Nonce)
	if err != nil {
		return LoginResult{}, domain.NewError("oidc_exchange_failed", "OIDC 登录失败，请重试", 502)
	}
	if len(provider.AllowedEmailDomains) > 0 && !emailDomainAllowed(ext.Email, provider.AllowedEmailDomains) {
		return LoginResult{}, domain.Forbidden("email_domain_not_allowed", "该邮箱域不允许登录此站点")
	}
	return a.completeLogin(ctx, flow, ext, meta)
}

func (a *AuthService) completeLogin(ctx context.Context, flow domain.AuthFlow, ext authpkg.ExternalUser, meta ClientMeta) (LoginResult, error) {
	if flow.Mode == domain.FlowBind {
		if err := a.bindIdentity(ctx, flow, ext); err != nil {
			return LoginResult{}, err
		}
		user, err := a.users.UserByID(ctx, flow.UserID)
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{User: user, ReturnTo: flow.ReturnTo, Mode: domain.FlowBind, SiteID: flow.SiteID}, nil
	}

	user, err := a.loginOrCreate(ctx, flow, ext)
	if err != nil {
		return LoginResult{}, err
	}
	raw, hash, err := authpkg.NewToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := a.now()
	expires := now.Add(a.cfg.SessionTTL)
	if err := a.sessions.CreateSession(ctx, domain.Session{
		TokenHash:  hash,
		UserID:     user.ID,
		UserAgent:  truncate(meta.UserAgent, 255),
		IP:         truncate(meta.IP, 64),
		CreatedAt:  now,
		ExpiresAt:  expires,
		LastSeenAt: now,
	}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: raw, ExpiresAt: expires, User: user, ReturnTo: flow.ReturnTo, Mode: domain.FlowLogin, SiteID: flow.SiteID}, nil
}

func (a *AuthService) loginOrCreate(ctx context.Context, flow domain.AuthFlow, ext authpkg.ExternalUser) (domain.User, error) {
	provider, siteID, issuer := flowIdentityParts(flow)
	identity, err := a.identities.IdentityByProviderSubject(ctx, provider, siteID, issuer, ext.Subject)
	switch {
	case err == nil:
		_ = a.identities.TouchIdentityLogin(ctx, identity.ID, a.now().Unix())
		return a.users.UserByID(ctx, identity.UserID)
	case errors.Is(err, domain.ErrNotFound):
		user, err := a.users.CreateUser(ctx, domain.User{
			DisplayName: truncate(ext.Name, 100),
			AvatarURL:   ext.AvatarURL,
			Status:      domain.UserActive,
		})
		if err != nil {
			return domain.User{}, err
		}
		if _, err := a.identities.CreateIdentity(ctx, domain.Identity{
			UserID:        user.ID,
			Provider:      provider,
			SiteID:        siteID,
			Issuer:        issuer,
			Subject:       ext.Subject,
			Handle:        ext.Handle,
			Email:         ext.Email,
			EmailVerified: ext.EmailVerified,
			AvatarURL:     ext.AvatarURL,
		}); err != nil {
			return domain.User{}, err
		}
		return a.applyAutoSuperadmin(ctx, provider, ext, user)
	default:
		return domain.User{}, err
	}
}

func (a *AuthService) bindIdentity(ctx context.Context, flow domain.AuthFlow, ext authpkg.ExternalUser) error {
	provider, siteID, issuer := flowIdentityParts(flow)
	existing, err := a.identities.IdentityByProviderSubject(ctx, provider, siteID, issuer, ext.Subject)
	if err == nil {
		if existing.UserID != flow.UserID {
			return domain.Conflict("identity_already_bound", "该登录身份已绑定到其他账号")
		}
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	_, err = a.identities.CreateIdentity(ctx, domain.Identity{
		UserID:        flow.UserID,
		Provider:      provider,
		SiteID:        siteID,
		Issuer:        issuer,
		Subject:       ext.Subject,
		Handle:        ext.Handle,
		Email:         ext.Email,
		EmailVerified: ext.EmailVerified,
		AvatarURL:     ext.AvatarURL,
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AuthService) applyAutoSuperadmin(ctx context.Context, provider domain.ProviderType, ext authpkg.ExternalUser, user domain.User) (domain.User, error) {
	if provider != domain.ProviderGitHub {
		return user, nil
	}
	if !a.cfg.SuperadminGitHubLogins[strings.ToLower(ext.Handle)] {
		return user, nil
	}
	if err := a.users.SetUserSuperadmin(ctx, user.ID, true); err != nil {
		return domain.User{}, err
	}
	user.IsSuperadmin = true
	return user, nil
}

func flowIdentityParts(flow domain.AuthFlow) (domain.ProviderType, int64, string) {
	if flow.ProviderType == domain.ProviderGitHub {
		return domain.ProviderGitHub, 0, authpkg.GitHubIssuer
	}
	if flow.ProviderType == domain.ProviderDev {
		return domain.ProviderDev, 0, "dev"
	}
	return domain.ProviderOIDC, flow.SiteID, flow.Issuer
}

func (a *AuthService) consumeFlow(ctx context.Context, state string) (domain.AuthFlow, error) {
	if strings.TrimSpace(state) == "" {
		return domain.AuthFlow{}, domain.Invalid("invalid_state", "缺少登录状态")
	}
	flow, err := a.flows.ConsumeAuthFlow(ctx, authpkg.HashToken(state))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.AuthFlow{}, domain.Invalid("invalid_state", "登录状态无效或已使用")
	}
	return flow, err
}

func (a *AuthService) oidcConfig(p domain.OIDCProvider) (authpkg.OIDCConfig, string, error) {
	secret, err := a.box.Open(p.ClientSecretCipher)
	if err != nil {
		return authpkg.OIDCConfig{}, "", domain.WrapError("provider_secret_error", "读取 OIDC 密钥失败", 500, err)
	}
	return authpkg.OIDCConfig{
		Issuer:       authpkg.NormalizeIssuer(p.Issuer),
		ClientID:     p.ClientID,
		ClientSecret: string(secret),
		RedirectURL:  a.oidcRedirectURL(p.ID),
		Scopes:       p.Scopes,
	}, string(secret), nil
}

func (a *AuthService) oidcRedirectURL(providerID int64) string {
	return strings.TrimRight(a.cfg.PublicBaseURL, "/") +
		"/api/v1/auth/oidc/" + strconv.FormatInt(providerID, 10) + "/callback"
}

func (a *AuthService) Authenticate(ctx context.Context, rawToken string) (domain.Principal, error) {
	p, _, err := a.AuthenticateSession(ctx, rawToken)
	return p, err
}

func (a *AuthService) AuthenticateSession(ctx context.Context, rawToken string) (domain.Principal, domain.Session, error) {
	if strings.TrimSpace(rawToken) == "" {
		return domain.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	sess, err := a.sessions.SessionByTokenHash(ctx, authpkg.HashToken(rawToken))
	if err != nil {
		return domain.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	if sess.RevokedAt != nil || !a.now().Before(sess.ExpiresAt) {
		return domain.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	user, err := a.users.UserByID(ctx, sess.UserID)
	if err != nil {
		return domain.Principal{}, domain.Session{}, domain.ErrUnauthorized
	}
	if user.Status == domain.UserBanned {
		return domain.Principal{}, domain.Session{}, domain.Unauthorized("账号已被封禁")
	}
	if a.now().Sub(sess.LastSeenAt) > 5*time.Minute {
		_ = a.sessions.TouchSession(ctx, sess.ID, a.now().Unix())
	}
	p := domain.Principal{User: user, IsSuperadmin: user.IsSuperadmin}
	if a.localCreds != nil {
		cred, err := a.localCreds.LocalCredentialByUserID(ctx, user.ID)
		switch {
		case err == nil:
			p.LocalUsername = cred.Username
			p.MustChangePassword = cred.MustChangePassword
		case errors.Is(err, domain.ErrNotFound):
			// 非本地账号，忽略。
		default:
			return domain.Principal{}, domain.Session{}, err
		}
	}
	return p, sess, nil
}

func (a *AuthService) Logout(ctx context.Context, userID, sessionID int64) error {
	return a.sessions.RevokeSession(ctx, sessionID, userID)
}

func (a *AuthService) ListIdentities(ctx context.Context, userID int64) ([]domain.Identity, error) {
	return a.identities.ListIdentitiesByUser(ctx, userID)
}

func (a *AuthService) UnbindIdentity(ctx context.Context, userID, identityID int64) error {
	list, err := a.identities.ListIdentitiesByUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(list) <= 1 {
		return domain.Invalid("last_identity", "至少需要保留一个登录方式")
	}
	found := false
	for _, item := range list {
		if item.ID == identityID {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	return a.identities.DeleteIdentity(ctx, identityID, userID)
}

func (a *AuthService) ListSessions(ctx context.Context, userID int64) ([]domain.Session, error) {
	return a.sessions.ListSessionsByUser(ctx, userID)
}

func (a *AuthService) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	return a.sessions.RevokeSession(ctx, sessionID, userID)
}

func (a *AuthService) RevokeAllSessions(ctx context.Context, userID int64) error {
	return a.sessions.RevokeAllSessionsForUser(ctx, userID)
}

func (a *AuthService) EncryptSecret(plain string) ([]byte, error) {
	return a.box.Seal([]byte(plain))
}

func (a *AuthService) ValidateIssuer(ctx context.Context, issuer string) error {
	return a.oidc.Validate(ctx, issuer)
}

func (a *AuthService) DevLogin(ctx context.Context, name string, meta ClientMeta) (LoginResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return LoginResult{}, domain.Invalid("validation_error", "调试用户名不能为空")
	}
	if len(name) > 64 {
		return LoginResult{}, domain.Invalid("validation_error", "调试用户名过长")
	}
	ext := authpkg.ExternalUser{Subject: name, Handle: name, Name: name}
	flow := domain.AuthFlow{ProviderType: domain.ProviderDev, Issuer: "dev", SiteID: 0}
	user, err := a.loginOrCreate(ctx, flow, ext)
	if err != nil {
		return LoginResult{}, err
	}
	raw, hash, err := authpkg.NewToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := a.now()
	expires := now.Add(a.cfg.SessionTTL)
	if err := a.sessions.CreateSession(ctx, domain.Session{
		TokenHash: hash, UserID: user.ID, UserAgent: truncate(meta.UserAgent, 255),
		IP: truncate(meta.IP, 64), CreatedAt: now, ExpiresAt: expires, LastSeenAt: now,
	}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: raw, ExpiresAt: expires, User: user, Mode: domain.FlowLogin}, nil
}

func safeReturnTo(site domain.Site, raw string) string {
	fallback := "/"
	if len(site.Origins) > 0 {
		fallback = site.Origins[0] + "/"
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fallback
	}
	origin := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
	for _, allowed := range site.Origins {
		if strings.EqualFold(allowed, origin) {
			u.Fragment = ""
			return u.String()
		}
	}
	return fallback
}

func emailDomainAllowed(email string, allowed []string) bool {
	if email == "" {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domainPart := strings.ToLower(email[at+1:])
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(item), domainPart) {
			return true
		}
	}
	return false
}

func newState() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	return raw, authpkg.HashToken(raw), nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func randomBase64(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

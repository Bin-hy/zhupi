package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/ratelimit"
	"github.com/Bin-hy/zhupi/internal/service"
	"github.com/Bin-hy/zhupi/internal/store"
)

type Server struct {
	cfg         *config.Config
	store       *store.Store
	sites       *service.SiteService
	auth        *service.AuthService
	annotations *service.AnnotationService
	moderation  *service.ModerationService
	admin       *service.AdminService

	readLimiter      *ratelimit.Limiter
	writeLimiter     *ratelimit.Limiter
	userWriteLimiter *ratelimit.Limiter
	loginLimiter     *ratelimit.Limiter
	logger           *slog.Logger
}

func New(cfg *config.Config, st *store.Store, sites *service.SiteService, authSvc *service.AuthService,
	ann *service.AnnotationService, mod *service.ModerationService, admin *service.AdminService,
	logger *slog.Logger) *Server {
	return &Server{
		cfg: cfg, store: st, sites: sites, auth: authSvc, annotations: ann, moderation: mod, admin: admin,
		readLimiter:      ratelimit.New(cfg.ReadRatePerMin),
		writeLimiter:     ratelimit.New(cfg.WriteRatePerMin),
		userWriteLimiter: ratelimit.New(cfg.WriteUserRatePerMin),
		loginLimiter:     ratelimit.New(cfg.LocalLoginRatePerMin),
		logger:           logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)

	// SDK 静态资源（含 ESM 拆分的 chunk）
	mux.HandleFunc("GET /sdk/v1/{name...}", s.handleSDKFile)

	// 管理后台静态资源（SPA fallback）
	mux.HandleFunc("GET /admin", s.handleAdminRoot)
	mux.HandleFunc("GET /admin/{path...}", s.handleAdminSPA)

	// 管理端登录（公开）
	mux.Handle("GET /api/v1/admin/auth/providers", chain(http.HandlerFunc(s.handleAdminProviders), s.readLimit))
	mux.Handle("POST /api/v1/admin/auth/local/login", chain(http.HandlerFunc(s.handleLocalLogin), s.loginLimit))
	mux.Handle("GET /api/v1/admin/auth/identities", chain(http.HandlerFunc(s.handleAdminIdentities), s.requireAuth, s.requireLocalSession, s.readLimit))
	mux.Handle("POST /api/v1/admin/auth/identities/github/start", chain(http.HandlerFunc(s.handleAdminBindGitHubStart), s.requireAuth, s.requireLocalSession, s.readLimit))
	mux.Handle("POST /api/v1/admin/auth/identities/oidc/{providerID}/start", chain(http.HandlerFunc(s.handleAdminBindOIDCStart), s.requireAuth, s.requireLocalSession, s.readLimit))

	// 站点公共接口
	mux.Handle("GET /api/v1/site-config", s.publicSite(s.handleSiteConfig))
	mux.Handle("GET /api/v1/annotations", s.publicSite(s.handleListAnnotations))
	mux.Handle("POST /api/v1/annotations", s.writeSite(s.handleCreateAnnotation))
	mux.Handle("PATCH /api/v1/annotations/{id}", s.writeSite(s.handlePatchAnnotation))
	mux.Handle("DELETE /api/v1/annotations/{id}", s.writeSite(s.handleDeleteAnnotation))
	mux.Handle("POST /api/v1/annotations/{id}/comments", s.writeSite(s.handleCreateComment))
	mux.Handle("DELETE /api/v1/comments/{id}", s.writeSite(s.handleDeleteComment))

	// 登录
	mux.Handle("GET /api/v1/auth/github/start", chain(http.HandlerFunc(s.handleGitHubStart), s.readLimit))
	mux.HandleFunc("GET /api/v1/auth/github/callback", s.handleGitHubCallback)
	mux.Handle("GET /api/v1/auth/oidc/{providerID}/start", chain(http.HandlerFunc(s.handleOIDCStart), s.readLimit))
	mux.HandleFunc("GET /api/v1/auth/oidc/{providerID}/callback", s.handleOIDCCallback)
	mux.Handle("POST /api/v1/auth/dev/login", chain(http.HandlerFunc(s.handleDevLogin), s.readLimit))
	mux.Handle("POST /api/v1/auth/logout", s.account(s.handleLogout))

	// 账号
	mux.Handle("GET /api/v1/me", s.account(s.handleMe))
	mux.Handle("POST /api/v1/me/password", s.account(s.handleChangePassword))
	mux.Handle("GET /api/v1/me/sites", s.account(s.handleMeSites))
	mux.Handle("GET /api/v1/me/identities", s.account(s.handleListIdentities))
	mux.Handle("DELETE /api/v1/me/identities/{id}", s.account(s.handleUnbindIdentity))
	mux.Handle("GET /api/v1/me/sessions", s.account(s.handleListSessions))
	mux.Handle("DELETE /api/v1/me/sessions", s.account(s.handleRevokeAllSessions))
	mux.Handle("DELETE /api/v1/me/sessions/{id}", s.account(s.handleRevokeSession))
	mux.Handle("POST /api/v1/me/identities/github/start",
		chain(http.HandlerFunc(s.handleBindGitHubStart), s.siteResolve, s.requireAuth, s.resolveSiteRole))
	mux.Handle("POST /api/v1/me/identities/oidc/{providerID}/start",
		chain(http.HandlerFunc(s.handleBindOIDCStart), s.siteResolve, s.requireAuth, s.resolveSiteRole))

	// 管理：站点与用户
	mux.Handle("GET /api/v1/admin/sites", chain(http.HandlerFunc(s.handleAdminListSites), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites", chain(http.HandlerFunc(s.handleAdminCreateSite), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("PATCH /api/v1/admin/sites/{id}", chain(http.HandlerFunc(s.handleAdminUpdateSite), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites/{id}/rotate-key", chain(http.HandlerFunc(s.handleAdminRotateKey), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("GET /api/v1/admin/sites/{id}/admins", chain(http.HandlerFunc(s.handleAdminListAdmins), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites/{id}/admins", chain(http.HandlerFunc(s.handleAdminAddAdmin), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("DELETE /api/v1/admin/sites/{id}/admins/{userID}", chain(http.HandlerFunc(s.handleAdminRemoveAdmin), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("GET /api/v1/admin/overview", chain(http.HandlerFunc(s.handleAdminOverview), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged))
	mux.Handle("GET /api/v1/admin/users", chain(http.HandlerFunc(s.handleAdminListUsers), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/users/{id}/ban", chain(http.HandlerFunc(s.handleAdminGlobalBan), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("DELETE /api/v1/admin/users/{id}/ban", chain(http.HandlerFunc(s.handleAdminGlobalUnban), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))

	// 管理：管理员账号
	mux.Handle("GET /api/v1/admin/local-admins", chain(http.HandlerFunc(s.handleAdminListLocalAdmins), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/local-admins", chain(http.HandlerFunc(s.handleAdminCreateLocalAdmin), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/local-admins/open", chain(http.HandlerFunc(s.handleAdminOpenLocalCredential), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/local-admins/{userID}/password/reset", chain(http.HandlerFunc(s.handleAdminResetLocalPassword), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))
	mux.Handle("PUT /api/v1/admin/users/{userID}/superadmin", chain(http.HandlerFunc(s.handleAdminSetSuperadmin), s.requireAuth, s.requireLocalSession, s.requirePasswordChanged, s.requireSuperadmin))

	// 管理：站点级审核与提供商
	mux.Handle("GET /api/v1/admin/sites/{siteID}/queue", s.adminSite(s.handleAdminQueue))
	mux.Handle("GET /api/v1/admin/sites/{siteID}/annotations/{id}", s.adminSite(s.handleAdminAnnotationDetail))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/annotations/{id}/moderate", s.adminSite(s.handleAdminModerateAnnotation))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/comments/{id}/moderate", s.adminSite(s.handleAdminModerateComment))
	mux.Handle("GET /api/v1/admin/sites/{siteID}/bans", s.adminSite(s.handleAdminListBans))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/bans/{userID}", s.adminSite(s.handleAdminBanUser))
	mux.Handle("DELETE /api/v1/admin/sites/{siteID}/bans/{userID}", s.adminSite(s.handleAdminUnbanUser))
	mux.Handle("GET /api/v1/admin/sites/{siteID}/providers", s.adminSite(s.handleAdminListProviders))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/providers", s.adminSite(s.handleAdminCreateProvider))
	mux.Handle("PATCH /api/v1/admin/sites/{siteID}/providers/{providerID}", s.adminSite(s.handleAdminUpdateProvider))
	mux.Handle("DELETE /api/v1/admin/sites/{siteID}/providers/{providerID}", s.adminSite(s.handleAdminDeleteProvider))

	return s.recoverer(s.requestID(s.logging(s.bodyLimit(s.cors(mux)))))
}

func (s *Server) adminSite(h http.HandlerFunc) http.Handler {
	return chain(h, s.requireAuth, s.requireLocalSession, s.requirePasswordChanged)
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Bin-hy/go-campus/annotation-service/internal/config"
	"github.com/Bin-hy/go-campus/annotation-service/internal/ratelimit"
	"github.com/Bin-hy/go-campus/annotation-service/internal/service"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
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
		logger:           logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)

	// SDK 静态资源（含 ESM 拆分的 chunk）
	mux.HandleFunc("GET /sdk/v1/{name...}", s.handleSDKFile)

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
	mux.Handle("GET /api/v1/admin/sites", chain(http.HandlerFunc(s.handleAdminListSites), s.requireAuth, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites", chain(http.HandlerFunc(s.handleAdminCreateSite), s.requireAuth, s.requireSuperadmin))
	mux.Handle("PATCH /api/v1/admin/sites/{id}", chain(http.HandlerFunc(s.handleAdminUpdateSite), s.requireAuth, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites/{id}/rotate-key", chain(http.HandlerFunc(s.handleAdminRotateKey), s.requireAuth, s.requireSuperadmin))
	mux.Handle("GET /api/v1/admin/sites/{id}/admins", chain(http.HandlerFunc(s.handleAdminListAdmins), s.requireAuth, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/sites/{id}/admins", chain(http.HandlerFunc(s.handleAdminAddAdmin), s.requireAuth, s.requireSuperadmin))
	mux.Handle("DELETE /api/v1/admin/sites/{id}/admins/{userID}", chain(http.HandlerFunc(s.handleAdminRemoveAdmin), s.requireAuth, s.requireSuperadmin))
	mux.Handle("GET /api/v1/admin/users", chain(http.HandlerFunc(s.handleAdminListUsers), s.requireAuth, s.requireSuperadmin))
	mux.Handle("POST /api/v1/admin/users/{id}/ban", chain(http.HandlerFunc(s.handleAdminGlobalBan), s.requireAuth, s.requireSuperadmin))
	mux.Handle("DELETE /api/v1/admin/users/{id}/ban", chain(http.HandlerFunc(s.handleAdminGlobalUnban), s.requireAuth, s.requireSuperadmin))

	// 管理：站点级审核与提供商
	mux.Handle("GET /api/v1/admin/sites/{siteID}/queue", s.adminSite(s.handleAdminQueue))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/annotations/{id}/moderate", s.adminSite(s.handleAdminModerateAnnotation))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/comments/{id}/moderate", s.adminSite(s.handleAdminModerateComment))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/bans/{userID}", s.adminSite(s.handleAdminBanUser))
	mux.Handle("DELETE /api/v1/admin/sites/{siteID}/bans/{userID}", s.adminSite(s.handleAdminUnbanUser))
	mux.Handle("GET /api/v1/admin/sites/{siteID}/providers", s.adminSite(s.handleAdminListProviders))
	mux.Handle("POST /api/v1/admin/sites/{siteID}/providers", s.adminSite(s.handleAdminCreateProvider))
	mux.Handle("PATCH /api/v1/admin/sites/{siteID}/providers/{providerID}", s.adminSite(s.handleAdminUpdateProvider))
	mux.Handle("DELETE /api/v1/admin/sites/{siteID}/providers/{providerID}", s.adminSite(s.handleAdminDeleteProvider))

	return s.recoverer(s.requestID(s.logging(s.bodyLimit(s.cors(mux)))))
}

func (s *Server) adminSite(h http.HandlerFunc) http.Handler {
	return chain(h, s.requireAuth)
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

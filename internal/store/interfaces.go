package store

import (
	"context"

	"github.com/Bin-hy/zhupi/internal/domain"
)

// 每个领域使用唯一的方法名，避免同一 *Store 上出现同名不同签名的冲突；
// 服务层只依赖所需的窄接口。

type SiteRepo interface {
	CreateSite(ctx context.Context, s domain.Site) (domain.Site, error)
	SiteByID(ctx context.Context, id int64) (domain.Site, error)
	SiteByKey(ctx context.Context, key string) (domain.Site, error)
	UpdateSite(ctx context.Context, s domain.Site) error
	RotateSiteKey(ctx context.Context, id int64, newKey string) error
	ReplaceSiteOrigins(ctx context.Context, siteID int64, origins []string) error
	ListSites(ctx context.Context, cursor int64, limit int) ([]domain.Site, int64, error)
}

type UserRepo interface {
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	UserByID(ctx context.Context, id int64) (domain.User, error)
	UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error
	SetUserSuperadmin(ctx context.Context, id int64, v bool) error
	ListUsers(ctx context.Context, cursor int64, limit int) ([]domain.User, int64, error)
}

type IdentityRepo interface {
	CreateIdentity(ctx context.Context, i domain.Identity) (domain.Identity, error)
	IdentityByID(ctx context.Context, id int64) (domain.Identity, error)
	IdentityByProviderSubject(ctx context.Context, provider domain.ProviderType, siteID int64, issuer, subject string) (domain.Identity, error)
	IdentityByHandle(ctx context.Context, provider domain.ProviderType, handle string) (domain.Identity, error)
	ListIdentitiesByUser(ctx context.Context, userID int64) ([]domain.Identity, error)
	DeleteIdentity(ctx context.Context, id, userID int64) error
	TouchIdentityLogin(ctx context.Context, id int64, at int64) error
}

type SessionRepo interface {
	CreateSession(ctx context.Context, s domain.Session) error
	SessionByTokenHash(ctx context.Context, hash string) (domain.Session, error)
	ListSessionsByUser(ctx context.Context, userID int64) ([]domain.Session, error)
	RevokeSession(ctx context.Context, id, userID int64) error
	RevokeAllSessionsForUser(ctx context.Context, userID int64) error
	RevokeOtherSessionsForUser(ctx context.Context, userID, keepSessionID int64) error
	TouchSession(ctx context.Context, id int64, at int64) error
	PurgeExpiredSessions(ctx context.Context) error
}

type LocalCredentialRepo interface {
	CreateLocalCredential(ctx context.Context, c domain.LocalCredential) error
	LocalCredentialByUsername(ctx context.Context, username string) (domain.LocalCredential, error)
	LocalCredentialByUserID(ctx context.Context, userID int64) (domain.LocalCredential, error)
	CountLocalCredentials(ctx context.Context) (int, error)
	UpdateLocalCredentialPassword(ctx context.Context, userID int64, hash string, mustChange bool) error
	ListLocalAdmins(ctx context.Context) ([]domain.LocalAdminAccount, error)
	CountActiveSuperadminCredentials(ctx context.Context) (int, error)
	CreateLocalAdmin(ctx context.Context, u domain.User, c domain.LocalCredential, siteIDs []int64, grantedBy int64) (domain.User, error)
}

type AuthFlowRepo interface {
	CreateAuthFlow(ctx context.Context, f domain.AuthFlow) error
	ConsumeAuthFlow(ctx context.Context, stateHash string) (domain.AuthFlow, error)
	PurgeExpiredAuthFlows(ctx context.Context) error
}

type AnnotationQuery struct {
	SiteID            int64
	PageKey           string
	ViewerID          int64
	Filter            domain.ListFilter
	Sort              domain.AnnotationSort
	Keyword           string
	IncludeUnapproved bool
	Cursor            int64
	Limit             int
}

type AnnotationRepo interface {
	CreateAnnotation(ctx context.Context, a domain.Annotation, firstComment string) (domain.Annotation, error)
	AnnotationByID(ctx context.Context, siteID, id int64) (domain.Annotation, error)
	AnnotationsByIDs(ctx context.Context, siteID int64, ids []int64) (map[int64]domain.Annotation, error)
	FirstCommentsByAnnotationIDs(ctx context.Context, siteID int64, ids []int64) (map[int64]string, error)
	ListAnnotations(ctx context.Context, q AnnotationQuery) ([]domain.Annotation, int64, error)
	CountPageAnnotations(ctx context.Context, siteID int64, pageKey string, viewerID int64, includeUnapproved bool) (int, error)
	CountPageAnnotationsByFilter(ctx context.Context, siteID int64, pageKey string, viewerID int64, includeUnapproved bool) (map[domain.ListFilter]int, error)
	ListAnnotationsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, keyword string, cursor int64, limit int) ([]domain.Annotation, int64, error)
	CountAnnotationsForModeration(ctx context.Context, siteID int64) (map[domain.ModerationStatus]int, error)
	UpdateAnnotationStatus(ctx context.Context, siteID, id int64, status domain.AnnotationStatus) error
	UpdateAnnotationAnchor(ctx context.Context, siteID, id int64, a domain.Anchor) error
	SetAnnotationVisibility(ctx context.Context, siteID, id int64, v domain.Visibility) error
	SetAnnotationModeration(ctx context.Context, siteID, id int64, m domain.ModerationStatus) error
	SoftDeleteAnnotation(ctx context.Context, siteID, id int64) error
}

type CommentRepo interface {
	CreateComment(ctx context.Context, c domain.Comment) (domain.Comment, error)
	CommentByID(ctx context.Context, siteID, id int64) (domain.Comment, error)
	ListCommentsByAnnotation(ctx context.Context, siteID, annotationID, viewerID int64, includeUnapproved bool) ([]domain.Comment, error)
	ListCommentsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, keyword string, cursor int64, limit int) ([]domain.Comment, int64, error)
	CountCommentsForModeration(ctx context.Context, siteID int64) (map[domain.ModerationStatus]int, error)
	SetCommentModeration(ctx context.Context, siteID, id int64, m domain.ModerationStatus) error
	SoftDeleteComment(ctx context.Context, siteID, id int64) error
}

type ProviderWithSite struct {
	Provider domain.OIDCProvider
	SiteID   int64
	SiteName string
}

type ProviderRepo interface {
	CreateProvider(ctx context.Context, p domain.OIDCProvider) (domain.OIDCProvider, error)
	ProviderByID(ctx context.Context, siteID, id int64) (domain.OIDCProvider, error)
	ProviderByIDAny(ctx context.Context, id int64) (domain.OIDCProvider, error)
	ListProvidersBySite(ctx context.Context, siteID int64, onlyEnabled bool) ([]domain.OIDCProvider, error)
	ListEnabledWithSite(ctx context.Context) ([]ProviderWithSite, error)
	UpdateProvider(ctx context.Context, p domain.OIDCProvider) error
	DeleteProvider(ctx context.Context, siteID, id int64) error
}

type AdminRepo interface {
	AddSiteAdmin(ctx context.Context, a domain.SiteAdmin) error
	RemoveSiteAdmin(ctx context.Context, siteID, userID int64) error
	ListSiteAdmins(ctx context.Context, siteID int64) ([]domain.SiteAdmin, error)
	IsSiteAdmin(ctx context.Context, siteID, userID int64) (bool, error)
	BanSite(ctx context.Context, b domain.SiteBan) error
	UnbanSite(ctx context.Context, siteID, userID int64) error
	IsBanned(ctx context.Context, siteID, userID int64) (bool, error)
	ListAdminSiteIDs(ctx context.Context, userID int64) ([]int64, error)
	ListSitesForAdmin(ctx context.Context, userID int64) ([]domain.Site, error)
	ListSiteBans(ctx context.Context, siteID int64) ([]domain.SiteBanView, error)
}

type StatsRepo interface {
	CountSites(ctx context.Context) (int, error)
	CountUsers(ctx context.Context) (int, error)
	CountAnnotations(ctx context.Context, siteIDs []int64, moderation domain.ModerationStatus) (int, error)
	CountComments(ctx context.Context, siteIDs []int64, moderation domain.ModerationStatus) (int, error)
	CountDistinctAuthors(ctx context.Context, siteIDs []int64) (int, error)
	RecentPendingAnnotations(ctx context.Context, siteIDs []int64, limit int) ([]domain.OverviewPending, error)
	RecentPendingComments(ctx context.Context, siteIDs []int64, limit int) ([]domain.OverviewPending, error)
}

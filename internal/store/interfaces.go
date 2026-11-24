package store

import (
	"context"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
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
	TouchSession(ctx context.Context, id int64, at int64) error
	PurgeExpiredSessions(ctx context.Context) error
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
	IncludeUnapproved bool
	Cursor            int64
	Limit             int
}

type AnnotationRepo interface {
	CreateAnnotation(ctx context.Context, a domain.Annotation, firstComment string) (domain.Annotation, error)
	AnnotationByID(ctx context.Context, siteID, id int64) (domain.Annotation, error)
	ListAnnotations(ctx context.Context, q AnnotationQuery) ([]domain.Annotation, int64, error)
	ListAnnotationsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, cursor int64, limit int) ([]domain.Annotation, int64, error)
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
	ListCommentsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, cursor int64, limit int) ([]domain.Comment, int64, error)
	SetCommentModeration(ctx context.Context, siteID, id int64, m domain.ModerationStatus) error
	SoftDeleteComment(ctx context.Context, siteID, id int64) error
}

type ProviderRepo interface {
	CreateProvider(ctx context.Context, p domain.OIDCProvider) (domain.OIDCProvider, error)
	ProviderByID(ctx context.Context, siteID, id int64) (domain.OIDCProvider, error)
	ListProvidersBySite(ctx context.Context, siteID int64, onlyEnabled bool) ([]domain.OIDCProvider, error)
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
}

package service

import (
	"context"
	"strings"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
)

type AdminService struct {
	sites      store.SiteRepo
	users      store.UserRepo
	identities store.IdentityRepo
	admins     store.AdminRepo
	sessions   store.SessionRepo
}

func NewAdminService(sites store.SiteRepo, users store.UserRepo, identities store.IdentityRepo,
	admins store.AdminRepo, sessions store.SessionRepo) *AdminService {
	return &AdminService{sites: sites, users: users, identities: identities, admins: admins, sessions: sessions}
}

func (a *AdminService) AddSiteAdmin(ctx context.Context, p domain.Principal, siteID, userID int64) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	if _, err := a.sites.SiteByID(ctx, siteID); err != nil {
		return domain.NotFound("站点不存在")
	}
	if _, err := a.users.UserByID(ctx, userID); err != nil {
		return domain.NotFound("用户不存在")
	}
	return a.admins.AddSiteAdmin(ctx, domain.SiteAdmin{SiteID: siteID, UserID: userID, GrantedBy: p.User.ID})
}

func (a *AdminService) RemoveSiteAdmin(ctx context.Context, p domain.Principal, siteID, userID int64) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	return a.admins.RemoveSiteAdmin(ctx, siteID, userID)
}

func (a *AdminService) ListSiteAdmins(ctx context.Context, p domain.Principal, siteID int64) ([]domain.SiteAdmin, error) {
	if !p.IsSuperadmin && !p.CanModerateSite(siteID) {
		return nil, domain.ErrForbidden
	}
	return a.admins.ListSiteAdmins(ctx, siteID)
}

func (a *AdminService) ListUsers(ctx context.Context, p domain.Principal, cursor int64, limit int) ([]domain.User, int64, error) {
	if !p.IsSuperadmin {
		return nil, 0, domain.ErrForbidden
	}
	return a.users.ListUsers(ctx, cursor, limit)
}

func (a *AdminService) GlobalBan(ctx context.Context, p domain.Principal, userID int64) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	if userID == p.User.ID {
		return domain.Invalid("validation_error", "不能封禁自己")
	}
	if err := a.users.UpdateUserStatus(ctx, userID, domain.UserBanned); err != nil {
		return err
	}
	return a.sessions.RevokeAllSessionsForUser(ctx, userID)
}

func (a *AdminService) GlobalUnban(ctx context.Context, p domain.Principal, userID int64) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	return a.users.UpdateUserStatus(ctx, userID, domain.UserActive)
}

func (a *AdminService) PromoteByGitHubHandle(ctx context.Context, handle string) (int64, error) {
	identity, err := a.identities.IdentityByHandle(ctx, domain.ProviderGitHub, strings.TrimSpace(handle))
	if err != nil {
		return 0, err
	}
	if err := a.users.SetUserSuperadmin(ctx, identity.UserID, true); err != nil {
		return 0, err
	}
	return identity.UserID, nil
}

func (a *AdminService) PromoteUser(ctx context.Context, userID int64) error {
	return a.users.SetUserSuperadmin(ctx, userID, true)
}

package service

import (
	"context"
	"strings"

	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

type AdminService struct {
	sites      store.SiteRepo
	users      store.UserRepo
	identities store.IdentityRepo
	admins     store.AdminRepo
	sessions   store.SessionRepo
	stats      store.StatsRepo
}

func NewAdminService(sites store.SiteRepo, users store.UserRepo, identities store.IdentityRepo,
	admins store.AdminRepo, sessions store.SessionRepo, stats store.StatsRepo) *AdminService {
	return &AdminService{sites: sites, users: users, identities: identities, admins: admins, sessions: sessions, stats: stats}
}

type OverviewCounts struct {
	Sites              int `json:"sites"`
	Users              int `json:"users"`
	Annotations        int `json:"annotations"`
	PendingAnnotations int `json:"pendingAnnotations"`
	Comments           int `json:"comments"`
	PendingComments    int `json:"pendingComments"`
}

type Overview struct {
	Counts            OverviewCounts           `json:"counts"`
	Sites             []domain.Site            `json:"sites"`
	RecentAnnotations []domain.OverviewPending `json:"recentAnnotations"`
	RecentComments    []domain.OverviewPending `json:"recentComments"`
}

func (a *AdminService) AccessibleSites(ctx context.Context, p domain.Principal) ([]domain.Site, error) {
	if p.IsSuperadmin {
		sites, _, err := a.sites.ListSites(ctx, 0, 200)
		return sites, err
	}
	return a.admins.ListSitesForAdmin(ctx, p.User.ID)
}

func (a *AdminService) Overview(ctx context.Context, p domain.Principal) (Overview, error) {
	sites, err := a.AccessibleSites(ctx, p)
	if err != nil {
		return Overview{}, err
	}
	var siteIDs []int64
	for _, site := range sites {
		siteIDs = append(siteIDs, site.ID)
	}

	var scope []int64
	if !p.IsSuperadmin {
		if len(siteIDs) == 0 {
			return Overview{Sites: sites, RecentAnnotations: []domain.OverviewPending{}, RecentComments: []domain.OverviewPending{}}, nil
		}
		scope = siteIDs
	}

	counts := OverviewCounts{}
	if p.IsSuperadmin {
		if counts.Sites, err = a.stats.CountSites(ctx); err != nil {
			return Overview{}, err
		}
		if counts.Users, err = a.stats.CountUsers(ctx); err != nil {
			return Overview{}, err
		}
	} else {
		counts.Sites = len(sites)
		if counts.Users, err = a.stats.CountDistinctAuthors(ctx, scope); err != nil {
			return Overview{}, err
		}
	}
	if counts.Annotations, err = a.stats.CountAnnotations(ctx, scope, ""); err != nil {
		return Overview{}, err
	}
	if counts.PendingAnnotations, err = a.stats.CountAnnotations(ctx, scope, domain.ModerationPending); err != nil {
		return Overview{}, err
	}
	if counts.Comments, err = a.stats.CountComments(ctx, scope, ""); err != nil {
		return Overview{}, err
	}
	if counts.PendingComments, err = a.stats.CountComments(ctx, scope, domain.ModerationPending); err != nil {
		return Overview{}, err
	}

	recentAnnotations, err := a.stats.RecentPendingAnnotations(ctx, scope, 10)
	if err != nil {
		return Overview{}, err
	}
	recentComments, err := a.stats.RecentPendingComments(ctx, scope, 10)
	if err != nil {
		return Overview{}, err
	}
	if sites == nil {
		sites = []domain.Site{}
	}
	if recentAnnotations == nil {
		recentAnnotations = []domain.OverviewPending{}
	}
	if recentComments == nil {
		recentComments = []domain.OverviewPending{}
	}
	return Overview{
		Counts:            counts,
		Sites:             sites,
		RecentAnnotations: recentAnnotations,
		RecentComments:    recentComments,
	}, nil
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

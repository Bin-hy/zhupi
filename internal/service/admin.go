package service

import (
	"context"
	"errors"
	"strings"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

type AdminService struct {
	sites      store.SiteRepo
	users      store.UserRepo
	admins     store.AdminRepo
	sessions   store.SessionRepo
	stats      store.StatsRepo
	localCreds store.LocalCredentialRepo
}

func NewAdminService(sites store.SiteRepo, users store.UserRepo,
	admins store.AdminRepo, sessions store.SessionRepo, stats store.StatsRepo,
	localCreds store.LocalCredentialRepo) *AdminService {
	return &AdminService{
		sites: sites, users: users, admins: admins, sessions: sessions,
		stats: stats, localCreds: localCreds,
	}
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

type SiteAdminView struct {
	domain.SiteAdmin
	DisplayName        string `json:"displayName"`
	LocalUsername      string `json:"localUsername"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func (a *AdminService) ListSiteAdmins(ctx context.Context, p domain.Principal, siteID int64) ([]domain.SiteAdmin, error) {
	if !p.IsSuperadmin && !p.CanModerateSite(siteID) {
		return nil, domain.ErrForbidden
	}
	return a.admins.ListSiteAdmins(ctx, siteID)
}

func (a *AdminService) ListSiteAdminsView(ctx context.Context, p domain.Principal, siteID int64) ([]SiteAdminView, error) {
	list, err := a.ListSiteAdmins(ctx, p, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]SiteAdminView, 0, len(list))
	for _, item := range list {
		view := SiteAdminView{SiteAdmin: item}
		if user, err := a.users.UserByID(ctx, item.UserID); err == nil {
			view.DisplayName = user.DisplayName
		}
		if cred, err := a.localCreds.LocalCredentialByUserID(ctx, item.UserID); err == nil {
			view.LocalUsername = cred.Username
			view.MustChangePassword = cred.MustChangePassword
		}
		out = append(out, view)
	}
	return out, nil
}

func (a *AdminService) ListUsers(ctx context.Context, p domain.Principal, cursor int64, limit int) ([]domain.User, int64, error) {
	if !p.IsSuperadmin {
		return nil, 0, domain.ErrForbidden
	}
	return a.users.ListUsers(ctx, cursor, limit)
}

type CreateLocalAdminInput struct {
	Username        string  `json:"username"`
	DisplayName     string  `json:"displayName"`
	InitialPassword string  `json:"initialPassword"`
	IsSuperadmin    bool    `json:"isSuperadmin"`
	SiteIDs         []int64 `json:"siteIds"`
}

type OpenLocalCredentialInput struct {
	UserID          int64   `json:"userId"`
	Username        string  `json:"username"`
	InitialPassword string  `json:"initialPassword"`
	IsSuperadmin    bool    `json:"isSuperadmin"`
	SiteIDs         []int64 `json:"siteIds"`
}

type LocalAdminResult struct {
	Account           domain.LocalAdminAccount `json:"account"`
	GeneratedPassword string                   `json:"generatedPassword,omitempty"`
}

func (a *AdminService) ListLocalAdmins(ctx context.Context, p domain.Principal) ([]domain.LocalAdminAccount, error) {
	if !p.IsSuperadmin {
		return nil, domain.ErrForbidden
	}
	return a.localCreds.ListLocalAdmins(ctx)
}

func (a *AdminService) CreateLocalAdmin(ctx context.Context, p domain.Principal, in CreateLocalAdminInput) (LocalAdminResult, error) {
	if !p.IsSuperadmin {
		return LocalAdminResult{}, domain.ErrForbidden
	}
	username := strings.TrimSpace(in.Username)
	if err := authpkg.ValidateUsername(username); err != nil {
		return LocalAdminResult{}, domain.Invalid("invalid_username", err.Error())
	}
	if err := a.ensureUsernameAvailable(ctx, username); err != nil {
		return LocalAdminResult{}, err
	}
	password, generated, err := resolveAdminPassword(in.InitialPassword)
	if err != nil {
		return LocalAdminResult{}, err
	}
	siteIDs := dedupeSiteIDs(in.SiteIDs)
	if err := a.validateSiteIDs(ctx, siteIDs); err != nil {
		return LocalAdminResult{}, err
	}
	hash, err := authpkg.HashPassword(password)
	if err != nil {
		return LocalAdminResult{}, err
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = username
	}
	user, err := a.localCreds.CreateLocalAdmin(ctx, domain.User{
		DisplayName: displayName, Status: domain.UserActive, IsSuperadmin: in.IsSuperadmin,
	}, domain.LocalCredential{
		Username: username, PasswordHash: hash, MustChangePassword: true,
	}, siteIDs, p.User.ID)
	if err != nil {
		return LocalAdminResult{}, err
	}
	result := LocalAdminResult{Account: domain.LocalAdminAccount{
		UserID: user.ID, Username: username, DisplayName: user.DisplayName,
		Status: user.Status, IsSuperadmin: user.IsSuperadmin,
		MustChangePassword: true, SiteIDs: siteIDs, CreatedAt: user.CreatedAt,
	}}
	if generated {
		result.GeneratedPassword = password
	}
	return result, nil
}

func (a *AdminService) OpenLocalCredential(ctx context.Context, p domain.Principal, in OpenLocalCredentialInput) (LocalAdminResult, error) {
	if !p.IsSuperadmin {
		return LocalAdminResult{}, domain.ErrForbidden
	}
	if in.UserID <= 0 {
		return LocalAdminResult{}, domain.Invalid("validation_error", "缺少 userId")
	}
	user, err := a.users.UserByID(ctx, in.UserID)
	if err != nil {
		return LocalAdminResult{}, domain.NotFound("用户不存在")
	}
	if user.Status == domain.UserBanned {
		return LocalAdminResult{}, domain.Conflict("user_banned", "用户已被封禁")
	}
	if _, err := a.localCreds.LocalCredentialByUserID(ctx, in.UserID); err == nil {
		return LocalAdminResult{}, domain.Conflict("local_credential_exists", "该用户已开通本地密码")
	} else if !errors.Is(err, domain.ErrNotFound) {
		return LocalAdminResult{}, err
	}
	username := strings.TrimSpace(in.Username)
	if err := authpkg.ValidateUsername(username); err != nil {
		return LocalAdminResult{}, domain.Invalid("invalid_username", err.Error())
	}
	if err := a.ensureUsernameAvailable(ctx, username); err != nil {
		return LocalAdminResult{}, err
	}
	password, generated, err := resolveAdminPassword(in.InitialPassword)
	if err != nil {
		return LocalAdminResult{}, err
	}
	siteIDs := dedupeSiteIDs(in.SiteIDs)
	if err := a.validateSiteIDs(ctx, siteIDs); err != nil {
		return LocalAdminResult{}, err
	}
	hash, err := authpkg.HashPassword(password)
	if err != nil {
		return LocalAdminResult{}, err
	}
	if err := a.localCreds.CreateLocalCredential(ctx, domain.LocalCredential{
		UserID: in.UserID, Username: username, PasswordHash: hash, MustChangePassword: true,
	}); err != nil {
		return LocalAdminResult{}, err
	}
	if in.IsSuperadmin && !user.IsSuperadmin {
		if err := a.users.SetUserSuperadmin(ctx, in.UserID, true); err != nil {
			return LocalAdminResult{}, err
		}
		user.IsSuperadmin = true
	}
	for _, siteID := range siteIDs {
		if err := a.admins.AddSiteAdmin(ctx, domain.SiteAdmin{
			SiteID: siteID, UserID: in.UserID, GrantedBy: p.User.ID,
		}); err != nil {
			return LocalAdminResult{}, err
		}
	}
	result := LocalAdminResult{Account: domain.LocalAdminAccount{
		UserID: user.ID, Username: username, DisplayName: user.DisplayName,
		Status: user.Status, IsSuperadmin: user.IsSuperadmin,
		MustChangePassword: true, SiteIDs: siteIDs, CreatedAt: user.CreatedAt,
	}}
	if generated {
		result.GeneratedPassword = password
	}
	return result, nil
}

func (a *AdminService) ResetLocalAdminPassword(ctx context.Context, p domain.Principal, userID int64, password string) (string, error) {
	if !p.IsSuperadmin {
		return "", domain.ErrForbidden
	}
	if userID == p.User.ID {
		return "", domain.Invalid("self_password_reset", "请在账号设置页修改自己的密码")
	}
	if _, err := a.localCreds.LocalCredentialByUserID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.NotFound("本地账号不存在")
		}
		return "", err
	}
	next, generated, err := resolveAdminPassword(password)
	if err != nil {
		return "", err
	}
	hash, err := authpkg.HashPassword(next)
	if err != nil {
		return "", err
	}
	if err := a.localCreds.UpdateLocalCredentialPassword(ctx, userID, hash, true); err != nil {
		return "", err
	}
	if err := a.sessions.RevokeAllSessionsForUser(ctx, userID); err != nil {
		return "", err
	}
	if generated {
		return next, nil
	}
	return "", nil
}

func (a *AdminService) SetSuperadmin(ctx context.Context, p domain.Principal, userID int64, v bool) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	if _, err := a.users.UserByID(ctx, userID); err != nil {
		return domain.NotFound("用户不存在")
	}
	if v {
		if _, err := a.localCreds.LocalCredentialByUserID(ctx, userID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.Conflict("local_credential_required", "请先为该用户开通本地密码")
			}
			return err
		}
	} else {
		last, err := a.isLastActiveSuperadmin(ctx, userID)
		if err != nil {
			return err
		}
		if last {
			return domain.Conflict("last_superadmin", "不能撤销最后一个可登录的超级管理员")
		}
	}
	return a.users.SetUserSuperadmin(ctx, userID, v)
}

func (a *AdminService) GlobalBan(ctx context.Context, p domain.Principal, userID int64) error {
	if !p.IsSuperadmin {
		return domain.ErrForbidden
	}
	if userID == p.User.ID {
		return domain.Invalid("validation_error", "不能封禁自己")
	}
	last, err := a.isLastActiveSuperadmin(ctx, userID)
	if err != nil {
		return err
	}
	if last {
		return domain.Conflict("last_superadmin", "不能封禁最后一个可登录的超级管理员")
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

func (a *AdminService) PromoteUser(ctx context.Context, userID int64) error {
	if _, err := a.localCreds.LocalCredentialByUserID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.Conflict("local_credential_required", "请先为该用户开通本地密码")
		}
		return err
	}
	return a.users.SetUserSuperadmin(ctx, userID, true)
}

func (a *AdminService) ensureUsernameAvailable(ctx context.Context, username string) error {
	_, err := a.localCreds.LocalCredentialByUsername(ctx, username)
	if err == nil {
		return domain.Conflict("username_taken", "用户名已被占用")
	}
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	return err
}

func resolveAdminPassword(raw string) (string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		password, err := authpkg.GeneratePassword()
		if err != nil {
			return "", false, err
		}
		return password, true, nil
	}
	if err := authpkg.ValidatePassword(raw); err != nil {
		return "", false, domain.Invalid("weak_password", err.Error())
	}
	return raw, false, nil
}

func dedupeSiteIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (a *AdminService) validateSiteIDs(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if _, err := a.sites.SiteByID(ctx, id); err != nil {
			return domain.NotFound("站点不存在")
		}
	}
	return nil
}

func (a *AdminService) isLastActiveSuperadmin(ctx context.Context, userID int64) (bool, error) {
	user, err := a.users.UserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if !user.IsSuperadmin || user.Status != domain.UserActive {
		return false, nil
	}
	if _, err := a.localCreds.LocalCredentialByUserID(ctx, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	n, err := a.localCreds.CountActiveSuperadminCredentials(ctx)
	if err != nil {
		return false, err
	}
	return n <= 1, nil
}

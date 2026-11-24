package store

import (
	"context"
	"time"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

func (s *Store) AddSiteAdmin(ctx context.Context, a domain.SiteAdmin) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO site_admins (site_id, user_id, granted_by, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(site_id, user_id) DO NOTHING`,
		a.SiteID, a.UserID, a.GrantedBy, toUnix(a.CreatedAt))
	return err
}

func (s *Store) RemoveSiteAdmin(ctx context.Context, siteID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM site_admins WHERE site_id = ? AND user_id = ?`, siteID, userID)
	return err
}

func (s *Store) ListSiteAdmins(ctx context.Context, siteID int64) ([]domain.SiteAdmin, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT site_id, user_id, granted_by, created_at FROM site_admins WHERE site_id = ? ORDER BY created_at ASC`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.SiteAdmin, 0)
	for rows.Next() {
		var (
			a       domain.SiteAdmin
			created int64
		)
		if err := rows.Scan(&a.SiteID, &a.UserID, &a.GrantedBy, &created); err != nil {
			return nil, err
		}
		a.CreatedAt = fromUnix(created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) IsSiteAdmin(ctx context.Context, siteID, userID int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM site_admins WHERE site_id = ? AND user_id = ?`, siteID, userID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) BanSite(ctx context.Context, b domain.SiteBan) error {
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO site_bans (site_id, user_id, banned_by, reason, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(site_id, user_id)
		DO UPDATE SET banned_by = excluded.banned_by, reason = excluded.reason, created_at = excluded.created_at`,
		b.SiteID, b.UserID, b.BannedBy, b.Reason, toUnix(b.CreatedAt))
	return err
}

func (s *Store) UnbanSite(ctx context.Context, siteID, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM site_bans WHERE site_id = ? AND user_id = ?`, siteID, userID)
	return err
}

func (s *Store) IsBanned(ctx context.Context, siteID, userID int64) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM site_bans WHERE site_id = ? AND user_id = ?`, siteID, userID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) ListAdminSiteIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT site_id FROM site_admins WHERE user_id = ? ORDER BY site_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) ListSitesForAdmin(ctx context.Context, userID int64) ([]domain.Site, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.key, s.name, s.display_name, s.status, s.moderation_mode,
		       s.theme_color, s.default_locale, s.created_at, s.updated_at
		FROM sites s JOIN site_admins sa ON sa.site_id = s.id
		WHERE sa.user_id = ?
		ORDER BY s.id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Site, 0)
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		origins, err := s.loadOrigins(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Origins = origins
	}
	return out, nil
}

func (s *Store) ListSiteBans(ctx context.Context, siteID int64) ([]domain.SiteBanView, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.user_id, u.display_name, u.avatar_url, b.reason, b.banned_by, b.created_at
		FROM site_bans b JOIN users u ON u.id = b.user_id
		WHERE b.site_id = ?
		ORDER BY b.created_at DESC
		LIMIT 200`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.SiteBanView, 0)
	for rows.Next() {
		var (
			v       domain.SiteBanView
			created int64
		)
		if err := rows.Scan(&v.UserID, &v.Name, &v.AvatarURL, &v.Reason, &v.BannedBy, &created); err != nil {
			return nil, err
		}
		v.CreatedAt = fromUnix(created)
		out = append(out, v)
	}
	return out, rows.Err()
}

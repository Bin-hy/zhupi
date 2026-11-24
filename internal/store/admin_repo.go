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

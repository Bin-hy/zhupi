package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const siteColumns = `id, key, name, display_name, status, moderation_mode, theme_color, default_locale, created_at, updated_at`

func scanSite(row interface{ Scan(...any) error }) (domain.Site, error) {
	var (
		s       domain.Site
		created int64
		updated int64
	)
	err := row.Scan(&s.ID, &s.Key, &s.Name, &s.DisplayName, &s.Status, &s.ModerationMode,
		&s.ThemeColor, &s.DefaultLocale, &created, &updated)
	if err != nil {
		return domain.Site{}, err
	}
	s.CreatedAt = fromUnix(created)
	s.UpdatedAt = fromUnix(updated)
	return s, nil
}

func (s *Store) loadOrigins(ctx context.Context, siteID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT origin FROM site_origins WHERE site_id = ? ORDER BY origin`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var origin string
		if err := rows.Scan(&origin); err != nil {
			return nil, err
		}
		out = append(out, origin)
	}
	return out, rows.Err()
}

func (s *Store) CreateSite(ctx context.Context, site domain.Site) (domain.Site, error) {
	now := time.Now()
	if site.CreatedAt.IsZero() {
		site.CreatedAt = now
	}
	site.UpdatedAt = now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Site{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO sites (key, name, display_name, status, moderation_mode, theme_color, default_locale, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		site.Key, site.Name, site.DisplayName, site.Status, site.ModerationMode,
		site.ThemeColor, site.DefaultLocale, toUnix(site.CreatedAt), toUnix(site.UpdatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Site{}, domain.Conflict("site_conflict", "站点名称或 siteKey 已存在")
		}
		return domain.Site{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Site{}, err
	}
	for _, origin := range site.Origins {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO site_origins (site_id, origin) VALUES (?, ?)`, id, origin); err != nil {
			return domain.Site{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Site{}, err
	}
	return s.SiteByID(ctx, id)
}

func (s *Store) SiteByID(ctx context.Context, id int64) (domain.Site, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+siteColumns+` FROM sites WHERE id = ?`, id)
	site, err := scanSite(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Site{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	origins, err := s.loadOrigins(ctx, id)
	if err != nil {
		return domain.Site{}, err
	}
	site.Origins = origins
	return site, nil
}

func (s *Store) SiteByKey(ctx context.Context, key string) (domain.Site, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+siteColumns+` FROM sites WHERE key = ?`, key)
	site, err := scanSite(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Site{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Site{}, err
	}
	origins, err := s.loadOrigins(ctx, site.ID)
	if err != nil {
		return domain.Site{}, err
	}
	site.Origins = origins
	return site, nil
}

func (s *Store) UpdateSite(ctx context.Context, site domain.Site) error {
	site.UpdatedAt = time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE sites SET name = ?, display_name = ?, status = ?, moderation_mode = ?,
			theme_color = ?, default_locale = ?, updated_at = ?
		WHERE id = ?`,
		site.Name, site.DisplayName, site.Status, site.ModerationMode,
		site.ThemeColor, site.DefaultLocale, toUnix(site.UpdatedAt), site.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Conflict("site_conflict", "站点名称已存在")
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	if site.Origins != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM site_origins WHERE site_id = ?`, site.ID); err != nil {
			return err
		}
		for _, origin := range site.Origins {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO site_origins (site_id, origin) VALUES (?, ?)`, site.ID, origin); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) RotateSiteKey(ctx context.Context, id int64, newKey string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sites SET key = ?, updated_at = ? WHERE id = ?`, newKey, time.Now().Unix(), id)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Conflict("site_key_conflict", "siteKey 冲突，请重试")
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ReplaceSiteOrigins(ctx context.Context, siteID int64, origins []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM site_origins WHERE site_id = ?`, siteID); err != nil {
		return err
	}
	for _, origin := range origins {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO site_origins (site_id, origin) VALUES (?, ?)`, siteID, origin); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// OriginAllowedAnywhere 用于 CORS 预检：浏览器预检不会携带 X-Site-Key。
// 只判断该来源是否属于任一启用站点；实际请求仍会严格校验 siteKey 与来源的对应关系。
func (s *Store) OriginAllowedAnywhere(ctx context.Context, origin string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM site_origins o JOIN sites s ON s.id = o.site_id
		WHERE o.origin = ? AND s.status = 'active'`, origin).Scan(&n)
	return n > 0, err
}

func (s *Store) ListSites(ctx context.Context, cursor int64, limit int) ([]domain.Site, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+siteColumns+` FROM sites WHERE id > ? ORDER BY id ASC LIMIT ?`, cursor, limit+1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, site)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var next int64
	if len(out) > limit {
		next = out[limit-1].ID
		out = out[:limit]
	}
	for i := range out {
		origins, err := s.loadOrigins(ctx, out[i].ID)
		if err != nil {
			return nil, 0, err
		}
		out[i].Origins = origins
	}
	return out, next, nil
}

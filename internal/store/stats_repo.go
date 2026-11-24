package store

import (
	"context"
	"strings"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func inClause(column string, n int) string {
	return column + " IN (" + strings.TrimSuffix(strings.Repeat("?,", n), ",") + ")"
}

func int64Args(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// siteCondition 返回站点过滤 SQL 与参数。nil 表示不限站点；空切片表示无结果。
func siteCondition(column string, siteIDs []int64) (string, []any) {
	if siteIDs == nil {
		return "", nil
	}
	if len(siteIDs) == 0 {
		return " AND 1 = 0", nil
	}
	return " AND " + inClause(column, len(siteIDs)), int64Args(siteIDs)
}

func (s *Store) CountSites(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sites`).Scan(&n)
	return n, err
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CountAnnotations(ctx context.Context, siteIDs []int64, moderation domain.ModerationStatus) (int, error) {
	cond, args := siteCondition("site_id", siteIDs)
	query := `SELECT COUNT(*) FROM annotations WHERE visibility <> 'deleted'` + cond
	if moderation != "" {
		query += ` AND moderation = ?`
		args = append(args, moderation)
	}
	var n int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func (s *Store) CountComments(ctx context.Context, siteIDs []int64, moderation domain.ModerationStatus) (int, error) {
	cond, args := siteCondition("site_id", siteIDs)
	query := `SELECT COUNT(*) FROM comments WHERE visibility <> 'deleted'` + cond
	if moderation != "" {
		query += ` AND moderation = ?`
		args = append(args, moderation)
	}
	var n int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func (s *Store) CountDistinctAuthors(ctx context.Context, siteIDs []int64) (int, error) {
	annCond, annArgs := siteCondition("site_id", siteIDs)
	comCond, comArgs := siteCondition("site_id", siteIDs)
	args := append(annArgs, comArgs...)
	query := `
		SELECT COUNT(*) FROM (
			SELECT created_by AS uid FROM annotations
			WHERE visibility <> 'deleted'` + annCond + `
			UNION
			SELECT user_id AS uid FROM comments
			WHERE visibility <> 'deleted'` + comCond + `
		)`
	var n int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

func (s *Store) RecentPendingAnnotations(ctx context.Context, siteIDs []int64, limit int) ([]domain.OverviewPending, error) {
	if siteIDs != nil && len(siteIDs) == 0 {
		return []domain.OverviewPending{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	cond, args := siteCondition("a.site_id", siteIDs)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.site_id, s.name, u.id, u.display_name, u.avatar_url, a.quote_exact, a.created_at
		FROM annotations a
		JOIN users u ON u.id = a.created_by
		JOIN sites s ON s.id = a.site_id
		WHERE a.moderation = 'pending' AND a.visibility <> 'deleted'`+cond+`
		ORDER BY a.created_at DESC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OverviewPending, 0)
	for rows.Next() {
		var (
			item    domain.OverviewPending
			created int64
		)
		if err := rows.Scan(&item.ID, &item.SiteID, &item.SiteName, &item.Author.ID, &item.Author.Name,
			&item.Author.AvatarURL, &item.QuoteExact, &created); err != nil {
			return nil, err
		}
		item.Kind = "annotation"
		item.CreatedAt = fromUnix(created)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecentPendingComments(ctx context.Context, siteIDs []int64, limit int) ([]domain.OverviewPending, error) {
	if siteIDs != nil && len(siteIDs) == 0 {
		return []domain.OverviewPending{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	cond, args := siteCondition("c.site_id", siteIDs)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.site_id, s.name, u.id, u.display_name, u.avatar_url, c.content, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		JOIN sites s ON s.id = c.site_id
		WHERE c.moderation = 'pending' AND c.visibility <> 'deleted'`+cond+`
		ORDER BY c.created_at DESC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OverviewPending, 0)
	for rows.Next() {
		var (
			item    domain.OverviewPending
			created int64
		)
		if err := rows.Scan(&item.ID, &item.SiteID, &item.SiteName, &item.Author.ID, &item.Author.Name,
			&item.Author.AvatarURL, &item.Content, &created); err != nil {
			return nil, err
		}
		item.Kind = "comment"
		item.CreatedAt = fromUnix(created)
		out = append(out, item)
	}
	return out, rows.Err()
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const commentSelect = `
	c.id, c.site_id, c.annotation_id, c.user_id, c.content, c.moderation, c.visibility, c.created_at,
	u.id, u.display_name, u.avatar_url`

func scanCommentRow(row interface{ Scan(...any) error }) (domain.Comment, error) {
	var (
		c       domain.Comment
		created int64
	)
	if err := row.Scan(&c.ID, &c.SiteID, &c.AnnotationID, &c.UserID, &c.Content,
		&c.Moderation, &c.Visibility, &created, &c.Author.ID, &c.Author.Name, &c.Author.AvatarURL); err != nil {
		return domain.Comment{}, err
	}
	c.CreatedAt = fromUnix(created)
	return c, nil
}

func (s *Store) CreateComment(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	if c.Moderation == "" {
		c.Moderation = domain.ModerationApproved
	}
	if c.Visibility == "" {
		c.Visibility = domain.VisibilityVisible
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO comments (site_id, annotation_id, user_id, content, moderation, visibility, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.SiteID, c.AnnotationID, c.UserID, c.Content, c.Moderation, c.Visibility, time.Now().Unix())
	if err != nil {
		return domain.Comment{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Comment{}, err
	}
	return s.CommentByID(ctx, c.SiteID, id)
}

func (s *Store) CommentByID(ctx context.Context, siteID, id int64) (domain.Comment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+commentSelect+`
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.site_id = ? AND c.id = ?`, siteID, id)
	c, err := scanCommentRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comment{}, domain.ErrNotFound
	}
	return c, err
}

func (s *Store) ListCommentsByAnnotation(ctx context.Context, siteID, annotationID, viewerID int64, includeUnapproved bool) ([]domain.Comment, error) {
	include := 0
	if includeUnapproved {
		include = 1
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+commentSelect+`
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.site_id = ? AND c.annotation_id = ? AND c.visibility <> 'deleted'
			AND ((c.moderation = 'approved' AND c.visibility = 'visible') OR c.user_id = ? OR ? = 1)
		ORDER BY c.id ASC`, siteID, annotationID, viewerID, include)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Comment, 0)
	for rows.Next() {
		c, err := scanCommentRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ListCommentsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, keyword string, cursor int64, limit int) ([]domain.Comment, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	conds := []string{"c.site_id = ?", "c.moderation = ?", "c.visibility <> 'deleted'", "c.is_initial = 0", "c.id > ?"}
	args := []any{siteID, status, cursor}
	if keyword != "" {
		like := likePattern(keyword)
		conds = append(conds, `(c.content LIKE ? ESCAPE '\' OR u.display_name LIKE ? ESCAPE '\' OR EXISTS (SELECT 1 FROM annotations a2 WHERE a2.id = c.annotation_id AND (a2.quote_exact LIKE ? ESCAPE '\' OR a2.page_title LIKE ? ESCAPE '\')))`)
		args = append(args, like, like, like, like)
	}
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+commentSelect+`
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY c.id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]domain.Comment, 0)
	for rows.Next() {
		c, err := scanCommentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var next int64
	if len(out) > limit {
		next = out[limit-1].ID
		out = out[:limit]
	}
	return out, next, nil
}

// CountCommentsForModeration 返回各审核状态下的评论数量。
func (s *Store) CountCommentsForModeration(ctx context.Context, siteID int64) (map[domain.ModerationStatus]int, error) {
	out := map[domain.ModerationStatus]int{
		domain.ModerationPending: 0, domain.ModerationApproved: 0, domain.ModerationRejected: 0,
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT moderation, COUNT(*)
		FROM comments
		WHERE site_id = ? AND visibility <> 'deleted' AND is_initial = 0
		GROUP BY moderation`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status domain.ModerationStatus
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

func (s *Store) SetCommentModeration(ctx context.Context, siteID, id int64, m domain.ModerationStatus) error {
	res, err := s.db.ExecContext(ctx, `UPDATE comments SET moderation = ? WHERE site_id = ? AND id = ?`, m, siteID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) SoftDeleteComment(ctx context.Context, siteID, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE comments SET visibility = 'deleted' WHERE site_id = ? AND id = ?`, siteID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

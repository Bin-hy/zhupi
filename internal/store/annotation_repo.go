package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

const annotationSelect = `
	a.id, a.site_id, a.page_key, a.page_title, a.anchor_json, a.quote_exact, a.section,
	a.status, a.moderation, a.visibility, a.created_by, a.created_at, a.updated_at,
	u.id, u.display_name, u.avatar_url`

func scanAnnotationRow(row interface{ Scan(...any) error }) (domain.Annotation, error) {
	var (
		a         domain.Annotation
		anchorRaw string
		created   int64
		updated   int64
	)
	if err := row.Scan(&a.ID, &a.SiteID, &a.PageKey, &a.PageTitle, &anchorRaw, &a.QuoteExact, &a.Section,
		&a.Status, &a.Moderation, &a.Visibility, &a.CreatedBy, &created, &updated,
		&a.Author.ID, &a.Author.Name, &a.Author.AvatarURL); err != nil {
		return domain.Annotation{}, err
	}
	if err := json.Unmarshal([]byte(anchorRaw), &a.Anchor); err != nil {
		return domain.Annotation{}, fmt.Errorf("解析锚点失败 (annotation %d): %w", a.ID, err)
	}
	a.CreatedAt = fromUnix(created)
	a.UpdatedAt = fromUnix(updated)
	a.Comments = []domain.Comment{}
	return a, nil
}

func (s *Store) CreateAnnotation(ctx context.Context, a domain.Annotation, firstComment string) (domain.Annotation, error) {
	anchorJSON, err := json.Marshal(a.Anchor)
	if err != nil {
		return domain.Annotation{}, err
	}
	now := time.Now()
	if a.Status == "" {
		a.Status = domain.AnnotationOpen
	}
	if a.Moderation == "" {
		a.Moderation = domain.ModerationApproved
	}
	if a.Visibility == "" {
		a.Visibility = domain.VisibilityVisible
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Annotation{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO annotations
			(site_id, page_key, page_title, anchor_json, quote_exact, section, status, moderation, visibility, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.SiteID, a.PageKey, a.PageTitle, string(anchorJSON), a.QuoteExact, a.Section,
		a.Status, a.Moderation, a.Visibility, a.CreatedBy, toUnix(now), toUnix(now))
	if err != nil {
		return domain.Annotation{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Annotation{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO comments (site_id, annotation_id, user_id, content, moderation, visibility, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.SiteID, id, a.CreatedBy, firstComment, domain.ModerationApproved, domain.VisibilityVisible, toUnix(now)); err != nil {
		return domain.Annotation{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Annotation{}, err
	}
	return s.AnnotationByID(ctx, a.SiteID, id)
}

func (s *Store) AnnotationByID(ctx context.Context, siteID, id int64) (domain.Annotation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+annotationSelect+`
		FROM annotations a JOIN users u ON u.id = a.created_by
		WHERE a.site_id = ? AND a.id = ?`, siteID, id)
	a, err := scanAnnotationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Annotation{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Annotation{}, err
	}
	comments, err := s.loadComments(ctx, siteID, []int64{a.ID}, 0, true)
	if err != nil {
		return domain.Annotation{}, err
	}
	if cs := comments[a.ID]; cs != nil {
		a.Comments = cs
	}
	return a, nil
}

func (s *Store) ListAnnotations(ctx context.Context, q AnnotationQuery) ([]domain.Annotation, int64, error) {
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	filter := q.Filter
	if filter == "" {
		filter = domain.ListAll
	}

	conds := []string{"a.site_id = ?", "a.page_key = ?", "a.visibility <> 'deleted'"}
	args := []any{q.SiteID, q.PageKey}
	visibleCond := "((a.moderation = 'approved' AND a.visibility = 'visible') OR a.created_by = ?)"

	switch filter {
	case domain.ListMine:
		if q.ViewerID == 0 {
			return []domain.Annotation{}, 0, nil
		}
		conds = append(conds, "a.created_by = ?")
		args = append(args, q.ViewerID)
	case domain.ListPending:
		if q.ViewerID == 0 {
			return []domain.Annotation{}, 0, nil
		}
		conds = append(conds, "a.created_by = ?", "a.moderation = 'pending'")
		args = append(args, q.ViewerID)
	case domain.ListResolved:
		if !q.IncludeUnapproved {
			conds = append(conds, visibleCond)
			args = append(args, q.ViewerID)
		}
		conds = append(conds, "a.status = 'resolved'")
	default: // all
		if !q.IncludeUnapproved {
			conds = append(conds, visibleCond)
			args = append(args, q.ViewerID)
		}
	}

	conds = append(conds, "a.id > ?")
	args = append(args, q.Cursor)
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, `
		SELECT `+annotationSelect+`
		FROM annotations a JOIN users u ON u.id = a.created_by
		WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY a.id ASC LIMIT ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Annotation, 0)
	for rows.Next() {
		a, err := scanAnnotationRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var next int64
	if len(out) > limit {
		next = out[limit-1].ID
		out = out[:limit]
	}
	if len(out) > 0 {
		ids := make([]int64, len(out))
		for i := range out {
			ids[i] = out[i].ID
		}
		comments, err := s.loadComments(ctx, q.SiteID, ids, q.ViewerID, q.IncludeUnapproved)
		if err != nil {
			return nil, 0, err
		}
		for i := range out {
			if cs := comments[out[i].ID]; cs != nil {
				out[i].Comments = cs
			}
		}
	}
	return out, next, nil
}

func (s *Store) UpdateAnnotationAnchor(ctx context.Context, siteID, id int64, anchor domain.Anchor) error {
	raw, err := json.Marshal(anchor)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE annotations
		SET anchor_json = ?, quote_exact = ?, section = ?, updated_at = ?
		WHERE site_id = ? AND id = ?`,
		string(raw), anchor.Quote.Exact, anchor.Section, time.Now().Unix(), siteID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAnnotationsForModeration(ctx context.Context, siteID int64, status domain.ModerationStatus, cursor int64, limit int) ([]domain.Annotation, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+annotationSelect+`
		FROM annotations a JOIN users u ON u.id = a.created_by
		WHERE a.site_id = ? AND a.moderation = ? AND a.visibility <> 'deleted' AND a.id > ?
		ORDER BY a.id ASC LIMIT ?`, siteID, status, cursor, limit+1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Annotation, 0)
	for rows.Next() {
		a, err := scanAnnotationRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var next int64
	if len(out) > limit {
		next = out[limit-1].ID
		out = out[:limit]
	}
	if len(out) > 0 {
		ids := make([]int64, len(out))
		for i := range out {
			ids[i] = out[i].ID
		}
		comments, err := s.loadComments(ctx, siteID, ids, 0, true)
		if err != nil {
			return nil, 0, err
		}
		for i := range out {
			if cs := comments[out[i].ID]; cs != nil {
				out[i].Comments = cs
			}
		}
	}
	return out, next, nil
}

func (s *Store) UpdateAnnotationStatus(ctx context.Context, siteID, id int64, status domain.AnnotationStatus) error {
	return s.updateAnnotation(ctx, siteID, id, `status = ?`, status)
}

func (s *Store) SetAnnotationVisibility(ctx context.Context, siteID, id int64, v domain.Visibility) error {
	return s.updateAnnotation(ctx, siteID, id, `visibility = ?`, v)
}

func (s *Store) SetAnnotationModeration(ctx context.Context, siteID, id int64, m domain.ModerationStatus) error {
	return s.updateAnnotation(ctx, siteID, id, `moderation = ?`, m)
}

func (s *Store) SoftDeleteAnnotation(ctx context.Context, siteID, id int64) error {
	return s.SetAnnotationVisibility(ctx, siteID, id, domain.VisibilityDeleted)
}

func (s *Store) updateAnnotation(ctx context.Context, siteID, id int64, setClause string, args ...any) error {
	query := `UPDATE annotations SET ` + setClause + `, updated_at = ? WHERE site_id = ? AND id = ?`
	args = append(args, time.Now().Unix(), siteID, id)
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) loadComments(ctx context.Context, siteID int64, annotationIDs []int64, viewerID int64, includeUnapproved bool) (map[int64][]domain.Comment, error) {
	result := make(map[int64][]domain.Comment, len(annotationIDs))
	if len(annotationIDs) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(annotationIDs)), ",")
	args := make([]any, 0, len(annotationIDs)+3)
	args = append(args, siteID)
	for _, id := range annotationIDs {
		args = append(args, id)
	}
	include := 0
	if includeUnapproved {
		include = 1
	}
	args = append(args, viewerID, include)

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.site_id, c.annotation_id, c.user_id, c.content, c.moderation, c.visibility, c.created_at,
		       u.id, u.display_name, u.avatar_url
		FROM comments c JOIN users u ON u.id = c.user_id
		WHERE c.site_id = ? AND c.annotation_id IN (`+placeholders+`)
			AND c.visibility <> 'deleted'
			AND ((c.moderation = 'approved' AND c.visibility = 'visible') OR c.user_id = ? OR ? = 1)
		ORDER BY c.id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			c       domain.Comment
			created int64
		)
		if err := rows.Scan(&c.ID, &c.SiteID, &c.AnnotationID, &c.UserID, &c.Content,
			&c.Moderation, &c.Visibility, &created, &c.Author.ID, &c.Author.Name, &c.Author.AvatarURL); err != nil {
			return nil, err
		}
		c.CreatedAt = fromUnix(created)
		result[c.AnnotationID] = append(result[c.AnnotationID], c)
	}
	return result, rows.Err()
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const userColumns = `id, display_name, avatar_url, status, is_superadmin, created_at`

func scanUser(row interface{ Scan(...any) error }) (domain.User, error) {
	var (
		u       domain.User
		super   int
		created int64
	)
	if err := row.Scan(&u.ID, &u.DisplayName, &u.AvatarURL, &u.Status, &super, &created); err != nil {
		return domain.User{}, err
	}
	u.IsSuperadmin = super != 0
	u.CreatedAt = fromUnix(created)
	return u, nil
}

func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	super := 0
	if u.IsSuperadmin {
		super = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (display_name, avatar_url, status, is_superadmin, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		u.DisplayName, u.AvatarURL, u.Status, super, toUnix(u.CreatedAt))
	if err != nil {
		return domain.User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	return s.UserByID(ctx, id)
}

func (s *Store) UserByID(ctx context.Context, id int64) (domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	return u, err
}

func (s *Store) UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) SetUserSuperadmin(ctx context.Context, id int64, v bool) error {
	value := 0
	if v {
		value = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE users SET is_superadmin = ? WHERE id = ?`, value, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListUsers(ctx context.Context, cursor int64, limit int) ([]domain.User, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+userColumns+` FROM users WHERE id > ? ORDER BY id ASC LIMIT ?`, cursor, limit+1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, u)
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

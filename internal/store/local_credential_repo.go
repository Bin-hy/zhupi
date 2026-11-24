package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const localCredentialColumns = `user_id, username, password_hash, must_change_password, created_at, updated_at`

func scanLocalCredential(row interface{ Scan(...any) error }) (domain.LocalCredential, error) {
	var (
		c       domain.LocalCredential
		must    int
		created int64
		updated int64
	)
	if err := row.Scan(&c.UserID, &c.Username, &c.PasswordHash, &must, &created, &updated); err != nil {
		return domain.LocalCredential{}, err
	}
	c.MustChangePassword = must != 0
	c.CreatedAt = fromUnix(created)
	c.UpdatedAt = fromUnix(updated)
	return c, nil
}

func (s *Store) CreateLocalCredential(ctx context.Context, c domain.LocalCredential) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	must := 0
	if c.MustChangePassword {
		must = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO local_credentials (user_id, username, password_hash, must_change_password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.UserID, c.Username, c.PasswordHash, must, toUnix(c.CreatedAt), toUnix(c.UpdatedAt))
	return err
}

func (s *Store) LocalCredentialByUsername(ctx context.Context, username string) (domain.LocalCredential, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+localCredentialColumns+` FROM local_credentials WHERE username = ? COLLATE NOCASE`, username)
	c, err := scanLocalCredential(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LocalCredential{}, domain.ErrNotFound
	}
	return c, err
}

func (s *Store) LocalCredentialByUserID(ctx context.Context, userID int64) (domain.LocalCredential, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+localCredentialColumns+` FROM local_credentials WHERE user_id = ?`, userID)
	c, err := scanLocalCredential(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.LocalCredential{}, domain.ErrNotFound
	}
	return c, err
}

func (s *Store) CountLocalCredentials(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM local_credentials`).Scan(&n)
	return n, err
}

func (s *Store) UpdateLocalCredentialPassword(ctx context.Context, userID int64, hash string, mustChange bool) error {
	must := 0
	if mustChange {
		must = 1
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE local_credentials SET password_hash = ?, must_change_password = ?, updated_at = ? WHERE user_id = ?`,
		hash, must, time.Now().Unix(), userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

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
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Conflict("username_taken", "用户名已被占用")
		}
		return err
	}
	return nil
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

func (s *Store) ListLocalAdmins(ctx context.Context) ([]domain.LocalAdminAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT lc.user_id, lc.username, u.display_name, u.status, u.is_superadmin,
		       lc.must_change_password, u.created_at
		FROM local_credentials lc
		JOIN users u ON u.id = lc.user_id
		ORDER BY lc.username COLLATE NOCASE ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.LocalAdminAccount, 0)
	for rows.Next() {
		var (
			a       domain.LocalAdminAccount
			super   int
			must    int
			created int64
		)
		if err := rows.Scan(&a.UserID, &a.Username, &a.DisplayName, &a.Status, &super, &must, &created); err != nil {
			return nil, err
		}
		a.IsSuperadmin = super != 0
		a.MustChangePassword = must != 0
		a.CreatedAt = fromUnix(created)
		a.SiteIDs = []int64{}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	siteRows, err := s.db.QueryContext(ctx,
		`SELECT user_id, site_id FROM site_admins ORDER BY user_id ASC, site_id ASC`)
	if err != nil {
		return nil, err
	}
	defer siteRows.Close()
	byUser := map[int64][]int64{}
	for siteRows.Next() {
		var userID, siteID int64
		if err := siteRows.Scan(&userID, &siteID); err != nil {
			return nil, err
		}
		byUser[userID] = append(byUser[userID], siteID)
	}
	if err := siteRows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if ids := byUser[out[i].UserID]; len(ids) > 0 {
			out[i].SiteIDs = ids
		}
	}
	return out, nil
}

func (s *Store) CountActiveSuperadminCredentials(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN local_credentials lc ON lc.user_id = u.id
		WHERE u.is_superadmin = 1 AND u.status = ?`, domain.UserActive).Scan(&n)
	return n, err
}

func (s *Store) CreateLocalAdmin(ctx context.Context, u domain.User, c domain.LocalCredential,
	siteIDs []int64, grantedBy int64) (domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()

	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = u.CreatedAt
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	status := u.Status
	if status == "" {
		status = domain.UserActive
	}
	super := 0
	if u.IsSuperadmin {
		super = 1
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (display_name, avatar_url, status, is_superadmin, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		u.DisplayName, u.AvatarURL, status, super, toUnix(u.CreatedAt))
	if err != nil {
		return domain.User{}, err
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}

	must := 0
	if c.MustChangePassword {
		must = 1
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO local_credentials (user_id, username, password_hash, must_change_password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, c.Username, c.PasswordHash, must, toUnix(c.CreatedAt), toUnix(c.UpdatedAt)); err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.Conflict("username_taken", "用户名已被占用")
		}
		return domain.User{}, err
	}

	for _, siteID := range siteIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO site_admins (site_id, user_id, granted_by, created_at)
			VALUES (?, ?, ?, ?)`,
			siteID, userID, grantedBy, toUnix(now)); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}
	return s.UserByID(ctx, userID)
}

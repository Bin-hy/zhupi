package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const identityColumns = `id, user_id, provider, site_id, issuer, subject, handle, email, email_verified, avatar_url, created_at, last_login_at`

func scanIdentity(row interface{ Scan(...any) error }) (domain.Identity, error) {
	var (
		i        domain.Identity
		verified int
		created  int64
		last     int64
	)
	if err := row.Scan(&i.ID, &i.UserID, &i.Provider, &i.SiteID, &i.Issuer, &i.Subject,
		&i.Handle, &i.Email, &verified, &i.AvatarURL, &created, &last); err != nil {
		return domain.Identity{}, err
	}
	i.EmailVerified = verified != 0
	i.CreatedAt = fromUnix(created)
	i.LastLoginAt = fromUnix(last)
	return i, nil
}

func (s *Store) CreateIdentity(ctx context.Context, i domain.Identity) (domain.Identity, error) {
	now := time.Now()
	if i.CreatedAt.IsZero() {
		i.CreatedAt = now
	}
	if i.LastLoginAt.IsZero() {
		i.LastLoginAt = now
	}
	verified := 0
	if i.EmailVerified {
		verified = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO identities (user_id, provider, site_id, issuer, subject, handle, email, email_verified, avatar_url, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		i.UserID, i.Provider, i.SiteID, i.Issuer, i.Subject, i.Handle, i.Email, verified, i.AvatarURL,
		toUnix(i.CreatedAt), toUnix(i.LastLoginAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Identity{}, domain.Conflict("identity_conflict", "该登录身份已存在")
		}
		return domain.Identity{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Identity{}, err
	}
	return s.IdentityByID(ctx, id)
}

func (s *Store) IdentityByID(ctx context.Context, id int64) (domain.Identity, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+identityColumns+` FROM identities WHERE id = ?`, id)
	i, err := scanIdentity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Identity{}, domain.ErrNotFound
	}
	return i, err
}

func (s *Store) IdentityByProviderSubject(ctx context.Context, provider domain.ProviderType, siteID int64, issuer, subject string) (domain.Identity, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+identityColumns+`
		FROM identities WHERE provider = ? AND site_id = ? AND issuer = ? AND subject = ?`,
		provider, siteID, issuer, subject)
	i, err := scanIdentity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Identity{}, domain.ErrNotFound
	}
	return i, err
}

func (s *Store) IdentityByHandle(ctx context.Context, provider domain.ProviderType, handle string) (domain.Identity, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+identityColumns+`
		FROM identities WHERE provider = ? AND lower(handle) = lower(?) ORDER BY id ASC LIMIT 1`, provider, handle)
	i, err := scanIdentity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Identity{}, domain.ErrNotFound
	}
	return i, err
}

func (s *Store) ListIdentitiesByUser(ctx context.Context, userID int64) ([]domain.Identity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+identityColumns+`
		FROM identities WHERE user_id = ? ORDER BY id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Identity
	for rows.Next() {
		i, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) DeleteIdentity(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM identities WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) TouchIdentityLogin(ctx context.Context, id int64, at int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE identities SET last_login_at = ? WHERE id = ?`, at, id)
	return err
}

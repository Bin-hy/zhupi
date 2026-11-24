package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/zhupi/internal/domain"
)

const sessionColumns = `id, token_hash, user_id, user_agent, ip, created_at, expires_at, last_seen_at, revoked_at`

func scanSession(row interface{ Scan(...any) error }) (domain.Session, error) {
	var (
		s       domain.Session
		created int64
		expires int64
		last    int64
		revoked sql.NullInt64
	)
	if err := row.Scan(&s.ID, &s.TokenHash, &s.UserID, &s.UserAgent, &s.IP,
		&created, &expires, &last, &revoked); err != nil {
		return domain.Session{}, err
	}
	s.CreatedAt = fromUnix(created)
	s.ExpiresAt = fromUnix(expires)
	s.LastSeenAt = fromUnix(last)
	if revoked.Valid {
		t := fromUnix(revoked.Int64)
		s.RevokedAt = &t
	}
	return s, nil
}

func (s *Store) CreateSession(ctx context.Context, sess domain.Session) error {
	now := time.Now()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.LastSeenAt.IsZero() {
		sess.LastSeenAt = sess.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, user_agent, ip, created_at, expires_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.TokenHash, sess.UserID, sess.UserAgent, sess.IP,
		toUnix(sess.CreatedAt), toUnix(sess.ExpiresAt), toUnix(sess.LastSeenAt))
	return err
}

func (s *Store) SessionByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+sessionColumns+` FROM sessions WHERE token_hash = ?`, hash)
	sess, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrNotFound
	}
	return sess, err
}

func (s *Store) ListSessionsByUser(ctx context.Context, userID int64) ([]domain.Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+sessionColumns+`
		FROM sessions
		WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?
		ORDER BY id DESC`, userID, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *Store) RevokeSession(ctx context.Context, id, userID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		time.Now().Unix(), id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) RevokeAllSessionsForUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		time.Now().Unix(), userID)
	return err
}

// RevokeOtherSessionsForUser 吊销除 keepSessionID 外的全部会话。
func (s *Store) RevokeOtherSessionsForUser(ctx context.Context, userID, keepSessionID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND id <> ? AND revoked_at IS NULL`,
		time.Now().Unix(), userID, keepSessionID)
	return err
}

func (s *Store) TouchSession(ctx context.Context, id int64, at int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id = ?`, at, id)
	return err
}

func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= ? OR revoked_at IS NOT NULL`,
		time.Now().Unix())
	return err
}

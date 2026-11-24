package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

func (s *Store) CreateAuthFlow(ctx context.Context, f domain.AuthFlow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_flows (state_hash, mode, provider_type, provider_id, issuer, site_id, user_id, return_to, code_verifier, nonce, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.StateHash, f.Mode, f.ProviderType, f.ProviderID, f.Issuer, f.SiteID, f.UserID,
		f.ReturnTo, f.CodeVerifier, f.Nonce, toUnix(f.ExpiresAt))
	return err
}

func (s *Store) ConsumeAuthFlow(ctx context.Context, stateHash string) (domain.AuthFlow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AuthFlow{}, err
	}
	defer tx.Rollback()

	var (
		f       domain.AuthFlow
		expires int64
	)
	err = tx.QueryRowContext(ctx, `
		SELECT state_hash, mode, provider_type, provider_id, issuer, site_id, user_id, return_to, code_verifier, nonce, expires_at
		FROM auth_flows WHERE state_hash = ?`, stateHash).
		Scan(&f.StateHash, &f.Mode, &f.ProviderType, &f.ProviderID, &f.Issuer, &f.SiteID, &f.UserID,
			&f.ReturnTo, &f.CodeVerifier, &f.Nonce, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuthFlow{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.AuthFlow{}, err
	}
	f.ExpiresAt = fromUnix(expires)
	if time.Now().After(f.ExpiresAt) {
		tx.ExecContext(ctx, `DELETE FROM auth_flows WHERE state_hash = ?`, stateHash)
		tx.Commit()
		return domain.AuthFlow{}, domain.NewError("auth_flow_expired", "登录状态已过期，请重试", 400)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_flows WHERE state_hash = ?`, stateHash); err != nil {
		return domain.AuthFlow{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.AuthFlow{}, err
	}
	return f, nil
}

func (s *Store) PurgeExpiredAuthFlows(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_flows WHERE expires_at <= ?`, time.Now().Unix())
	return err
}

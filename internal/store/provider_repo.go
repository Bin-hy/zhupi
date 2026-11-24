package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
)

const providerColumns = `id, site_id, name, issuer, client_id, client_secret_cipher, scopes, allowed_email_domains, enabled, created_at, updated_at`

func scanProvider(row interface{ Scan(...any) error }) (domain.OIDCProvider, error) {
	var (
		p                     domain.OIDCProvider
		scopesRaw, domainsRaw string
		enabled               int
		created, updated      int64
	)
	if err := row.Scan(&p.ID, &p.SiteID, &p.Name, &p.Issuer, &p.ClientID, &p.ClientSecretCipher,
		&scopesRaw, &domainsRaw, &enabled, &created, &updated); err != nil {
		return domain.OIDCProvider{}, err
	}
	if err := json.Unmarshal([]byte(scopesRaw), &p.Scopes); err != nil {
		return domain.OIDCProvider{}, err
	}
	if err := json.Unmarshal([]byte(domainsRaw), &p.AllowedEmailDomains); err != nil {
		return domain.OIDCProvider{}, err
	}
	p.Enabled = enabled != 0
	p.CreatedAt = fromUnix(created)
	p.UpdatedAt = fromUnix(updated)
	return p, nil
}

func (s *Store) CreateProvider(ctx context.Context, p domain.OIDCProvider) (domain.OIDCProvider, error) {
	scopes, err := json.Marshal(p.Scopes)
	if err != nil {
		return domain.OIDCProvider{}, err
	}
	domains, err := json.Marshal(p.AllowedEmailDomains)
	if err != nil {
		return domain.OIDCProvider{}, err
	}
	now := time.Now()
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO oidc_providers (site_id, name, issuer, client_id, client_secret_cipher, scopes, allowed_email_domains, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.SiteID, p.Name, p.Issuer, p.ClientID, p.ClientSecretCipher, string(scopes), string(domains), enabled, toUnix(now), toUnix(now))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.OIDCProvider{}, domain.Conflict("provider_conflict", "同一站点下已存在相同 issuer/clientId 的提供商")
		}
		return domain.OIDCProvider{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.OIDCProvider{}, err
	}
	return s.ProviderByID(ctx, p.SiteID, id)
}

func (s *Store) ProviderByID(ctx context.Context, siteID, id int64) (domain.OIDCProvider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+providerColumns+`
		FROM oidc_providers WHERE site_id = ? AND id = ?`, siteID, id)
	p, err := scanProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.OIDCProvider{}, domain.ErrNotFound
	}
	return p, err
}

func (s *Store) ListProvidersBySite(ctx context.Context, siteID int64, onlyEnabled bool) ([]domain.OIDCProvider, error) {
	query := `SELECT ` + providerColumns + ` FROM oidc_providers WHERE site_id = ?`
	if onlyEnabled {
		query += ` AND enabled = 1`
	}
	query += ` ORDER BY id ASC`
	rows, err := s.db.QueryContext(ctx, query, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.OIDCProvider, 0)
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdateProvider(ctx context.Context, p domain.OIDCProvider) error {
	scopes, err := json.Marshal(p.Scopes)
	if err != nil {
		return err
	}
	domains, err := json.Marshal(p.AllowedEmailDomains)
	if err != nil {
		return err
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE oidc_providers
		SET name = ?, issuer = ?, client_id = ?, client_secret_cipher = ?, scopes = ?, allowed_email_domains = ?, enabled = ?, updated_at = ?
		WHERE site_id = ? AND id = ?`,
		p.Name, p.Issuer, p.ClientID, p.ClientSecretCipher, string(scopes), string(domains), enabled,
		time.Now().Unix(), p.SiteID, p.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Conflict("provider_conflict", "同一站点下已存在相同 issuer/clientId 的提供商")
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteProvider(ctx context.Context, siteID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM oidc_providers WHERE site_id = ? AND id = ?`, siteID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

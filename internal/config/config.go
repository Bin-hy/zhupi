package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr           string
	DBPath         string
	PublicBaseURL  string
	SecretKey      []byte
	SessionTTL     time.Duration
	GitHubClientID string
	GitHubSecret   string
	// SuperadminGitHubLogins 命中时，该 GitHub 用户登录后自动成为超级管理员。
	SuperadminGitHubLogins map[string]bool
	TrustProxy             bool
	AdminOrigins           []string
	DevLogin               bool
	AllowMissingOrigin     bool
	ReadRatePerMin         int
	WriteRatePerMin        int
	WriteUserRatePerMin    int
}

func Load() (*Config, error) {
	c := &Config{
		Addr:                   getenv("ADDR", ":8080"),
		DBPath:                 getenv("DB_PATH", "annotations.db"),
		PublicBaseURL:          strings.TrimRight(getenv("PUBLIC_BASE_URL", "http://127.0.0.1:8080"), "/"),
		GitHubClientID:         os.Getenv("GITHUB_CLIENT_ID"),
		GitHubSecret:           os.Getenv("GITHUB_CLIENT_SECRET"),
		SuperadminGitHubLogins: map[string]bool{},
		TrustProxy:             getenv("TRUST_PROXY", "") == "1",
		DevLogin:               getenv("DEV_LOGIN", "") == "1",
		AllowMissingOrigin:     getenv("ALLOW_MISSING_ORIGIN", "") == "1",
	}

	if u, err := url.Parse(c.PublicBaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("PUBLIC_BASE_URL 必须是绝对地址，例如 https://annot.example.com")
	}

	secret, err := decodeSecretKey(os.Getenv("SECRET_KEY"))
	if err != nil {
		return nil, err
	}
	c.SecretKey = secret

	hours, err := strconv.Atoi(getenv("SESSION_TTL_HOURS", "720"))
	if err != nil || hours <= 0 {
		return nil, fmt.Errorf("SESSION_TTL_HOURS 必须是正整数")
	}
	c.SessionTTL = time.Duration(hours) * time.Hour

	for _, name := range splitCSV(os.Getenv("SUPERADMIN_GITHUB_LOGINS")) {
		c.SuperadminGitHubLogins[strings.ToLower(name)] = true
	}
	for _, origin := range splitCSV(os.Getenv("ADMIN_ORIGINS")) {
		c.AdminOrigins = append(c.AdminOrigins, strings.TrimRight(origin, "/"))
	}

	c.ReadRatePerMin = positiveInt(getenv("READ_RATE_PER_MIN", "300"), 300)
	c.WriteRatePerMin = positiveInt(getenv("WRITE_RATE_PER_MIN", "60"), 60)
	c.WriteUserRatePerMin = positiveInt(getenv("WRITE_USER_RATE_PER_MIN", "30"), 30)

	return c, nil
}

// GitHubConfigured 表示部署级 GitHub OAuth App 是否可用。
func (c *Config) GitHubConfigured() bool {
	return c.GitHubClientID != "" && c.GitHubSecret != ""
}

func decodeSecretKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("SECRET_KEY 必须配置（32 字节的 hex 或 base64，可用 openssl rand -hex 32 生成）")
	}
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(raw); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("SECRET_KEY 无效：需要 32 字节的 hex 或 base64")
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func positiveInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

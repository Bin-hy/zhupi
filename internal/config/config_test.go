package config

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	key := hex.EncodeToString(make([]byte, 32))
	t.Setenv("ADDR", ":9090")
	t.Setenv("DB_PATH", "/tmp/x.db")
	t.Setenv("PUBLIC_BASE_URL", "https://annot.example.com/")
	t.Setenv("SECRET_KEY", key)
	t.Setenv("SESSION_TTL_HOURS", "24")
	t.Setenv("SUPERADMIN_GITHUB_LOGINS", "Alice, bob")
	t.Setenv("ADMIN_ORIGINS", "https://admin.example.com")
	t.Setenv("DEV_LOGIN", "1")
	t.Setenv("TRUST_PROXY", "1")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Addr != ":9090" || c.DBPath != "/tmp/x.db" {
		t.Fatalf("unexpected addr/db: %+v", c)
	}
	if c.PublicBaseURL != "https://annot.example.com" {
		t.Fatalf("public base url = %q", c.PublicBaseURL)
	}
	if len(c.SecretKey) != 32 {
		t.Fatalf("secret key len = %d", len(c.SecretKey))
	}
	if c.SessionTTL != 24*time.Hour {
		t.Fatalf("session ttl = %v", c.SessionTTL)
	}
	if !c.SuperadminGitHubLogins["alice"] || !c.SuperadminGitHubLogins["bob"] {
		t.Fatalf("superadmins = %+v", c.SuperadminGitHubLogins)
	}
	if !c.DevLogin || !c.TrustProxy {
		t.Fatalf("flags not loaded: %+v", c)
	}
}

func TestLoadRequiresSecret(t *testing.T) {
	t.Setenv("SECRET_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("missing SECRET_KEY should fail")
	}
}

func TestLoadRejectsInvalidBaseURL(t *testing.T) {
	t.Setenv("SECRET_KEY", hex.EncodeToString(make([]byte, 32)))
	t.Setenv("PUBLIC_BASE_URL", "not-a-url")
	if _, err := Load(); err == nil {
		t.Fatal("invalid PUBLIC_BASE_URL should fail")
	}
}

func TestDecodeSecretKeyBase64(t *testing.T) {
	raw := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" // 32 bytes zero
	if b, err := decodeSecretKey(raw); err != nil || len(b) != 32 {
		t.Fatalf("decode base64: %v len=%d", err, len(b))
	}
}

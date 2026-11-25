package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/crypto"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
	"github.com/Bin-hy/zhupi/internal/store"
)

func TestProviderInputHelpers(t *testing.T) {
	scopes := providerScopes("openid, email, openid")
	if len(scopes) != 2 || scopes[0] != "openid" || scopes[1] != "email" {
		t.Fatalf("scopes = %#v", scopes)
	}
	if got := providerScopes(" , "); len(got) != 3 || got[0] != "openid" {
		t.Fatalf("default scopes = %#v", got)
	}

	domains := normalizeProviderDomains("@Example.COM, example.com, Foo.ORG")
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "foo.org" {
		t.Fatalf("domains = %#v", domains)
	}

	for raw, want := range map[string]bool{
		"true": true, "1": true, "YES": true,
		"false": false, "0": false, "no": false,
	} {
		got, ok := parseProviderEnabled(raw)
		if !ok || got != want {
			t.Fatalf("parseProviderEnabled(%q) = %v, %v", raw, got, ok)
		}
	}
	if _, ok := parseProviderEnabled("sometimes"); ok {
		t.Fatal("invalid enabled value should be rejected")
	}
}

func TestReadSecretFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oidc-secret")
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readSecret("", path)
	if err != nil || got != "file-secret" {
		t.Fatalf("readSecret = %q, %v", got, err)
	}
	got, err = readSecret("inline-secret", "")
	if err != nil || got != "inline-secret" {
		t.Fatalf("inline secret = %q, %v", got, err)
	}
}

func TestProviderCLIAddDisableDelete(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "zhupi.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	site, err := db.CreateSite(ctx, domain.Site{
		Key: "site-a", Name: "Site A", DisplayName: "Site A",
		Status: domain.SiteActive, ModerationMode: domain.ModerationPost,
		ThemeColor: "#087f5b", DefaultLocale: "zh-CN",
		Origins: []string{"https://site.test"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var discoveryRequests atomic.Int32
	srv := newOIDCServer(t, &discoveryRequests)
	box, err := crypto.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{SecretKey: make([]byte, 32), PublicBaseURL: "https://zhupi.test"}
	authSvc := service.NewAuthService(cfg, db, db, db, db, db, db, box, nil, authpkg.NewOIDCClient(srv.Client()))

	runProviderCLI(ctx, db, authSvc, []string{
		"add",
		"--site", site.Key,
		"--name", "Company SSO",
		"--issuer", srv.URL,
		"--client-id", "zhupi",
		"--client-secret", "s3cret",
		"--scopes", "openid, email, openid",
		"--allowed-email-domains", "@Example.COM, example.com, Foo.ORG",
	})

	providers, err := db.ListProvidersBySite(ctx, site.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 1 {
		t.Fatalf("providers = %#v", providers)
	}
	p := providers[0]
	if p.Name != "Company SSO" || p.Issuer != srv.URL || p.ClientID != "zhupi" || !p.Enabled {
		t.Fatalf("provider = %+v", p)
	}
	if len(p.Scopes) != 2 || len(p.AllowedEmailDomains) != 2 ||
		p.AllowedEmailDomains[0] != "example.com" || p.AllowedEmailDomains[1] != "foo.org" {
		t.Fatalf("normalized provider = %+v", p)
	}
	secret, err := box.Open(p.ClientSecretCipher)
	if err != nil || string(secret) != "s3cret" {
		t.Fatalf("secret = %q, %v", secret, err)
	}

	// 使用全新的 OIDC client：只有 issuer 变化时才应触发 Discovery。
	discoveryRequests.Store(0)
	updateAuthSvc := service.NewAuthService(cfg, db, db, db, db, db, db, box, nil, authpkg.NewOIDCClient(srv.Client()))
	runProviderCLI(ctx, db, updateAuthSvc, []string{
		"update", "--id", strconv.FormatInt(p.ID, 10), "--enabled", "false",
	})
	if discoveryRequests.Load() != 0 {
		t.Fatalf("disable provider should not trigger discovery, requests = %d", discoveryRequests.Load())
	}
	p, err = db.ProviderByID(ctx, site.ID, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Enabled {
		t.Fatal("provider should be disabled")
	}

	runProviderCLI(ctx, db, authSvc, []string{"delete", "--id", strconv.FormatInt(p.ID, 10)})
	providers, err = db.ListProvidersBySite(ctx, site.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 0 {
		t.Fatalf("providers after delete = %#v", providers)
	}
}

func newOIDCServer(t *testing.T, requests *atomic.Int32) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/authorize",
			"token_endpoint":                        srv.URL + "/token",
			"jwks_uri":                              srv.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

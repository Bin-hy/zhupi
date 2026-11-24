package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	authpkg "github.com/Bin-hy/go-campus/annotation-service/internal/auth"
	"github.com/Bin-hy/go-campus/annotation-service/internal/config"
	"github.com/Bin-hy/go-campus/annotation-service/internal/crypto"
	"github.com/Bin-hy/go-campus/annotation-service/internal/domain"
	"github.com/Bin-hy/go-campus/annotation-service/internal/service"
	"github.com/Bin-hy/go-campus/annotation-service/internal/store"
)

type testEnv struct {
	ts    *httptest.Server
	store *store.Store
	sites *service.SiteService
	cfg   *config.Config
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := &config.Config{
		Addr:                ":0",
		DBPath:              "x",
		PublicBaseURL:       "http://127.0.0.1",
		SecretKey:           bytes.Repeat([]byte{1}, 32),
		SessionTTL:          time.Hour,
		DevLogin:            true,
		ReadRatePerMin:      100000,
		WriteRatePerMin:     100000,
		WriteUserRatePerMin: 100000,
		AdminOrigins:        []string{"http://admin.test"},
	}
	box, _ := crypto.New(cfg.SecretKey)
	github := authpkg.NewGitHubClient(authpkg.GitHubConfig{}, nil)
	oidc := authpkg.NewOIDCClient(nil)
	sites := service.NewSiteService(st, false)
	authSvc := service.NewAuthService(cfg, st, st, st, st, st, box, github, oidc)
	annotations := service.NewAnnotationService(st, st)
	moderation := service.NewModerationService(st, st, st, st, st)
	admin := service.NewAdminService(st, st, st, st, st)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, st, sites, authSvc, annotations, moderation, admin, log)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return &testEnv{ts: ts, store: st, sites: sites, cfg: cfg}
}

func (e *testEnv) site(t *testing.T, name, origin string, mode domain.ModerationMode) domain.Site {
	t.Helper()
	site, err := e.sites.Create(context.Background(), service.CreateSiteInput{
		Name: name, DisplayName: name, Origins: []string{origin}, ModerationMode: mode,
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	return site
}

func (e *testEnv) login(t *testing.T, name string) string {
	t.Helper()
	resp, body := e.do(t, http.MethodPost, "/api/v1/auth/dev/login", "", "", "", map[string]any{"name": name})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dev login %s: %d %s", name, resp.StatusCode, body)
	}
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &out)
	if out.Token == "" {
		t.Fatal("empty token")
	}
	return out.Token
}

func (e *testEnv) do(t *testing.T, method, path, token, siteKey, origin string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, e.ts.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if siteKey != "" {
		req.Header.Set("X-Site-Key", siteKey)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestSiteConfigOrigin(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)

	resp, body := e.do(t, http.MethodGet, "/api/v1/site-config", "", site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("site config: %d %s", resp.StatusCode, body)
	}
	var cfg struct {
		SiteKey string `json:"siteKey"`
	}
	_ = json.Unmarshal(body, &cfg)
	if cfg.SiteKey != site.Key {
		t.Fatalf("siteKey = %q", cfg.SiteKey)
	}

	resp, _ = e.do(t, http.MethodGet, "/api/v1/site-config", "", site.Key, "http://evil.test", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong origin status = %d", resp.StatusCode)
	}

	resp, _ = e.do(t, http.MethodGet, "/api/v1/site-config", "", "", "http://a.test", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing site key status = %d", resp.StatusCode)
	}
}

func TestAnnotationVisibilityAndModeration(t *testing.T) {
	e := newTestEnv(t)
	siteA := e.site(t, "a", "http://a.test", domain.ModerationPost)
	siteB := e.site(t, "b", "http://b.test", domain.ModerationPre)
	alice := e.login(t, "alice")

	// 站点 A：先发后审，匿名立即可见。
	resp, body := e.do(t, http.MethodPost, "/api/v1/annotations", alice, siteA.Key, "http://a.test", map[string]any{
		"pageKey": "/doc",
		"anchor":  map[string]any{"quote": map[string]any{"exact": "hello"}},
		"content": "A 的批注",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create A: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", "", siteA.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("A 的批注")) {
		t.Fatalf("A anon list: %d %s", resp.StatusCode, body)
	}

	// 站点 B：先审后发，匿名不可见，作者可见。
	resp, body = e.do(t, http.MethodPost, "/api/v1/annotations", alice, siteB.Key, "http://b.test", map[string]any{
		"pageKey": "/doc",
		"anchor":  map[string]any{"quote": map[string]any{"exact": "pending"}},
		"content": "B 的待审",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create B: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", "", siteB.Key, "http://b.test", nil)
	if bytes.Contains(body, []byte("B 的待审")) {
		t.Fatalf("pending should be hidden from anon: %s", body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", alice, siteB.Key, "http://b.test", nil)
	if !bytes.Contains(body, []byte("B 的待审")) {
		t.Fatalf("author should see pending: %s", body)
	}

	// 跨站点写操作不可见。
	resp, _ = e.do(t, http.MethodPatch, "/api/v1/annotations/2", alice, siteA.Key, "http://a.test", map[string]any{"status": "resolved"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-site patch status = %d", resp.StatusCode)
	}
}

func TestModerationApproveShowsFirstComment(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "b", "http://b.test", domain.ModerationPre)
	alice := e.login(t, "alice")
	_ = e.store.SetUserSuperadmin(context.Background(), 1, true)

	_, body := e.do(t, http.MethodPost, "/api/v1/annotations", alice, site.Key, "http://b.test", map[string]any{
		"pageKey": "/doc",
		"anchor":  map[string]any{"quote": map[string]any{"exact": "pending"}},
		"content": "待审正文",
	})
	var created domain.Annotation
	_ = json.Unmarshal(body, &created)
	if created.Moderation != domain.ModerationPending {
		t.Fatalf("moderation = %s", created.Moderation)
	}

	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(site.ID)+"/annotations/"+itoa(created.ID)+"/moderate",
		alice, "", "", map[string]any{"action": "approve"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", "", site.Key, "http://b.test", nil)
	if !bytes.Contains(body, []byte("待审正文")) {
		t.Fatalf("approved content should be visible: %s", body)
	}
}

func TestSiteAdminScopeAndBan(t *testing.T) {
	e := newTestEnv(t)
	siteA := e.site(t, "a", "http://a.test", domain.ModerationPost)
	siteB := e.site(t, "b", "http://b.test", domain.ModerationPost)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	_ = e.store.SetUserSuperadmin(context.Background(), 1, true) // alice = 1

	// 超管把 bob(=2) 设为站点 B 管理员。
	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(siteB.ID)+"/admins", alice, "", "", map[string]any{"userId": 2})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add admin: %d %s", resp.StatusCode, body)
	}

	// bob 可读 B 的队列，不能读 A 的队列。
	resp, _ = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(siteB.ID)+"/queue", bob, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob queue B status = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(siteA.ID)+"/queue", bob, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob queue A status = %d", resp.StatusCode)
	}

	// bob 不能给 A 分配管理员（非超管）。
	resp, _ = e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(siteA.ID)+"/admins", bob, "", "", map[string]any{"userId": 2})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob add admin status = %d", resp.StatusCode)
	}

	// 站点 B 封禁 bob：B 写 403，A 写正常。
	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(siteB.ID)+"/bans/2", alice, "", "", map[string]any{"reason": "spam"})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ban: %d %s", resp.StatusCode, body)
	}
	resp, _ = e.do(t, http.MethodPost, "/api/v1/annotations", bob, siteB.Key, "http://b.test", map[string]any{
		"pageKey": "/doc", "anchor": map[string]any{"quote": map[string]any{"exact": "x"}}, "content": "c",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("banned site write status = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodPost, "/api/v1/annotations", bob, siteA.Key, "http://a.test", map[string]any{
		"pageKey": "/doc", "anchor": map[string]any{"quote": map[string]any{"exact": "x"}}, "content": "c",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("other site write status = %d", resp.StatusCode)
	}
}

func TestSessionRevocation(t *testing.T) {
	e := newTestEnv(t)
	alice := e.login(t, "alice")

	resp, body := e.do(t, http.MethodGet, "/api/v1/me", alice, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me: %d %s", resp.StatusCode, body)
	}
	resp, _ = e.do(t, http.MethodDelete, "/api/v1/me/sessions", alice, "", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke all status = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/me", alice, "", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("after revoke status = %d", resp.StatusCode)
	}
}

func (e *testEnv) createAnnotation(t *testing.T, token string, site domain.Site, origin, content, exact string) domain.Annotation {
	t.Helper()
	resp, body := e.do(t, http.MethodPost, "/api/v1/annotations", token, site.Key, origin, map[string]any{
		"pageKey": "/doc",
		"anchor":  map[string]any{"quote": map[string]any{"exact": exact}},
		"content": content,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create annotation: %d %s", resp.StatusCode, body)
	}
	var a domain.Annotation
	if err := json.Unmarshal(body, &a); err != nil {
		t.Fatalf("decode annotation: %v", err)
	}
	return a
}

func (e *testEnv) listAnnotations(t *testing.T, token, siteKey, origin, filter string) []domain.Annotation {
	t.Helper()
	path := "/api/v1/annotations?page=%2Fdoc"
	if filter != "" {
		path += "&filter=" + filter
	}
	resp, body := e.do(t, http.MethodGet, path, token, siteKey, origin, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list annotations (%s): %d %s", filter, resp.StatusCode, body)
	}
	var out struct {
		Annotations []domain.Annotation `json:"annotations"`
	}
	_ = json.Unmarshal(body, &out)
	return out.Annotations
}

func TestListFilters(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	a1 := e.createAnnotation(t, alice, site, "http://a.test", "alice open", "alpha")
	e.createAnnotation(t, alice, site, "http://a.test", "alice resolved", "beta")
	e.createAnnotation(t, bob, site, "http://a.test", "bob open", "gamma")

	resp, _ := e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a1.ID), alice, site.Key, "http://a.test", map[string]any{"status": "resolved"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resolve status = %d", resp.StatusCode)
	}

	if got := len(e.listAnnotations(t, "", site.Key, "http://a.test", "all")); got != 3 {
		t.Fatalf("all = %d", got)
	}
	if got := len(e.listAnnotations(t, alice, site.Key, "http://a.test", "mine")); got != 2 {
		t.Fatalf("alice mine = %d", got)
	}
	if got := len(e.listAnnotations(t, bob, site.Key, "http://a.test", "mine")); got != 1 {
		t.Fatalf("bob mine = %d", got)
	}
	if got := len(e.listAnnotations(t, "", site.Key, "http://a.test", "resolved")); got != 1 {
		t.Fatalf("resolved = %d", got)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc&filter=mine", "", site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("mine without auth = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc&filter=bogus", "", site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid filter = %d", resp.StatusCode)
	}
}

func TestPendingFilter(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "pre", "http://pre.test", domain.ModerationPre)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	e.createAnnotation(t, alice, site, "http://pre.test", "pending alice", "p1")
	e.createAnnotation(t, bob, site, "http://pre.test", "pending bob", "p2")

	if got := len(e.listAnnotations(t, "", site.Key, "http://pre.test", "all")); got != 0 {
		t.Fatalf("anon all = %d", got)
	}
	if got := len(e.listAnnotations(t, alice, site.Key, "http://pre.test", "pending")); got != 1 {
		t.Fatalf("alice pending = %d", got)
	}
	if got := len(e.listAnnotations(t, alice, site.Key, "http://pre.test", "all")); got != 1 {
		t.Fatalf("author all = %d", got)
	}
}

func TestUpdateAnnotationAnchor(t *testing.T) {
	e := newTestEnv(t)
	siteA := e.site(t, "a", "http://a.test", domain.ModerationPost)
	siteB := e.site(t, "b", "http://b.test", domain.ModerationPost)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	a := e.createAnnotation(t, alice, siteA, "http://a.test", "content", "old text")

	newAnchor := map[string]any{"quote": map[string]any{"exact": "new text", "prefix": "pre", "suffix": "suf"}}
	resp, body := e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a.ID), alice, siteA.Key, "http://a.test", map[string]any{"anchor": newAnchor})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relocate: %d %s", resp.StatusCode, body)
	}
	var updated domain.Annotation
	_ = json.Unmarshal(body, &updated)
	if updated.QuoteExact != "new text" || updated.Anchor.Quote.Prefix != "pre" {
		t.Fatalf("unexpected updated annotation: %+v", updated)
	}

	resp, _ = e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a.ID), bob, siteA.Key, "http://a.test", map[string]any{"anchor": newAnchor})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob relocate = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a.ID), alice, siteB.Key, "http://b.test", map[string]any{"anchor": newAnchor})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-site relocate = %d", resp.StatusCode)
	}
	resp, _ = e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a.ID), alice, siteA.Key, "http://a.test", map[string]any{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty patch = %d", resp.StatusCode)
	}
}

func TestSiteConfigExposesIssuer(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	if _, err := e.store.CreateProvider(context.Background(), domain.OIDCProvider{
		SiteID: site.ID, Name: "SSO", Issuer: "https://sso.test", ClientID: "cid",
		ClientSecretCipher: []byte("cipher"), Scopes: []string{"openid"}, Enabled: true,
	}); err != nil {
		t.Fatalf("create provider: %v", err)
	}
	resp, body := e.do(t, http.MethodGet, "/api/v1/site-config", "", site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("site config: %d %s", resp.StatusCode, body)
	}
	var cfg struct {
		Providers []struct {
			Type   string `json:"type"`
			Issuer string `json:"issuer"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(body, &cfg)
	if len(cfg.Providers) != 1 || cfg.Providers[0].Type != "oidc" || cfg.Providers[0].Issuer != "https://sso.test" {
		t.Fatalf("providers = %+v", cfg.Providers)
	}
	if bytes.Contains(body, []byte("cipher")) {
		t.Fatalf("secret leaked: %s", body)
	}
}

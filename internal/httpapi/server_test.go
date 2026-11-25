package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/crypto"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
	"github.com/Bin-hy/zhupi/internal/store"
)

type testEnv struct {
	ts    *httptest.Server
	store *store.Store
	sites *service.SiteService
	auth  *service.AuthService
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
		Addr:                 ":0",
		DBPath:               "x",
		PublicBaseURL:        "http://127.0.0.1",
		SecretKey:            bytes.Repeat([]byte{1}, 32),
		SessionTTL:           time.Hour,
		DevLogin:             true,
		ReadRatePerMin:       100000,
		WriteRatePerMin:      100000,
		WriteUserRatePerMin:  100000,
		LocalLoginRatePerMin: 100000,
		AdminOrigins:         []string{"http://admin.test"},
	}
	box, _ := crypto.New(cfg.SecretKey)
	github := authpkg.NewGitHubClient(authpkg.GitHubConfig{}, nil)
	oidc := authpkg.NewOIDCClient(nil)
	sites := service.NewSiteService(st, false)
	authSvc := service.NewAuthService(cfg, st, st, st, st, st, st, box, github, oidc)
	annotations := service.NewAnnotationService(st, st)
	moderation := service.NewModerationService(st, st, st, st, st)
	admin := service.NewAdminService(st, st, st, st, st, st)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, st, sites, authSvc, annotations, moderation, admin, log)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return &testEnv{ts: ts, store: st, sites: sites, auth: authSvc, cfg: cfg}
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

func TestAdminQueueCommentContextCountsAndSearch(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "pre", "http://pre.test", domain.ModerationPre)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	_ = e.store.SetUserSuperadmin(context.Background(), 1, true)

	a := e.createAnnotation(t, alice, site, "http://pre.test", "批注正文", "被引用的原文")
	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(site.ID)+"/annotations/"+itoa(a.ID)+"/moderate",
		alice, "", "", map[string]any{"action": "approve"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve annotation: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodPost, "/api/v1/annotations/"+itoa(a.ID)+"/comments", bob, site.Key, "http://pre.test",
		map[string]any{"content": "这是一条待审评论"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create comment: %d %s", resp.StatusCode, body)
	}

	type queueResult struct {
		Annotations []domain.Annotation `json:"annotations"`
		Comments    []struct {
			domain.Comment
			Parent *struct {
				ID         int64  `json:"id"`
				PageKey    string `json:"pageKey"`
				PageTitle  string `json:"pageTitle"`
				QuoteExact string `json:"quoteExact"`
				Body       string `json:"body"`
			} `json:"parent"`
		} `json:"comments"`
		Counts struct {
			Annotations struct{ Pending, Approved, Rejected int } `json:"annotations"`
			Comments    struct{ Pending, Approved, Rejected int } `json:"comments"`
		} `json:"counts"`
	}
	get := func(query string) queueResult {
		t.Helper()
		resp, body := e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(site.ID)+"/queue?status=pending"+query, alice, "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("queue %s: %d %s", query, resp.StatusCode, body)
		}
		var out queueResult
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decode queue: %v", err)
		}
		return out
	}

	queue := get("")
	if len(queue.Comments) != 1 || queue.Comments[0].Parent == nil {
		t.Fatalf("expected comment with parent context: %+v", queue.Comments)
	}
	if queue.Comments[0].Parent.ID != a.ID || queue.Comments[0].Parent.QuoteExact != "被引用的原文" || queue.Comments[0].Parent.PageKey != "/doc" || queue.Comments[0].Parent.Body != "批注正文" {
		t.Fatalf("unexpected parent: %+v", queue.Comments[0].Parent)
	}
	if queue.Counts.Annotations.Approved != 1 || queue.Counts.Comments.Pending != 1 {
		t.Fatalf("unexpected counts: %+v", queue.Counts)
	}

	if got := get("&q=" + url.QueryEscape("不存在的词")); len(got.Comments) != 0 || len(got.Annotations) != 0 {
		t.Fatalf("unexpected search result: %+v", got)
	}
	if got := get("&q=" + url.QueryEscape("bob")); len(got.Comments) != 1 {
		t.Fatalf("author search should match: %+v", got.Comments)
	}
	if got := get("&q=" + url.QueryEscape("被引用")); len(got.Comments) != 1 {
		t.Fatalf("parent quote search should match: %+v", got.Comments)
	}
	if got := get("&type=comment"); len(got.Annotations) != 0 || len(got.Comments) != 1 {
		t.Fatalf("comment-only queue: %+v", got)
	}
	if got := get("&type=annotation"); len(got.Annotations) != 0 || len(got.Comments) != 0 {
		t.Fatalf("annotation-only queue: %+v", got)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(site.ID)+"/queue?type=bogus", alice, "", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid queue type = %d", resp.StatusCode)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(site.ID)+"/annotations/"+itoa(a.ID), alice, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("annotation detail: %d %s", resp.StatusCode, body)
	}
	var detail domain.Annotation
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Comments) != 2 || detail.Comments[1].Content != "这是一条待审评论" {
		t.Fatalf("detail should include pending comment: %+v", detail.Comments)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(site.ID)+"/annotations/"+itoa(a.ID), bob, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-moderator detail status = %d", resp.StatusCode)
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

func TestListCountsAndSort(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	a1 := e.createAnnotation(t, alice, site, "http://a.test", "alice one", "one")
	a2 := e.createAnnotation(t, alice, site, "http://a.test", "alice two", "two")
	a3 := e.createAnnotation(t, bob, site, "http://a.test", "bob three", "three")
	resp, body := e.do(t, http.MethodPost, "/api/v1/annotations/"+itoa(a1.ID)+"/comments", bob, site.Key, "http://a.test",
		map[string]any{"content": "评论里的独特关键词"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create comment: %d %s", resp.StatusCode, body)
	}
	resp, _ = e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a1.ID), alice, site.Key, "http://a.test", map[string]any{"status": "resolved"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resolve status = %d", resp.StatusCode)
	}

	type listResult struct {
		Annotations []domain.Annotation `json:"annotations"`
		Total       int                 `json:"total"`
		Counts      struct {
			All      int `json:"all"`
			Mine     int `json:"mine"`
			Resolved int `json:"resolved"`
			Pending  int `json:"pending"`
		} `json:"counts"`
	}
	get := func(query string) listResult {
		t.Helper()
		resp, body := e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc"+query, alice, site.Key, "http://a.test", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list %s: %d %s", query, resp.StatusCode, body)
		}
		var out listResult
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decode %s: %v", query, err)
		}
		return out
	}

	all := get("")
	if all.Total != 3 || all.Counts.All != 3 || all.Counts.Mine != 2 || all.Counts.Resolved != 1 || all.Counts.Pending != 0 {
		t.Fatalf("unexpected counts: %+v", all)
	}

	newest := get("&sort=newest")
	if len(newest.Annotations) != 3 || newest.Annotations[0].ID != a3.ID {
		t.Fatalf("newest order = %+v", newest.Annotations)
	}
	nextPage := get("&sort=newest&cursor=" + itoa(a3.ID))
	if len(nextPage.Annotations) != 2 || nextPage.Annotations[0].ID != a2.ID {
		t.Fatalf("newest cursor order = %+v", nextPage.Annotations)
	}

	byQuote := get("&q=" + url.QueryEscape("three"))
	if len(byQuote.Annotations) != 1 || byQuote.Annotations[0].ID != a3.ID {
		t.Fatalf("search by quote = %+v", byQuote.Annotations)
	}
	byAuthor := get("&q=" + url.QueryEscape("bob"))
	if len(byAuthor.Annotations) != 2 {
		t.Fatalf("search by author = %+v", byAuthor.Annotations)
	}
	byComment := get("&q=" + url.QueryEscape("独特关键词"))
	if len(byComment.Annotations) != 1 || byComment.Annotations[0].ID != a1.ID {
		t.Fatalf("search by comment = %+v", byComment.Annotations)
	}
	if none := get("&q=" + url.QueryEscape("不存在的词")); len(none.Annotations) != 0 {
		t.Fatalf("search should be empty: %+v", none.Annotations)
	}

	resp, _ = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc&sort=bogus", alice, site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid sort = %d", resp.StatusCode)
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

func TestAnnotationSearchRespectsCommentVisibility(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "pre", "http://pre.test", domain.ModerationPre)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")
	_ = e.store.SetUserSuperadmin(context.Background(), 1, true)

	a := e.createAnnotation(t, alice, site, "http://pre.test", "批注正文", "引用原文")
	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/sites/"+itoa(site.ID)+"/annotations/"+itoa(a.ID)+"/moderate",
		alice, "", "", map[string]any{"action": "approve"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve annotation: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodPost, "/api/v1/annotations/"+itoa(a.ID)+"/comments", bob, site.Key, "http://pre.test",
		map[string]any{"content": "待审评论里的隐藏关键词"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create comment: %d %s", resp.StatusCode, body)
	}

	search := func(token string) int {
		t.Helper()
		resp, body := e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc&q="+url.QueryEscape("隐藏关键词"),
			token, site.Key, "http://pre.test", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("search: %d %s", resp.StatusCode, body)
		}
		var out struct {
			Annotations []domain.Annotation `json:"annotations"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decode search: %v", err)
		}
		return len(out.Annotations)
	}

	if n := search(""); n != 0 {
		t.Fatalf("anonymous should not search pending comments, got %d", n)
	}
	if n := search(bob); n != 1 {
		t.Fatalf("comment author should find own pending comment, got %d", n)
	}
	if n := search(alice); n != 1 {
		t.Fatalf("moderator should find pending comment, got %d", n)
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

func (e *testEnv) loginUser(t *testing.T, name string) (string, int64) {
	t.Helper()
	resp, body := e.do(t, http.MethodPost, "/api/v1/auth/dev/login", "", "", "", map[string]any{"name": name})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dev login %s: %d %s", name, resp.StatusCode, body)
	}
	var out struct {
		Token string `json:"token"`
		User  struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	_ = json.Unmarshal(body, &out)
	return out.Token, out.User.ID
}

func TestAdminConsoleAPIs(t *testing.T) {
	e := newTestEnv(t)
	siteA := e.site(t, "a", "http://a.test", domain.ModerationPost)
	siteB := e.site(t, "b", "http://b.test", domain.ModerationPre)
	alice, aliceID := e.loginUser(t, "alice")
	bob, bobID := e.loginUser(t, "bob")
	charlie, _ := e.loginUser(t, "charlie")
	_ = e.store.SetUserSuperadmin(context.Background(), aliceID, true)
	_ = e.store.AddSiteAdmin(context.Background(), domain.SiteAdmin{SiteID: siteB.ID, UserID: bobID, GrantedBy: aliceID})

	// 站点管理员范围
	resp, body := e.do(t, http.MethodGet, "/api/v1/me/sites", alice, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice me/sites: %d %s", resp.StatusCode, body)
	}
	var aliceSites struct {
		Sites []domain.Site `json:"sites"`
	}
	_ = json.Unmarshal(body, &aliceSites)
	if len(aliceSites.Sites) != 2 {
		t.Fatalf("alice sites = %d", len(aliceSites.Sites))
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/me/sites", bob, "", "", nil)
	var bobSites struct {
		Sites []domain.Site `json:"sites"`
	}
	_ = json.Unmarshal(body, &bobSites)
	if len(bobSites.Sites) != 1 || bobSites.Sites[0].ID != siteB.ID {
		t.Fatalf("bob sites = %+v", bobSites.Sites)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/me/sites", charlie, "", "", nil)
	_ = json.Unmarshal(body, &bobSites)
	if len(bobSites.Sites) != 0 {
		t.Fatalf("charlie sites = %d", len(bobSites.Sites))
	}

	// 预置待审内容
	e.createAnnotation(t, charlie, siteB, "http://b.test", "pending", "p")

	// 概览：超管与站点管理员范围不同
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", alice, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice overview: %d %s", resp.StatusCode, body)
	}
	var aliceOverview struct {
		Counts struct {
			Sites              int `json:"sites"`
			PendingAnnotations int `json:"pendingAnnotations"`
		} `json:"counts"`
	}
	_ = json.Unmarshal(body, &aliceOverview)
	if aliceOverview.Counts.Sites != 2 || aliceOverview.Counts.PendingAnnotations < 1 {
		t.Fatalf("alice overview = %+v", aliceOverview.Counts)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", bob, "", "", nil)
	var bobOverview struct {
		Counts struct {
			Sites int `json:"sites"`
		} `json:"counts"`
	}
	_ = json.Unmarshal(body, &bobOverview)
	if bobOverview.Counts.Sites != 1 {
		t.Fatalf("bob overview = %+v", bobOverview.Counts)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", charlie, "", "", nil)
	var charlieOverview struct {
		Counts struct {
			Sites int `json:"sites"`
		} `json:"counts"`
	}
	_ = json.Unmarshal(body, &charlieOverview)
	if charlieOverview.Counts.Sites != 0 {
		t.Fatalf("charlie overview = %+v", charlieOverview.Counts)
	}

	// 管理端登录方式：包含启用 OIDC 提供商
	if _, err := e.store.CreateProvider(context.Background(), domain.OIDCProvider{
		SiteID: siteA.ID, Name: "SSO", Issuer: "https://sso.test", ClientID: "cid",
		ClientSecretCipher: []byte("cipher"), Scopes: []string{"openid"}, Enabled: true,
	}); err != nil {
		t.Fatalf("create provider: %v", err)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/auth/providers", "", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin providers: %d %s", resp.StatusCode, body)
	}
	var providers struct {
		GitHub    bool `json:"github"`
		Providers []struct {
			SiteName string `json:"siteName"`
			Issuer   string `json:"issuer"`
		} `json:"providers"`
	}
	_ = json.Unmarshal(body, &providers)
	if providers.GitHub || len(providers.Providers) != 1 || providers.Providers[0].SiteName != "a" {
		t.Fatalf("providers = %+v", providers)
	}
	if bytes.Contains(body, []byte("cipher")) {
		t.Fatalf("secret leaked: %s", body)
	}

	// 站点封禁列表
	if err := e.store.BanSite(context.Background(), domain.SiteBan{SiteID: siteB.ID, UserID: 3, BannedBy: aliceID, Reason: "spam"}); err != nil {
		t.Fatalf("ban: %v", err)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(siteB.ID)+"/bans", bob, "", "", nil)
	if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("spam")) {
		t.Fatalf("bans: %d %s", resp.StatusCode, body)
	}
	resp, _ = e.do(t, http.MethodGet, "/api/v1/admin/sites/"+itoa(siteA.ID)+"/bans", bob, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("bob cross-site bans = %d", resp.StatusCode)
	}

	// 静态入口（不跟随重定向）
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, _ := http.NewRequest(http.MethodGet, e.ts.URL+"/admin", nil)
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatalf("/admin: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/admin/" {
		t.Fatalf("/admin status = %d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp, adminBody := e.do(t, http.MethodGet, "/admin/", "", "", "", nil)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("/admin/ status = %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusOK && !bytes.Contains(adminBody, []byte(`id="app"`)) {
		t.Fatalf("/admin/ did not return app html: %s", adminBody[:min(200, len(adminBody))])
	}
}

func (e *testEnv) listPage(t *testing.T, token, siteKey, origin, filter string, cursor, limit int64) ([]domain.Annotation, int64, int) {
	t.Helper()
	path := "/api/v1/annotations?page=%2Fdoc"
	if filter != "" {
		path += "&filter=" + filter
	}
	if cursor > 0 {
		path += "&cursor=" + itoa(cursor)
	}
	if limit > 0 {
		path += "&limit=" + itoa(limit)
	}
	resp, body := e.do(t, http.MethodGet, path, token, siteKey, origin, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list page: %d %s", resp.StatusCode, body)
	}
	var out struct {
		Annotations []domain.Annotation `json:"annotations"`
		NextCursor  int64               `json:"nextCursor"`
		Total       int                 `json:"total"`
	}
	_ = json.Unmarshal(body, &out)
	return out.Annotations, out.NextCursor, out.Total
}

func TestPageTotal(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	alice := e.login(t, "alice")
	bob := e.login(t, "bob")

	a1 := e.createAnnotation(t, alice, site, "http://a.test", "alice open", "alpha")
	a2 := e.createAnnotation(t, alice, site, "http://a.test", "alice resolved", "beta")
	e.createAnnotation(t, bob, site, "http://a.test", "bob open", "gamma")
	resp, _ := e.do(t, http.MethodPatch, "/api/v1/annotations/"+itoa(a2.ID), alice, site.Key, "http://a.test", map[string]any{"status": "resolved"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resolve = %d", resp.StatusCode)
	}

	if _, _, total := e.listPage(t, "", site.Key, "http://a.test", "all", 0, 0); total != 3 {
		t.Fatalf("anon all total = %d", total)
	}
	if _, _, total := e.listPage(t, alice, site.Key, "http://a.test", "mine", 0, 0); total != 3 {
		t.Fatalf("alice mine total = %d", total)
	}
	if _, _, total := e.listPage(t, "", site.Key, "http://a.test", "resolved", 0, 0); total != 3 {
		t.Fatalf("resolved total = %d", total)
	}
	if _, next, total := e.listPage(t, "", site.Key, "http://a.test", "all", 0, 1); total != 3 {
		t.Fatalf("paginated total = %d", total)
	} else if next == 0 {
		t.Fatal("expected next cursor")
	}

	// 先审后发站点：匿名 total=0，作者 total=1
	pre := e.site(t, "pre", "http://pre.test", domain.ModerationPre)
	e.createAnnotation(t, alice, pre, "http://pre.test", "pending", "p")
	if _, _, total := e.listPage(t, "", pre.Key, "http://pre.test", "all", 0, 0); total != 0 {
		t.Fatalf("pre anon total = %d", total)
	}
	if _, _, total := e.listPage(t, alice, pre.Key, "http://pre.test", "pending", 0, 0); total != 1 {
		t.Fatalf("pre author total = %d", total)
	}

	// 删除后 total 减少
	resp, _ = e.do(t, http.MethodDelete, "/api/v1/annotations/"+itoa(a1.ID), alice, site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", resp.StatusCode)
	}
	if _, _, total := e.listPage(t, "", site.Key, "http://a.test", "all", 0, 0); total != 2 {
		t.Fatalf("after delete total = %d", total)
	}
}

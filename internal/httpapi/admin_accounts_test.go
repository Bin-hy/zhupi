package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/service"
)

func (e *testEnv) bootstrapRoot(t *testing.T) (string, int64) {
	t.Helper()
	if _, err := e.auth.BootstrapLocalAdmin(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "admin",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root login: %d %s", resp.StatusCode, body)
	}
	var login struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &login)
	resp, body = e.do(t, http.MethodPost, "/api/v1/me/password", login.Token, "", "", map[string]any{
		"currentPassword": "admin", "newPassword": "root-pass-123",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("root change password: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/me", login.Token, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root me: %d %s", resp.StatusCode, body)
	}
	var me struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(body, &me)
	return login.Token, me.ID
}

func TestCreateLocalAdminAndFirstLogin(t *testing.T) {
	e := newTestEnv(t)
	if _, err := e.auth.BootstrapLocalAdmin(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "admin",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root login: %d %s", resp.StatusCode, body)
	}
	var rootLogin struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &rootLogin)
	if rootLogin.Token == "" {
		t.Fatalf("root token missing: %s", body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/me/password", rootLogin.Token, "", "", map[string]any{
		"currentPassword": "admin", "newPassword": "root-pass-123",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("root change password: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins", rootLogin.Token, "", "", map[string]any{
		"username": "ops", "displayName": "Ops", "isSuperadmin": false, "siteIds": []int64{},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create admin: %d %s", resp.StatusCode, body)
	}
	var created struct {
		Account struct {
			UserID   int64  `json:"userId"`
			Username string `json:"username"`
		} `json:"account"`
		GeneratedPassword string `json:"generatedPassword"`
	}
	_ = json.Unmarshal(body, &created)
	if created.Account.UserID == 0 || created.Account.Username != "ops" || created.GeneratedPassword == "" {
		t.Fatalf("unexpected create result: %s", body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/local-admins", rootLogin.Token, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list admins: %d %s", resp.StatusCode, body)
	}
	var list struct {
		Admins []struct {
			Username string `json:"username"`
		} `json:"admins"`
	}
	_ = json.Unmarshal(body, &list)
	if len(list.Admins) != 2 {
		t.Fatalf("admins = %+v", list.Admins)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "ops", "password": created.GeneratedPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ops login: %d %s", resp.StatusCode, body)
	}
	var opsLogin struct {
		Token              string `json:"token"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	_ = json.Unmarshal(body, &opsLogin)
	if opsLogin.Token == "" || !opsLogin.MustChangePassword {
		t.Fatalf("unexpected ops login: %s", body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", opsLogin.Token, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("must-change overview: %d %s", resp.StatusCode, body)
	}
	var blocked struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &blocked)
	if blocked.Error.Code != "password_change_required" {
		t.Fatalf("unexpected error code: %s", body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/me/password", opsLogin.Token, "", "", map[string]any{
		"currentPassword": created.GeneratedPassword, "newPassword": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ops change password: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", opsLogin.Token, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("overview after change: %d %s", resp.StatusCode, body)
	}
}

func assertErrorCode(t *testing.T, body []byte, want string) {
	t.Helper()
	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &out)
	if out.Error.Code != want {
		t.Fatalf("error code = %q, want %q: %s", out.Error.Code, want, body)
	}
}

func TestRedirectBindError(t *testing.T) {
	server := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/1/callback", nil)
	err := &service.BindError{
		Flow: domain.AuthFlow{
			Mode:     domain.FlowBind,
			ReturnTo: "http://127.0.0.1:8080/admin/account",
		},
		Err: domain.Conflict("identity_already_bound", "该登录身份已绑定到其他账号"),
	}
	if !server.redirectBindError(rec, req, err) {
		t.Fatal("bind error should redirect")
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "http://127.0.0.1:8080/admin/account#") ||
		!strings.Contains(loc, "annot_bind=error") {
		t.Fatalf("unexpected location: %s", loc)
	}
	if !strings.Contains(loc, "identity") && !strings.Contains(loc, "%E8%AF%A5") {
		t.Fatalf("message missing from location: %s", loc)
	}

	rec = httptest.NewRecorder()
	if server.redirectBindError(rec, req, domain.Invalid("x", "login failed")) {
		t.Fatal("non-bind error should not redirect")
	}
}

func TestBannedUserCannotLocalLogin(t *testing.T) {
	e := newTestEnv(t)
	root, _ := e.bootstrapRoot(t)

	visitor := e.login(t, "banned")
	resp, body := e.do(t, http.MethodGet, "/api/v1/me", visitor, "", "", nil)
	var me struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(body, &me)

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins/open", root, "", "", map[string]any{
		"userId": me.ID, "username": "banned", "initialPassword": "banned-pass-1",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("open credential: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/users/"+itoa(me.ID)+"/ban", root, "", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ban: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "banned", "password": "banned-pass-1",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("banned login: %d %s", resp.StatusCode, body)
	}
}

func TestLegacySessionCannotAccessAdmin(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	user, err := e.store.CreateUser(context.Background(), domain.User{
		DisplayName: "legacy", Status: domain.UserActive, IsSuperadmin: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	raw, hash, err := authpkg.NewToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if err := e.store.CreateSession(context.Background(), domain.Session{
		TokenHash: hash, UserID: user.ID, AuthMethod: domain.AuthMethodLegacy,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	resp, body := e.do(t, http.MethodGet, "/api/v1/admin/overview", raw, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("legacy admin: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "admin_local_auth_required")

	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", raw, site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("legacy site api: %d %s", resp.StatusCode, body)
	}
}

func TestLocalAdminValidationAndOpen(t *testing.T) {
	e := newTestEnv(t)
	root, _ := e.bootstrapRoot(t)

	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "ab", "initialPassword": "valid-pass-1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid username: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "invalid_username")

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "weakuser", "initialPassword": "short",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("weak password: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "weak_password")

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "ops", "initialPassword": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create ops: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "OPS", "initialPassword": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate username: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "username_taken")

	visitor := e.login(t, "visitor")
	resp, body = e.do(t, http.MethodGet, "/api/v1/me", visitor, "", "", nil)
	var me struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(body, &me)

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins/open", root, "", "", map[string]any{
		"userId": me.ID, "username": "visitor", "initialPassword": "visitor-pass-1",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("open credential: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins/open", root, "", "", map[string]any{
		"userId": me.ID, "username": "visitor2", "initialPassword": "visitor-pass-2",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("open twice: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "local_credential_exists")

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "visitor", "password": "visitor-pass-1",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("visitor local login: %d %s", resp.StatusCode, body)
	}
}

func TestResetLocalAdminPassword(t *testing.T) {
	e := newTestEnv(t)
	root, rootID := e.bootstrapRoot(t)

	resp, body := e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "ops", "initialPassword": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create ops: %d %s", resp.StatusCode, body)
	}
	var created struct {
		Account struct {
			UserID int64 `json:"userId"`
		} `json:"account"`
	}
	_ = json.Unmarshal(body, &created)

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "ops", "password": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ops login: %d %s", resp.StatusCode, body)
	}
	var opsLogin struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &opsLogin)

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins/"+itoa(rootID)+"/password/reset", root, "", "", map[string]any{
		"password": "whatever-1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("self reset: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "self_password_reset")

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins/"+itoa(created.Account.UserID)+"/password/reset", root, "", "", map[string]any{
		"password": "new-ops-pass",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reset ops: %d %s", resp.StatusCode, body)
	}

	resp, _ = e.do(t, http.MethodGet, "/api/v1/me", opsLogin.Token, "", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ops session should be revoked: %d", resp.StatusCode)
	}

	resp, _ = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "ops", "password": "ops-pass-123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password should fail: %d", resp.StatusCode)
	}
	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "ops", "password": "new-ops-pass",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("new password login: %d %s", resp.StatusCode, body)
	}
	var relogin struct {
		MustChangePassword bool `json:"mustChangePassword"`
	}
	_ = json.Unmarshal(body, &relogin)
	if !relogin.MustChangePassword {
		t.Fatal("reset password should require change")
	}
}

func TestLastSuperadminGuard(t *testing.T) {
	e := newTestEnv(t)
	root, rootID := e.bootstrapRoot(t)

	resp, body := e.do(t, http.MethodPut, "/api/v1/admin/users/"+itoa(rootID)+"/superadmin", root, "", "", map[string]any{
		"isSuperadmin": false,
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("demote last superadmin: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "last_superadmin")

	visitor := e.login(t, "visitor")
	resp, body = e.do(t, http.MethodGet, "/api/v1/me", visitor, "", "", nil)
	var me struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(body, &me)
	resp, body = e.do(t, http.MethodPut, "/api/v1/admin/users/"+itoa(me.ID)+"/superadmin", root, "", "", map[string]any{
		"isSuperadmin": true,
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("grant without credential: %d %s", resp.StatusCode, body)
	}
	assertErrorCode(t, body, "local_credential_required")

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/local-admins", root, "", "", map[string]any{
		"username": "admin2", "initialPassword": "admin2-pass-1", "isSuperadmin": true,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create second superadmin: %d %s", resp.StatusCode, body)
	}
	var second struct {
		Account struct {
			UserID int64 `json:"userId"`
		} `json:"account"`
	}
	_ = json.Unmarshal(body, &second)

	resp, body = e.do(t, http.MethodPut, "/api/v1/admin/users/"+itoa(second.Account.UserID)+"/superadmin", root, "", "", map[string]any{
		"isSuperadmin": false,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("demote other superadmin: %d %s", resp.StatusCode, body)
	}
}

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func TestLocalAdminLoginAndForcedPasswordChange(t *testing.T) {
	e := newTestEnv(t)
	if _, err := e.auth.BootstrapLocalAdmin(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	resp, body := e.do(t, http.MethodGet, "/api/v1/admin/auth/providers", "", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("providers: %d %s", resp.StatusCode, body)
	}
	var providers struct {
		Local bool `json:"local"`
	}
	_ = json.Unmarshal(body, &providers)
	if !providers.Local {
		t.Fatalf("local provider should be enabled: %s", body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "wrong",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "admin",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d %s", resp.StatusCode, body)
	}
	var login struct {
		Token              string `json:"token"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	_ = json.Unmarshal(body, &login)
	if login.Token == "" || !login.MustChangePassword {
		t.Fatalf("unexpected login response: %s", body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/me", login.Token, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me: %d %s", resp.StatusCode, body)
	}
	var me struct {
		Username           string `json:"username"`
		MustChangePassword bool   `json:"mustChangePassword"`
	}
	_ = json.Unmarshal(body, &me)
	if me.Username != "admin" || !me.MustChangePassword {
		t.Fatalf("unexpected me: %s", body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/sites", login.Token, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin API should be blocked: %d %s", resp.StatusCode, body)
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

	resp, body = e.do(t, http.MethodPost, "/api/v1/me/password", login.Token, "", "", map[string]any{
		"currentPassword": "wrong", "newPassword": "new-password-1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong current password: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/me/password", login.Token, "", "", map[string]any{
		"currentPassword": "admin", "newPassword": "new-password-1",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("change password: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/sites", login.Token, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin API after change: %d %s", resp.StatusCode, body)
	}

	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "admin",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password should fail: %d %s", resp.StatusCode, body)
	}
	resp, body = e.do(t, http.MethodPost, "/api/v1/admin/auth/local/login", "", "", "", map[string]any{
		"username": "admin", "password": "new-password-1",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("new password login: %d %s", resp.StatusCode, body)
	}
	_ = json.Unmarshal(body, &login)
	if login.MustChangePassword {
		t.Fatalf("mustChangePassword should be false: %s", body)
	}
}

func TestAdminAPIsRequireLocalSession(t *testing.T) {
	e := newTestEnv(t)
	site := e.site(t, "a", "http://a.test", domain.ModerationPost)
	visitor := e.login(t, "visitor")

	for _, path := range []string{"/api/v1/admin/auth/github/start", "/api/v1/admin/auth/oidc/1/start"} {
		resp, _ := e.do(t, http.MethodGet, path, "", "", "", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("removed admin login %s status = %d", path, resp.StatusCode)
		}
	}

	resp, body := e.do(t, http.MethodGet, "/api/v1/admin/overview", visitor, "", "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("visitor admin overview: %d %s", resp.StatusCode, body)
	}
	var blocked struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &blocked)
	if blocked.Error.Code != "admin_local_auth_required" {
		t.Fatalf("unexpected error code: %s", body)
	}

	resp, body = e.do(t, http.MethodGet, "/api/v1/annotations?page=%2Fdoc", visitor, site.Key, "http://a.test", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("visitor site api: %d %s", resp.StatusCode, body)
	}

	local := e.localize(t, visitor)
	resp, body = e.do(t, http.MethodGet, "/api/v1/admin/overview", local, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("local admin overview: %d %s", resp.StatusCode, body)
	}
}

package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

func newLocalAuthEnv(t *testing.T, username, password string) (*AuthService, *store.Store, *config.Config) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cfg := &config.Config{
		SessionTTL:          time.Hour,
		LocalAdminUsername:  username,
		LocalAdminPassword:  password,
		ReadRatePerMin:      1000,
		WriteRatePerMin:     1000,
		WriteUserRatePerMin: 1000,
	}
	svc := NewAuthService(cfg, st, st, st, st, st, st, nil, nil, nil)
	return svc, st, cfg
}

func TestBootstrapLocalAdminAndLogin(t *testing.T) {
	ctx := context.Background()
	svc, st, _ := newLocalAuthEnv(t, "admin", "admin")

	boot, err := svc.BootstrapLocalAdmin(ctx)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if !boot.Created || !boot.MustChangePassword {
		t.Fatalf("unexpected bootstrap result: %+v", boot)
	}

	cred, err := st.LocalCredentialByUsername(ctx, "ADMIN")
	if err != nil {
		t.Fatalf("credential lookup (case-insensitive): %v", err)
	}
	if cred.Username != "admin" || !cred.MustChangePassword {
		t.Fatalf("unexpected credential: %+v", cred)
	}
	user, err := st.UserByID(ctx, cred.UserID)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if !user.IsSuperadmin {
		t.Fatal("local admin should be superadmin")
	}

	if _, err := svc.LocalLogin(ctx, "admin", "wrong", ClientMeta{}); err == nil {
		t.Fatal("wrong password should fail")
	} else if appErr, ok := domain.AsAppError(err); !ok || appErr.Status != 401 {
		t.Fatalf("wrong password error = %v", err)
	}
	if _, err := svc.LocalLogin(ctx, "nobody", "admin", ClientMeta{}); err == nil {
		t.Fatal("unknown user should fail")
	}

	res, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{UserAgent: "test", IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token == "" || !res.MustChangePassword {
		t.Fatalf("unexpected login result: %+v", res)
	}
	p, _, err := svc.AuthenticateSession(ctx, res.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.LocalUsername != "admin" || !p.MustChangePassword || !p.IsSuperadmin {
		t.Fatalf("unexpected principal: %+v", p)
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newLocalAuthEnv(t, "admin", "admin")
	if _, err := svc.BootstrapLocalAdmin(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	first, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{})
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{})
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	_, firstSess, err := svc.AuthenticateSession(ctx, first.Token)
	if err != nil {
		t.Fatalf("first session: %v", err)
	}

	if err := svc.ChangePassword(ctx, first.User.ID, firstSess.ID, "wrong", "new-password-1"); err == nil {
		t.Fatal("wrong current password should fail")
	}
	if err := svc.ChangePassword(ctx, first.User.ID, firstSess.ID, "admin", "short"); err == nil {
		t.Fatal("short password should fail")
	}
	if err := svc.ChangePassword(ctx, first.User.ID, firstSess.ID, "admin", "admin"); err == nil {
		t.Fatal("unchanged password should fail")
	}
	if err := svc.ChangePassword(ctx, first.User.ID, firstSess.ID, "admin", "new-password-1"); err != nil {
		t.Fatalf("change password: %v", err)
	}

	if _, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{}); err == nil {
		t.Fatal("old password should fail after change")
	}
	if _, err := svc.LocalLogin(ctx, "admin", "new-password-1", ClientMeta{}); err != nil {
		t.Fatalf("new password login: %v", err)
	}
	if _, _, err := svc.AuthenticateSession(ctx, first.Token); err != nil {
		t.Fatalf("current session should remain valid: %v", err)
	}
	if _, _, err := svc.AuthenticateSession(ctx, second.Token); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("other session should be revoked, got %v", err)
	}
}

func TestBootstrapDoesNotOverwriteUnlessReset(t *testing.T) {
	ctx := context.Background()
	svc, st, cfg := newLocalAuthEnv(t, "admin", "admin")
	if _, err := svc.BootstrapLocalAdmin(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	login, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_, sess, err := svc.AuthenticateSession(ctx, login.Token)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if err := svc.ChangePassword(ctx, login.User.ID, sess.ID, "admin", "changed-pass-1"); err != nil {
		t.Fatalf("change: %v", err)
	}

	boot, err := svc.BootstrapLocalAdmin(ctx)
	if err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if !boot.Skipped {
		t.Fatalf("expected skipped, got %+v", boot)
	}
	if _, err := svc.LocalLogin(ctx, "admin", "changed-pass-1", ClientMeta{}); err != nil {
		t.Fatalf("changed password should survive restart: %v", err)
	}

	cfg.LocalAdminResetPassword = true
	boot, err = svc.BootstrapLocalAdmin(ctx)
	if err != nil {
		t.Fatalf("reset bootstrap: %v", err)
	}
	if !boot.Reset {
		t.Fatalf("expected reset, got %+v", boot)
	}
	if _, err := svc.LocalLogin(ctx, "admin", "admin", ClientMeta{}); err != nil {
		t.Fatalf("reset password should work: %v", err)
	}
	cred, err := st.LocalCredentialByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("credential: %v", err)
	}
	if !cred.MustChangePassword {
		t.Fatal("reset to default password should require change")
	}
}

func TestResetLocalPassword(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newLocalAuthEnv(t, "admin", "admin")
	if _, err := svc.BootstrapLocalAdmin(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if _, err := svc.ResetLocalPassword(ctx, "missing", "whatever-123"); err == nil {
		t.Fatal("missing user should fail")
	}
	if _, err := svc.ResetLocalPassword(ctx, "admin", "short"); err == nil {
		t.Fatal("short password should fail")
	}
	if _, err := svc.ResetLocalPassword(ctx, "admin", "cli-reset-pass"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := svc.LocalLogin(ctx, "admin", "cli-reset-pass", ClientMeta{}); err != nil {
		t.Fatalf("login with reset password: %v", err)
	}
}

package service

import (
	"context"
	"path/filepath"
	"testing"

	authpkg "github.com/Bin-hy/zhupi/internal/auth"
	"github.com/Bin-hy/zhupi/internal/config"
	"github.com/Bin-hy/zhupi/internal/domain"
	"github.com/Bin-hy/zhupi/internal/store"
)

func newTestAuthService(t *testing.T) (*AuthService, *store.Store, *config.Config) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cfg := &config.Config{PublicBaseURL: "http://127.0.0.1:8080"}
	svc := NewAuthService(cfg, st, st, st, st, st, st, nil,
		authpkg.NewGitHubClient(authpkg.GitHubConfig{}, nil), authpkg.NewOIDCClient(nil))
	return svc, st, cfg
}

func TestUnbindIdentityAllowsLocalCredential(t *testing.T) {
	svc, st, _ := newTestAuthService(t)
	ctx := context.Background()
	user, err := st.CreateUser(ctx, domain.User{DisplayName: "alice", Status: domain.UserActive})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	identity, err := st.CreateIdentity(ctx, domain.Identity{
		UserID: user.ID, Provider: domain.ProviderGitHub, SiteID: 0,
		Issuer: authpkg.GitHubIssuer, Subject: "s1", Handle: "alice",
	})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}

	if err := svc.UnbindIdentity(ctx, user.ID, identity.ID); err == nil {
		t.Fatal("visitor should not unbind last identity")
	}

	if err := st.CreateLocalCredential(ctx, domain.LocalCredential{
		UserID: user.ID, Username: "alice", PasswordHash: "hash",
	}); err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if err := svc.UnbindIdentity(ctx, user.ID, identity.ID); err != nil {
		t.Fatalf("local admin should unbind last identity: %v", err)
	}
	list, err := st.ListIdentitiesByUser(ctx, user.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("identities = %+v, err = %v", list, err)
	}
}

func TestBindIdentityConflict(t *testing.T) {
	svc, st, _ := newTestAuthService(t)
	ctx := context.Background()
	alice, _ := st.CreateUser(ctx, domain.User{DisplayName: "alice", Status: domain.UserActive})
	bob, _ := st.CreateUser(ctx, domain.User{DisplayName: "bob", Status: domain.UserActive})
	flow := domain.AuthFlow{
		Mode: domain.FlowBind, ProviderType: domain.ProviderGitHub,
		Issuer: authpkg.GitHubIssuer, SiteID: 0, UserID: alice.ID,
	}
	ext := authpkg.ExternalUser{Subject: "s1", Handle: "alice", Name: "alice"}
	if err := svc.bindIdentity(ctx, flow, ext); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	if err := svc.bindIdentity(ctx, flow, ext); err != nil {
		t.Fatalf("idempotent bind: %v", err)
	}
	flow.UserID = bob.ID
	err := svc.bindIdentity(ctx, flow, ext)
	if err == nil {
		t.Fatal("binding identity owned by another user should fail")
	}
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != "identity_already_bound" {
		t.Fatalf("err = %v", err)
	}
}

func TestAdminBindableIdentitiesAndGitHubStart(t *testing.T) {
	svc, st, cfg := newTestAuthService(t)
	ctx := context.Background()
	user, err := st.CreateUser(ctx, domain.User{DisplayName: "alice", Status: domain.UserActive})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if _, err := svc.StartAdminBindGitHub(ctx, user.ID); err == nil {
		t.Fatal("unconfigured github should fail")
	}

	cfg.GitHubClientID = "id"
	cfg.GitHubSecret = "secret"
	svc.github = authpkg.NewGitHubClient(authpkg.GitHubConfig{ClientID: "id", ClientSecret: "secret"}, nil)
	target, err := svc.StartAdminBindGitHub(ctx, user.ID)
	if err != nil || target == "" {
		t.Fatalf("start bind github: %q, %v", target, err)
	}

	out, err := svc.AdminBindableIdentities(ctx, user.ID)
	if err != nil {
		t.Fatalf("bindable identities: %v", err)
	}
	if !out.Bindable.GitHub {
		t.Fatal("github should be bindable")
	}
}

package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Bin-hy/zhupi/internal/domain"
)

func TestMigrateIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table'`).Scan(&n); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	// 11 张业务表 + schema_migrations
	if n < 12 {
		t.Fatalf("table count = %d", n)
	}
}

func TestMigrateRejectsOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if _, err := s.db.Exec(`CREATE TABLE annotations (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create old table: %v", err)
	}
	if err := s.Migrate(context.Background()); err != ErrOldSchema {
		t.Fatalf("err = %v, want ErrOldSchema", err)
	}
}

func TestCreateLocalAdminUsernameConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cred := domain.LocalCredential{Username: "admin", PasswordHash: "hash", MustChangePassword: true}
	u, err := s.CreateLocalAdmin(ctx, domain.User{DisplayName: "admin"}, cred, nil, 0)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if u.ID == 0 || !u.Status.Valid() {
		t.Fatalf("unexpected user: %+v", u)
	}

	_, err = s.CreateLocalAdmin(ctx, domain.User{DisplayName: "other"}, cred, nil, 0)
	if err == nil {
		t.Fatal("duplicate username should fail")
	}
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != "username_taken" {
		t.Fatalf("err = %v", err)
	}

	var users int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 1 {
		t.Fatalf("orphan user created, users = %d", users)
	}

	admins, err := s.ListLocalAdmins(ctx)
	if err != nil || len(admins) != 1 || admins[0].Username != "admin" {
		t.Fatalf("list admins = %+v, err = %v", admins, err)
	}
}

func TestCountActiveSuperadminCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cred := domain.LocalCredential{Username: "root", PasswordHash: "hash"}
	if _, err := s.CreateLocalAdmin(ctx, domain.User{DisplayName: "root", IsSuperadmin: true}, cred, nil, 0); err != nil {
		t.Fatalf("create superadmin: %v", err)
	}
	if n, err := s.CountActiveSuperadminCredentials(ctx); err != nil || n != 1 {
		t.Fatalf("count = %d, err = %v", n, err)
	}
	if err := s.UpdateUserStatus(ctx, 1, domain.UserBanned); err != nil {
		t.Fatalf("ban: %v", err)
	}
	if n, err := s.CountActiveSuperadminCredentials(ctx); err != nil || n != 0 {
		t.Fatalf("count after ban = %d, err = %v", n, err)
	}
}

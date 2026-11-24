package store

import (
	"context"
	"path/filepath"
	"testing"
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

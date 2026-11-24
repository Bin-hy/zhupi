package auth

import "testing"

func TestNewToken(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(raw) != 64 || len(hash) != 64 {
		t.Fatalf("unexpected lengths raw=%d hash=%d", len(raw), len(hash))
	}
	if HashToken(raw) != hash {
		t.Fatal("hash mismatch")
	}
	raw2, _, _ := NewToken()
	if raw2 == raw {
		t.Fatal("tokens should differ")
	}
}

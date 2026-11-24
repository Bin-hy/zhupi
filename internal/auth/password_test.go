package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt hash, got %q", hash)
	}
	if !VerifyPassword(hash, "s3cret-pass") {
		t.Fatal("correct password should verify")
	}
	if VerifyPassword(hash, "wrong-pass") {
		t.Fatal("wrong password should not verify")
	}
	if VerifyPassword("", "s3cret-pass") {
		t.Fatal("empty hash should not verify")
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name  string
		input string
		ok    bool
	}{
		{"too short", "short", false},
		{"min", "12345678", true},
		{"too long", strings.Repeat("a", 73), false},
		{"unicode ok", "密码密码密码密码", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.input)
			if tc.ok && err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	pwd, err := GeneratePassword()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(pwd) < 12 {
		t.Fatalf("password too short: %q", pwd)
	}
	other, err := GeneratePassword()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if pwd == other {
		t.Fatal("generated passwords should differ")
	}
}

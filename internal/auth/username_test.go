package auth

import "testing"

func TestValidateUsername(t *testing.T) {
	valid := []string{"abc", "Admin.User_1", "a-b-c", "user1234567890123456789012345678"}
	for _, name := range valid {
		if err := ValidateUsername(name); err != nil {
			t.Fatalf("ValidateUsername(%q) = %v", name, err)
		}
	}

	invalid := []string{"", "ab", "a", "user name", "user@example.com", "中文用户", "user/name",
		"user1234567890123456789012345678901"}
	for _, name := range invalid {
		if err := ValidateUsername(name); err == nil {
			t.Fatalf("ValidateUsername(%q) should fail", name)
		}
	}
}

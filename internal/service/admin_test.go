package service

import "testing"

func TestSafeAdminReturnTo(t *testing.T) {
	base := "https://annot.test"
	cases := []struct{ in, want string }{
		{"", "https://annot.test/admin/"},
		{"https://annot.test/admin/", "https://annot.test/admin/"},
		{"https://annot.test/admin/sites/1?tab=providers", "https://annot.test/admin/sites/1?tab=providers"},
		{"https://evil.test/admin/", "https://annot.test/admin/"},
		{"https://annot.test/sites/1", "https://annot.test/admin/"},
		{"javascript:alert(1)", "https://annot.test/admin/"},
	}
	for _, c := range cases {
		if got := safeAdminReturnTo(base, c.in); got != c.want {
			t.Errorf("safeAdminReturnTo(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

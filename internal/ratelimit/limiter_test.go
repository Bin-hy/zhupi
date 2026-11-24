package ratelimit

import "testing"

func TestLimiterBurst(t *testing.T) {
	l := New(60) // 60/min => burst 60
	for i := 0; i < 60; i++ {
		if !l.Allow("k") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if l.Allow("k") {
		t.Fatal("超过突发额度应被拒绝")
	}
	if !l.Allow("other") {
		t.Fatal("不同 key 互不影响")
	}
}

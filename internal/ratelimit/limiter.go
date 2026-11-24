package ratelimit

import (
	"sync"
	"time"
)

// Limiter 是进程内按 key 的令牌桶。单实例部署为前提。
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // 每秒补充的令牌数
	burst   float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

func New(perMinute int) *Limiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return &Limiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(perMinute),
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) > 50000 {
			l.buckets = make(map[string]*bucket)
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

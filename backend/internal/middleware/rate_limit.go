package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/pkg/api"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	refill   float64
	lastGC   time.Time
}

func NewRateLimiter(capacity int, refillPerSecond float64) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		buckets: make(map[string]*bucket), capacity: float64(capacity),
		refill: refillPerSecond, lastGC: now,
	}
}

func (l *RateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := scope + ":" + c.ClientIP()
		if !l.allow(key, time.Now()) {
			c.Header("Retry-After", "5")
			api.Fail(c, api.NewError(429, "RATE_LIMITED", "请求过于频繁，请稍后重试"))
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, exists := l.buckets[key]
	if !exists {
		entry = &bucket{tokens: l.capacity, lastRefill: now, lastSeen: now}
		l.buckets[key] = entry
	}
	elapsed := now.Sub(entry.lastRefill).Seconds()
	entry.tokens += elapsed * l.refill
	if entry.tokens > l.capacity {
		entry.tokens = l.capacity
	}
	entry.lastRefill = now
	entry.lastSeen = now
	allowed := entry.tokens >= 1
	if allowed {
		entry.tokens--
	}
	if now.Sub(l.lastGC) > 10*time.Minute {
		for bucketKey, candidate := range l.buckets {
			if now.Sub(candidate.lastSeen) > 30*time.Minute {
				delete(l.buckets, bucketKey)
			}
		}
		l.lastGC = now
	}
	return allowed
}

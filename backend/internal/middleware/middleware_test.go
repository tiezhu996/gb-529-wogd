package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterRefillsDeterministically(t *testing.T) {
	limiter := NewRateLimiter(2, 1)
	now := time.Unix(100, 0)
	if !limiter.allow("test", now) || !limiter.allow("test", now) {
		t.Fatal("initial bucket should allow two requests")
	}
	if limiter.allow("test", now) {
		t.Fatal("empty bucket should reject request")
	}
	if !limiter.allow("test", now.Add(time.Second)) {
		t.Fatal("one token should refill after one second")
	}
}

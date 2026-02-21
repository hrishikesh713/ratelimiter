package token

import (
	"testing"
	"testing/synctest"
	"time"

	rle "github.com/hrishikesh713/ratelimiter/internal/rlerrors"
)

func TestTokenBucket_BasicRateLimiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Create a token bucket rate limiter
		// Rate: 10 tokens per second
		// Burst: 10 tokens
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(10.0, 10.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		allowedCount := 0
		deniedCount := 0

		// Make 15 requests rapidly (should allow 10, deny 5)
		for i := 1; i <= 15; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				if err != rle.ErrRateLimitExceeded {
					t.Errorf("Request %d: unexpected error: %v", i, err)
				}
				deniedCount++
			} else if allowed {
				allowedCount++
			} else {
				deniedCount++
			}
			time.Sleep(10 * time.Millisecond)
		}

		if allowedCount != 10 {
			t.Errorf("Expected 10 allowed requests, got %d", allowedCount)
		}
		if deniedCount != 5 {
			t.Errorf("Expected 5 denied requests, got %d", deniedCount)
		}
	})
}

func TestTokenBucket_TokenRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Use a low rate for predictable refill testing
		// Rate: 5 tokens per second
		// Burst: 5 tokens
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(5.0, 5.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		// Exhaust the initial burst
		for i := 1; i <= 5; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				t.Errorf("Request %d: unexpected error: %v", i, err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i)
			}
		}

		// Should be denied now
		allowed, err := limiter.Allow(clientID)
		if err == nil || err == rle.ErrRateLimitExceeded {
			// Expected error
		} else {
			t.Errorf("Expected rate limit exceeded error, got: %v", err)
		}
		if allowed {
			t.Error("Request should be denied after burst exhausted")
		}

		// Wait for tokens to refill (1 second = 5 tokens)
		time.Sleep(1 * time.Second)

		// After 1 second, should have 5 more tokens
		allowedCount := 0
		for i := 1; i <= 5; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				t.Errorf("Request %d after refill: unexpected error: %v", i, err)
			}
			if allowed {
				allowedCount++
			}
		}

		if allowedCount != 5 {
			t.Errorf("Expected 5 allowed requests after refill, got %d", allowedCount)
		}
	})
}

func TestTokenBucket_MultipleClients(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter, err := NewTokenBucket(5.0, 5.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clients := []string{"alice", "bob", "charlie"}

		// Each client should have independent token buckets
		for _, client := range clients {
			for i := 1; i <= 5; i++ {
				allowed, err := limiter.Allow(client)
				if err != nil {
					t.Errorf("Client %s, request %d: unexpected error: %v", client, i, err)
				}
				if !allowed {
					t.Errorf("Client %s, request %d should be allowed", client, i)
				}
			}

			// Sixth request should be denied for each client
			allowed, err := limiter.Allow(client)
			if allowed {
				t.Errorf("Client %s: sixth request should be denied", client)
			}
			if err == nil {
				t.Errorf("Client %s: expected error for rate limit exceeded", client)
			}
		}
	})
}

func TestTokenBucket_InvalidClientID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter, err := NewTokenBucket(10.0, 10.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}

		// Test empty client ID
		allowed, err := limiter.Allow("")
		if err != rle.ErrInvalidClientID {
			t.Errorf("Allow() error = %v, wantErr %v", err, rle.ErrInvalidClientID)
		}
		if allowed {
			t.Error("Allow() should return false for invalid client ID")
		}
	})
}

func TestTokenBucket_InvalidRateLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test zero rate
		_, err := NewTokenBucket(0, 10.0, 1.0)
		if err != rle.ErrInvalidRateLimit {
			t.Errorf("NewTokenBucket() error = %v, wantErr %v", err, rle.ErrInvalidRateLimit)
		}

		// Test zero cost
		_, err = NewTokenBucket(10.0, 10.0, 0)
		if err != rle.ErrInvalidRateLimit {
			t.Errorf("NewTokenBucket() error = %v, wantErr %v", err, rle.ErrInvalidRateLimit)
		}

		// Test both zero
		_, err = NewTokenBucket(0, 10.0, 0)
		if err != rle.ErrInvalidRateLimit {
			t.Errorf("NewTokenBucket() error = %v, wantErr %v", err, rle.ErrInvalidRateLimit)
		}
	})
}

func TestTokenBucket_HighCostRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Rate: 10 tokens per second
		// Burst: 20 tokens
		// Cost: 5 tokens per request
		limiter, err := NewTokenBucket(10.0, 20.0, 5.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		allowedCount := 0
		deniedCount := 0

		// With 20 tokens initial burst and 5 per request, should allow 4 requests
		for i := 1; i <= 6; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				deniedCount++
			} else if allowed {
				allowedCount++
			}
		}

		if allowedCount != 4 {
			t.Errorf("Expected 4 allowed requests, got %d", allowedCount)
		}
		if deniedCount != 2 {
			t.Errorf("Expected 2 denied requests, got %d", deniedCount)
		}
	})
}

func TestTokenBucket_BurstCapacity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Rate: 10 tokens per second
		// Burst: 15 tokens (limits maximum stored tokens)
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(10.0, 15.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		// Make one request to initialize
		allowed, err := limiter.Allow(clientID)
		if err != nil || !allowed {
			t.Fatal("First request should be allowed")
		}

		// Wait for tokens to accumulate beyond burst
		time.Sleep(5 * time.Second)

		// Even though we waited 5 seconds (50 tokens at rate 10/s),
		// burst limits us to 15 tokens maximum
		// We already used 1, so we should have 14 + refilled tokens
		// But burst caps at 15, so we should have at most 15 tokens total
		allowedCount := 0
		for i := 1; i <= 20; i++ {
			allowed, _ := limiter.Allow(clientID)
			if allowed {
				allowedCount++
			}
		}

		// Should allow around 15 requests (burst capacity)
		if allowedCount > 16 {
			t.Errorf("Expected at most 16 allowed requests due to burst cap, got %d", allowedCount)
		}
	})
}

func TestTokenBucket_PartialTokens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test with fractional tokens
		// Rate: 2.5 tokens per second
		// Burst: 10 tokens
		// Cost: 2.5 tokens per request
		limiter, err := NewTokenBucket(2.5, 10.0, 2.5)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		// First request should succeed
		allowed, err := limiter.Allow(clientID)
		if err != nil {
			t.Errorf("First request: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("First request should be allowed")
		}

		// Second request should succeed (2.5 tokens remaining from 2.5 initial - 2.5 cost + 2.5 rate)
		allowed, err = limiter.Allow(clientID)
		if err != nil {
			t.Errorf("Second request: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Second request should be allowed")
		}
	})
}

func TestTokenBucket_ZeroBurst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Edge case: zero burst
		// Rate: 10 tokens per second
		// Burst: 0 (no token accumulation beyond current)
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(10.0, 0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		// First request should succeed (initializes with rate - cost)
		allowed, err := limiter.Allow(clientID)
		if err != nil {
			t.Errorf("First request: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("First request should be allowed")
		}

		// Immediate second request should fail (no burst capacity)
		allowed, err = limiter.Allow(clientID)
		if allowed {
			t.Error("Second request should be denied with zero burst")
		}
	})
}

func TestTokenBucket_HighThroughput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter, err := NewTokenBucket(100.0, 100.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		allowedCount := 0
		deniedCount := 0

		// Make 150 requests rapidly
		for range 150 {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				deniedCount++
			} else if allowed {
				allowedCount++
			}
		}

		if allowedCount != 100 {
			t.Errorf("Expected 100 allowed requests, got %d", allowedCount)
		}
		if deniedCount != 50 {
			t.Errorf("Expected 50 denied requests, got %d", deniedCount)
		}
	})
}

func TestTokenBucket_TimeAdvancement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test that demonstrates time advancement in synctest
		// Rate: 10 tokens per second
		// Burst: 5 tokens
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(10.0, 5.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		start := time.Now()

		// Use up initial tokens
		for i := 0; i < 5; i++ {
			limiter.Allow(clientID)
		}

		// This should be denied
		allowed, _ := limiter.Allow(clientID)
		if allowed {
			t.Error("Request should be denied after exhausting tokens")
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Time should not have advanced yet, but %v elapsed", elapsed)
		}

		// Advance time to refill tokens
		time.Sleep(1 * time.Second)

		// This should work after refill
		allowed, err = limiter.Allow(clientID)
		if err != nil {
			t.Errorf("Request after refill: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Request after refill should be allowed")
		}

		elapsed = time.Since(start)
		if elapsed < 1*time.Second {
			t.Errorf("Time should have advanced by at least 1s, but only %v elapsed", elapsed)
		}
	})
}

func TestTokenBucket_ConcurrentRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test concurrent requests from the same client
		limiter, err := NewTokenBucket(10.0, 10.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		results := make(chan bool, 20)

		// Launch 20 concurrent goroutines making requests
		for range 20 {
			go func() {
				allowed, _ := limiter.Allow(clientID)
				results <- allowed
			}()
		}

		// Wait for all goroutines to complete
		synctest.Wait()

		// Count allowed and denied requests
		allowedCount := 0
		deniedCount := 0
		for range 20 {
			if <-results {
				allowedCount++
			} else {
				deniedCount++
			}
		}

		// Note: Without mutex protection, the actual count may vary due to race conditions
		// This test demonstrates the need for concurrency safety
		t.Logf("Allowed: %d, Denied: %d (expected 10/10 with proper locking)", allowedCount, deniedCount)
	})
}

func TestTokenBucket_GradualRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test gradual token refill over time
		// Rate: 2 tokens per second
		// Burst: 10 tokens
		// Cost: 1 token per request
		limiter, err := NewTokenBucket(2.0, 10.0, 1.0)
		if err != nil {
			t.Fatalf("Failed to create token bucket: %v", err)
		}
		clientID := "user123"

		// Exhaust initial tokens
		for i := 0; i < 2; i++ {
			limiter.Allow(clientID)
		}

		// Next should fail
		allowed, _ := limiter.Allow(clientID)
		if allowed {
			t.Error("Should be denied after exhausting tokens")
		}

		// Wait 0.5 seconds (should refill 1 token)
		time.Sleep(500 * time.Millisecond)

		// Should be able to make 1 request
		allowed, err = limiter.Allow(clientID)
		if err != nil {
			t.Errorf("Request after partial refill: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Should be allowed after partial refill")
		}

		// Next should fail again
		allowed, _ = limiter.Allow(clientID)
		if allowed {
			t.Error("Should be denied again after using refilled token")
		}
	})
}

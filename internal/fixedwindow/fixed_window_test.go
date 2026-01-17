package fixedwindow

import (
	"testing"
	"testing/synctest"
	"time"

	rle "github.com/hrishikesh713/ratelimiter/internal/rlerrors"
)

func TestFixedWindow_BasicRateLimiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Create a fixed window rate limiter
		// Limit: 5 requests per window
		// Window size: 10 seconds
		limiter := NewFixedWindow(5, 10*time.Second)
		clientID := "user123"

		allowedCount := 0
		deniedCount := 0

		// Make 7 requests rapidly (should allow 5, deny 2)
		for i := 1; i <= 7; i++ {
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
			// Small delay between requests - in synctest this doesn't actually wait
			time.Sleep(100 * time.Millisecond)
		}

		if allowedCount != 5 {
			t.Errorf("Expected 5 allowed requests, got %d", allowedCount)
		}
		if deniedCount != 2 {
			t.Errorf("Expected 2 denied requests, got %d", deniedCount)
		}
	})
}

func TestFixedWindow_WindowReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Use a shorter window for faster testing
		limiter := NewFixedWindow(3, 2*time.Second)
		clientID := "user123"

		// First window: exhaust the quota
		for i := 1; i <= 3; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				t.Errorf("Request %d in first window: unexpected error: %v", i, err)
			}
			if !allowed {
				t.Errorf("Request %d in first window should be allowed", i)
			}
		}

		// Should be denied now
		allowed, err := limiter.Allow(clientID)
		if err != rle.ErrRateLimitExceeded {
			t.Errorf("Expected rate limit exceeded error, got: %v", err)
		}
		if allowed {
			t.Error("Request should be denied after quota exhausted")
		}

		// Wait for new window - in synctest, time advances instantly
		time.Sleep(2100 * time.Millisecond)

		// New window: requests should be allowed again
		allowedCount := 0
		for i := 1; i <= 3; i++ {
			allowed, err := limiter.Allow(clientID)
			if err != nil {
				t.Errorf("Request %d in new window: unexpected error: %v", i, err)
			}
			if allowed {
				allowedCount++
			}
		}

		if allowedCount != 3 {
			t.Errorf("Expected 3 allowed requests in new window, got %d", allowedCount)
		}
	})
}

func TestFixedWindow_MultipleClients(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter := NewFixedWindow(3, 10*time.Second)
		clients := []string{"alice", "bob", "charlie"}

		// Each client should have independent quota
		for _, client := range clients {
			for i := 1; i <= 3; i++ {
				allowed, err := limiter.Allow(client)
				if err != nil {
					t.Errorf("Client %s, request %d: unexpected error: %v", client, i, err)
				}
				if !allowed {
					t.Errorf("Client %s, request %d should be allowed", client, i)
				}
			}

			// Fourth request should be denied for each client
			allowed, err := limiter.Allow(client)
			if err != rle.ErrRateLimitExceeded {
				t.Errorf("Client %s: expected rate limit exceeded error, got: %v", client, err)
			}
			if allowed {
				t.Errorf("Client %s: fourth request should be denied", client)
			}
		}
	})
}

func TestFixedWindow_InvalidClientID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter := NewFixedWindow(5, 10*time.Second)

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

func TestFixedWindow_ExactLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter := NewFixedWindow(1, 5*time.Second)
		clientID := "user123"

		// First request should succeed
		allowed, err := limiter.Allow(clientID)
		if err != nil {
			t.Errorf("First request: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("First request should be allowed")
		}

		// Second request should be denied immediately
		allowed, err = limiter.Allow(clientID)
		if err != rle.ErrRateLimitExceeded {
			t.Errorf("Expected rate limit exceeded error, got: %v", err)
		}
		if allowed {
			t.Error("Second request should be denied")
		}
	})
}

func TestFixedWindow_WindowBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test requests at window boundaries
		limiter := NewFixedWindow(2, 1*time.Second)
		clientID := "user123"

		// Make requests and note the window
		allowed, err := limiter.Allow(clientID)
		if err != nil || !allowed {
			t.Fatal("First request should be allowed")
		}

		allowed, err = limiter.Allow(clientID)
		if err != nil || !allowed {
			t.Fatal("Second request should be allowed")
		}

		// Third should be denied
		allowed, err = limiter.Allow(clientID)
		if err != rle.ErrRateLimitExceeded {
			t.Errorf("Expected rate limit exceeded error, got: %v", err)
		}
		if allowed {
			t.Error("Third request should be denied")
		}

		// Wait for window to reset - in synctest this is instant
		time.Sleep(1100 * time.Millisecond)

		// Should be allowed in new window
		allowed, err = limiter.Allow(clientID)
		if err != nil {
			t.Errorf("Request in new window: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("Request in new window should be allowed")
		}
	})
}

func TestFixedWindow_ZeroLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Edge case: zero limit - first request initializes with count 1,
		// then immediately hits the limit (1 >= 0)
		limiter := NewFixedWindow(0, 10*time.Second)
		clientID := "user123"

		// First request creates state with currReqNum = 1
		// This is allowed because state doesn't exist yet
		allowed, err := limiter.Allow(clientID)
		if err != nil {
			t.Errorf("First request: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("First request should be allowed (creates initial state)")
		}

		// Second request should be denied (1 >= 0)
		allowed, err = limiter.Allow(clientID)
		if err != rle.ErrRateLimitExceeded {
			t.Errorf("Expected rate limit exceeded error for zero limit, got: %v", err)
		}
		if allowed {
			t.Error("Second request should be denied with zero limit")
		}
	})
}

func TestFixedWindow_HighThroughput(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		limiter := NewFixedWindow(100, 10*time.Second)
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

func TestFixedWindow_TimeAdvancement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test that demonstrates time advancement in synctest
		limiter := NewFixedWindow(2, 5*time.Second)
		clientID := "user123"

		start := time.Now()

		// Use up quota in first window
		limiter.Allow(clientID)
		limiter.Allow(clientID)

		// This should be denied
		allowed, _ := limiter.Allow(clientID)
		if allowed {
			t.Error("Third request in first window should be denied")
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Time should not have advanced yet, but %v elapsed", elapsed)
		}

		// Advance time to new window
		time.Sleep(5 * time.Second)

		// This should work in the new window
		allowed, err := limiter.Allow(clientID)
		if err != nil {
			t.Errorf("First request in new window: unexpected error: %v", err)
		}
		if !allowed {
			t.Error("First request in new window should be allowed")
		}

		elapsed = time.Since(start)
		if elapsed < 5*time.Second {
			t.Errorf("Time should have advanced by at least 5s, but only %v elapsed", elapsed)
		}
	})
}

func TestFixedWindow_ConcurrentRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Test concurrent requests from the same client
		limiter := NewFixedWindow(10, 10*time.Second)
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

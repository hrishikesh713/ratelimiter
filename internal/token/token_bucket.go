// Package token implements a token bucket rate limiting algorithm.
package token

import (
	"fmt"
	"math"
	"time"

	rle "github.com/hrishikesh713/ratelimiter/internal/rlerrors"
)

type TokenBucket struct {
	currTime time.Time
	rate     float64
	burst    float64
	cost     float64
	store    map[string]float64
}

func NewTokenBucket(rate, burst, cost float64) (*TokenBucket, error) {
	if rate == 0 || cost == 0 {
		return nil, rle.ErrInvalidRateLimit
	}
	return &TokenBucket{rate: rate, burst: burst, cost: cost, store: make(map[string]float64)}, nil
}

func (tb *TokenBucket) Allow(clientID string) (bool, error) {
	if len(clientID) == 0 {
		return false, rle.ErrInvalidClientID
	}
	now := time.Now()
	elapsed := now.Sub(tb.currTime)
	tb.currTime = now
	if elapsed.Nanoseconds() < 0 {
		return false, rle.ErrTimeSkewDetected
	}
	tokenCount, ok := tb.store[clientID]
	if !ok {
		tb.store[clientID] = tb.rate - tb.cost
		return true, nil
	}
	tokensElapsed := tb.rate * elapsed.Seconds()
	tokensToAdd := math.Min(tb.burst, tokensElapsed)
	tokenCount += tokensToAdd
	if t := tokenCount - tb.cost; t < 0 {
		tb.store[clientID] = tokenCount
		timeToWait := t / tb.rate
		return false, fmt.Errorf("retry after %s nanoseconds %w", time.Duration(timeToWait).String(), rle.ErrRateLimitExceeded)
	}
	tokenCount -= tb.cost
	tb.store[clientID] = tokenCount
	return true, nil
}

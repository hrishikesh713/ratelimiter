// Package fixedwindow implements a fixed window rate limiting algorithm.
package fixedwindow

import (
	"time"

	rle "github.com/hrishikesh713/ratelimiter/internal/rlerrors"
)

type FixedWindow struct {
	limit      int
	windowSize time.Duration
	store      map[string]*ClientState
}

type ClientState struct {
	currReqNum int
	currTime   time.Time
}

func NewFixedWindow(limit int, windowSize time.Duration) *FixedWindow {
	return &FixedWindow{limit: limit, windowSize: windowSize, store: make(map[string]*ClientState)}
}

func (fw *FixedWindow) Allow(clientID string) (bool, error) {
	if len(clientID) == 0 {
		return false, rle.ErrInvalidClientID
	}
	now := time.Now()
	windowStart := now.Truncate(fw.windowSize)
	windowEnd := windowStart.Add(fw.windowSize)
	state, ok := fw.store[clientID]
	if !ok {
		fw.store[clientID] = &ClientState{currReqNum: 1, currTime: now}
		return true, nil
	}
	ct := state.currTime
	state.currTime = now
	if ct.Before(windowStart) {
		state.currReqNum = 1
		return true, nil
	}
	if ct.Equal(windowStart) || ct.Before(windowEnd) {
		if state.currReqNum >= fw.limit {
			return false, rle.ErrRateLimitExceeded
		}
		state.currReqNum++
		return true, nil
	}
	// clock skew
	return false, rle.ErrRateLimitExceeded
}

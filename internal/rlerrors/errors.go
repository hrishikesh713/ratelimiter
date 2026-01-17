// Package rlerrors defines custom error types for rate limiting operations.
package rlerrors

import "errors"

var (
	ErrInvalidClientID   = errors.New("invalid client ID")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

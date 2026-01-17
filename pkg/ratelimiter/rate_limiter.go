// Package ratelimiter package provides a flexible rate limiting mechanism by allowing different rate limiting strategies to be plugged in.
package ratelimiter

import (
	"errors"
	"time"

	"github.com/hrishikesh713/ratelimiter/internal/fixedwindow"
)

type RateLimiter interface {
	Allow(string) (bool, error)
}

type RateLimit struct {
	rl     RateLimiter
	rlType string
}

type Option func(*RateLimit) error

func WithFixedWindow(limit int, windowsize time.Duration) Option {
	fw := fixedwindow.NewFixedWindow(limit, windowsize)
	return func(r *RateLimit) error {
		r.rl = RateLimiter(fw)
		r.rlType = "FixedWindow"
		return nil
	}
}

func WithTokenBucket() Option {
	return func(rl *RateLimit) error {
		return nil
	}
}

func NewRateLimit(opts ...Option) (*RateLimit, error) {
	r := RateLimit{}
	var cerr error
	for _, opt := range opts {
		if err := opt(&r); err != nil {
			cerr = errors.Join(cerr, err)
		}
	}
	return &r, cerr
}

func (r *RateLimit) Allow(clientID string) (bool, error) {
	return r.rl.Allow(clientID)
}

func (r *RateLimit) Type() string {
	return r.rlType
}

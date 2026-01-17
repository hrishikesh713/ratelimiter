package main

import (
	"fmt"
	"time"

	"github.com/hrishikesh713/ratelimiter/pkg/ratelimiter"
)

func main() {
	// Create a fixed window rate limiter
	// Limit: 5 requests per window
	// Window size: 10 seconds
	_, err := ratelimiter.NewRateLimit(ratelimiter.WithFixedWindow(5, 10*time.Second))
	if err != nil {
		fmt.Println("Error creating rate limiter:", err)
		return
	}
}

package flow

import "time"

type RateLimit struct {
	RPS   float64
	Burst int
}

type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

type TimeoutPolicy struct {
	Duration time.Duration
}

package backoff

import (
	"math"
	"math/rand"
	"time"
)

// Backoff keeps track of connection retry attempts and calculates the delay between each one.
type Backoff struct {
	// attempt tracks number of attempts so far
	attempt int
	// MaxAttempts specifies max attempts to try before erroring
	MaxAttempts int
	// BackoffInterval is duration to increment each attempt
	BackoffInterval float64
	// Jitter specifies deviation from backoff interval
	Jitter float64

	// MinInterval specifies minimum time allowed for backoff interval
	MinInterval time.Duration
	// MaxInterval specifies maximum time allowed for backoff interval
	MaxInterval time.Duration
}

const (
	defaultMaxAttempts     = 5
	defaultBackoffInterval = 2.0
	defaultJitter          = 0.05
	defaultMinInterval     = 1000 * time.Millisecond
	defaultMaxInterval     = 16 * time.Second
	// anything greater than this overflows time.Duration
	maxInt64 = float64(math.MaxInt64 - 512)
)

// DefaultBackoff creates a default configuration for Backoff.
func DefaultBackoff() *Backoff {
	return &Backoff{
		MaxAttempts:     defaultMaxAttempts,
		BackoffInterval: defaultBackoffInterval,
		Jitter:          defaultJitter,
		MinInterval:     defaultMinInterval,
		MaxInterval:     defaultMaxInterval,
	}
}

// NextDuration returns the duration and increases the number of attempts
func (b *Backoff) NextDuration() time.Duration {
	dur := b.ForAttempt(b.attempt)
	b.attempt++
	return dur
}

// TimeoutExceeded returns true if the backoff total duration has been exceeded
func (b *Backoff) TimeoutExceeded() bool {
	return float64(b.attempt) >= math.Max(0, float64(b.MaxAttempts))
}

// ForAttempt calculates the appropriate exponential duration given an attempt count
func (b *Backoff) ForAttempt(attempt int) time.Duration {
	min := b.MinInterval
	max := b.MaxInterval
	if min <= 0 {
		min = defaultMinInterval
	}
	if max <= 0 {
		max = defaultMaxInterval
	}
	if min >= max {
		return max
	}

	backoffInterval := b.BackoffInterval
	if backoffInterval <= 0 {
		backoffInterval = defaultBackoffInterval
	}

	// Calculate the new duration
	jitter := b.Jitter * 2 * (rand.Float64() - 0.5)
	durf := float64(min) * math.Pow(backoffInterval*(1-jitter), float64(attempt))

	// Check for overflow
	if durf > maxInt64 {
		return max
	}

	dur := time.Duration(durf)
	if dur < min {
		return min
	}
	if dur > max {
		return max
	}
	return dur
}

// Reset resets the attempt number for Backoff.
func (b *Backoff) Reset() {
	b.attempt = 0
}

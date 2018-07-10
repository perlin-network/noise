package backoff

import (
	"math"
	"time"
)

// Backoff keeps track of connection retry attempts and calculates the delay between each one.
type Backoff struct {
	attempt, MaxAttempts float64

	// Increment factor for each time step.
	Factor float64

	// Min and max intervals allowed for backoff intervals.
	MinInterval, MaxInterval time.Duration
}

const (
	defaultMaxAttempts = 5
	defaultFactor      = 2.0
	defaultMinInterval = 1000 * time.Millisecond
	defaultMaxInterval = 16 * time.Second
	maxInt64           = float64(math.MaxInt64 - 512)
)

// DefaultBackoff creates a default configuration for Backoff.
func DefaultBackoff() *Backoff {
	return &Backoff{
		attempt:     0,
		MaxAttempts: defaultMaxAttempts,
		Factor:      defaultFactor,
		MinInterval: defaultMinInterval,
		MaxInterval: defaultMaxInterval,
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
	return b.attempt >= math.Max(0, b.MaxAttempts)
}

// ForAttempt calculates the appropriate exponential duration given an attempt count
func (b *Backoff) ForAttempt(attempt float64) time.Duration {
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

	factor := b.Factor
	if factor <= 0 {
		factor = defaultFactor
	}

	// Calculate the new duration
	durf := float64(min) * math.Pow(factor, attempt)

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

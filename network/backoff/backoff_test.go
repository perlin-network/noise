package backoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTestBackoff() *Backoff {
	b := DefaultBackoff()

	b.MaxAttempts = 3
	b.BackoffInterval = 2
	b.MinInterval = 100 * time.Millisecond
	b.MaxInterval = 10 * time.Second

	return b
}

func TestBasic(t *testing.T) {
	t.Parallel()

	b := createTestBackoff()

	assert.InEpsilon(t, 0.1, b.NextDuration().Seconds(), 0.05)
	assert.InEpsilon(t, 0.2, b.NextDuration().Seconds(), 0.05)
	assert.Equal(t, false, b.TimeoutExceeded())
	assert.InEpsilon(t, 0.4, b.NextDuration().Seconds(), 0.05)
	assert.Equal(t, true, b.TimeoutExceeded())
}

func TestReset(t *testing.T) {
	t.Parallel()

	b := createTestBackoff()

	assert.InEpsilon(t, 0.1, b.NextDuration().Seconds(), 0.05)
	assert.InEpsilon(t, 0.2, b.NextDuration().Seconds(), 0.05)
	b.Reset()
	assert.InEpsilon(t, 0.1, b.NextDuration().Seconds(), 0.05)
	assert.InEpsilon(t, 0.2, b.NextDuration().Seconds(), 0.05)
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	b := createTestBackoff()

	b.MinInterval = 0 * time.Millisecond
	assert.Equal(t, defaultMinInterval, b.NextDuration())

	b.Reset()
	b.MaxInterval = 1 * time.Millisecond
	assert.Equal(t, b.MaxInterval, b.NextDuration())

	b.Reset()
	b.MinInterval = 2 * time.Millisecond
	b.MaxInterval = 1 * time.Millisecond
	assert.Equal(t, b.MaxInterval, b.NextDuration())

	b.Reset()
	b.MaxInterval = 0
	b.BackoffInterval = 0
	assert.Equal(t, 2*time.Millisecond, b.NextDuration())

	// trigger returning MaxInterval due to duration overflow
	b.BackoffInterval = maxInt64
	b.NextDuration()
	b.MinInterval = 3 * time.Second
	b.MaxInterval = 5 * time.Second
	assert.Equal(t, b.MaxInterval, b.NextDuration())

	// backoff duration less than min interval
	b.Reset()
	b.BackoffInterval = 0.1
	b.NextDuration()
	assert.Equal(t, b.MinInterval, b.NextDuration())

	// backoff duration greater than max interval
	b.Reset()
	b.BackoffInterval = 5
	b.NextDuration()
	assert.Equal(t, b.MaxInterval, b.NextDuration())
}

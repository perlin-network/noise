package backoff

import (
	"reflect"
	"testing"
	"time"
)

func assertEquals(t *testing.T, got, expected interface{}) {
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("got %v, expected %v", got, expected)
	}
}

func assertClose(t *testing.T, got, expected float64, diff float64) {
	if expected*(1.0+diff) < got || expected*(1.0-diff) > got {
		t.Fatalf("got %f, expected range [%f, %f]", got, expected*(1.0-diff), expected*(1.0+diff))
	}
}

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

	assertClose(t, b.NextDuration().Seconds(), 0.1, 0.1)
	assertClose(t, b.NextDuration().Seconds(), 0.2, 0.1)
	assertEquals(t, b.TimeoutExceeded(), false)
	assertClose(t, b.NextDuration().Seconds(), 0.4, 0.1)
	assertEquals(t, b.TimeoutExceeded(), true)
}

func TestReset(t *testing.T) {
	t.Parallel()

	b := createTestBackoff()

	assertClose(t, b.NextDuration().Seconds(), 0.1, 0.1)
	b.Reset()
	assertClose(t, b.NextDuration().Seconds(), 0.1, 0.1)
	assertClose(t, b.NextDuration().Seconds(), 0.2, 0.1)
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	b := createTestBackoff()

	b.MinInterval = 0 * time.Millisecond
	assertEquals(t, b.NextDuration(), defaultMinInterval)

	b.Reset()
	b.MaxInterval = 1 * time.Millisecond
	assertEquals(t, b.NextDuration(), 1*time.Millisecond)

	b.Reset()
	b.MinInterval = 2 * time.Millisecond
	b.MaxInterval = 1 * time.Millisecond
	assertEquals(t, b.NextDuration(), 1*time.Millisecond)
}

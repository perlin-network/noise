package backoff

import (
	"testing"
	"reflect"
	"time"
)

func assertEquals(t *testing.T, expected, got interface{}) {
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("Expected %v, got %v.", expected, got)
	}
}

func createTestBackoff() *Backoff {
	b := DefaultBackoff()

	b.MaxAttempts = 3
	b.Factor = 2
	b.MinInterval = 100 * time.Millisecond
	b.MaxInterval = 10 * time.Second

	return b
}

func TestBasic(t *testing.T) {
	b := createTestBackoff()

	assertEquals(t, b.NextDuration(), 100 * time.Millisecond)
	assertEquals(t, b.NextDuration(), 200 * time.Millisecond)
	assertEquals(t, b.TimeoutExceeded(), false)
	assertEquals(t, b.NextDuration(), 400 * time.Millisecond)
	assertEquals(t, b.TimeoutExceeded(), true)
}

func TestReset(t *testing.T) {
	b := createTestBackoff()

	assertEquals(t, b.NextDuration(), 100 * time.Millisecond)
	b.Reset()
	assertEquals(t, b.NextDuration(), 100 * time.Millisecond)
	assertEquals(t, b.NextDuration(), 200 * time.Millisecond)
}

func TestEdgeCases(t *testing.T) {
	b := createTestBackoff()

	b.MinInterval = 0 * time.Millisecond
	assertEquals(t, b.NextDuration(), defaultMinInterval)

	b.Reset()
	b.MaxInterval = 1 * time.Millisecond
	assertEquals(t, b.NextDuration(), 1 * time.Millisecond)

	b.Reset()
	b.MinInterval = 2 * time.Millisecond
	b.MaxInterval = 1 * time.Millisecond
	assertEquals(t, b.NextDuration(), 1 * time.Millisecond)
}

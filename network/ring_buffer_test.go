package network

import (
	"testing"
)

func TestRingBuffer(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(4)
	*rb.Index(0) = 1
	*rb.Index(1) = 2
	*rb.Index(2) = 3
	*rb.Index(3) = 4

	func() {
		defer func() {
			if recover() == nil {
				panic("expected panic")
			}
		}()
		*rb.Index(4) = 5
	}()

	rb.MoveForward(1)
	if (*rb.Index(0)).(int) != 2 || (*rb.Index(1)).(int) != 3 ||
		(*rb.Index(2)).(int) != 4 || (*rb.Index(3)).(int) != 1 {
		panic("incorrect value(s) after moving forward")
	}
}

func TestWrongPosOfIndex(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic is expected but not pop")
		}
	}()
	rb := NewRingBuffer(1)
	_ = rb.Index(-1)
}
func TestWrongMoveForward1(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic is expected but not pop")
		}
	}()
	rb := NewRingBuffer(1)
	*rb.Index(0) = 1
	rb.MoveForward(1)
}
func TestWrongMoveForwardCycle(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(2)
	*rb.Index(0) = 1
	*rb.Index(1) = 2

	rb.MoveForward(1)
	if rb.Position != 1 {
		t.Errorf("current position should be 1, got %d", rb.Position)
	}

	rb.MoveForward(1)
	if rb.Position != 0 {
		t.Errorf("current position should be 0, got %d", rb.Position)
	}
}

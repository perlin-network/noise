package network

import (
	"testing"
)

func TestRingBuffer(t *testing.T) {
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

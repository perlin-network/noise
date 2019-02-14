package blake2b

import (
	"bytes"
	"testing"
)

func TestHashBytes(t *testing.T) {
	t.Parallel()
	hp := New()

	r := hp.HashBytes([]byte("123"))

	n := []byte("123")
	if !bytes.Equal(n, r) {
		t.Errorf("Equal() n = %v, want %v", n, r)
	}
}

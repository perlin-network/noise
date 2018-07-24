package main

import (
	"testing"
)

// Usage:
//  vgo test -race .
func TestClient(t *testing.T) {
	t.Parallel()
	t.Log(run())
}

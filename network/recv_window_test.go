package network

import "testing"

func TestRecvWindow(t *testing.T) {
	r := NewRecvWindow(5)

	r.Push(1, "London")
	r.Push(2, "Berlin")
	r.Push(3, "Paris")
	r.Push(5, "Rome")

	vals := r.Pop()
	if len(vals) != 3 {
		t.Fatalf("expected 3, got %v", len(vals))
	}
	for i, v := range []interface{}{"London", "Berlin", "Paris"} {
		if v != vals[i] {
			t.Fatalf("expected `%v`, got `%v`", v, vals[i])
		}
	}

	r.Push(4, "Madrid")
	vals = r.Pop()

	if len(vals) != 2 {
		t.Fatalf("expected 2, got %v", len(vals))
	}
	for i, v := range []interface{}{"Madrid", "Rome"} {
		if v != vals[i] {
			t.Fatalf("expected `%v`, got `%v`", v, vals[i])
		}
	}
}

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

func TestRecvWindowRange(t *testing.T) {
	r := NewRecvWindow(5)

	r.Push(1, "London")
	r.Push(2, "Berlin")
	r.Push(3, "Paris")
	r.Push(5, "Rome")

	vals := r.Range(func(nonce uint64, v interface{}) bool {
		return nonce <= 3
	})

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

func TestRecvWindowDummyRange(t *testing.T) {
	r := NewRecvWindow(5)

	r.Push(1, "London")
	r.Push(2, "Berlin")
	r.Push(3, "Paris")
	r.Push(5, "Rome")

	vals := r.Range(func(nonce uint64, v interface{}) bool {
		return true
	})

	if len(vals) != 5 {
		t.Fatalf("expected 5, got %v", len(vals))
	}
}

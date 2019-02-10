package lru

import (
	"errors"
	"testing"
)

var (
	emptyFunc = func() (interface{}, error) { return "", nil }
)

func TestNormalSave(t *testing.T) {
	t.Parallel()

	cache := NewCache(1)
	myData := "mydata"
	testdata, err := cache.Get("mykey", func() (interface{}, error) {
		return myData, nil
	})
	if err != nil || testdata != myData {
		t.Fatalf("saving error, got : %v/%v", testdata, err)
	}
	data, err := cache.Get("mykey", emptyFunc)
	if data != myData || err != nil {
		t.Fatalf("loading error, got : %v/%v", testdata, err)
	}
}

func TestErrorSave(t *testing.T) {
	t.Parallel()

	cache := NewCache(1)
	myError := errors.New("myerror")

	testdata, err := cache.Get("mykey", func() (interface{}, error) {
		return "mydata", myError
	})
	if err != myError || testdata != nil {
		t.Fatalf("saving error, got : %v/%v", testdata, err)
	}
	data, err := cache.Get("mykey", emptyFunc)
	if data != "" || err != nil {
		t.Fatalf("loading error, got : %v/%v", data, err)
	}
}

func TestOldEntryDeleting(t *testing.T) {
	t.Parallel()

	cache := NewCache(2)
	cache.Get("mykey1", func() (interface{}, error) {
		return "mydata1", nil
	})
	cache.Get("mykey2", func() (interface{}, error) {
		return "mydata2", nil
	})
	cache.Get("mykey3", func() (interface{}, error) {
		return "mydata3", nil
	})
	data, err := cache.Get("mykey1", emptyFunc)
	if data != "" || err != nil {
		t.Fatalf("deleting error")
	}
}

func TestUnusedEntryDeleting(t *testing.T) {
	t.Parallel()

	cache := NewCache(2)
	cache.Get("mykey1", func() (interface{}, error) {
		return "mydata1", nil
	})
	cache.Get("mykey2", func() (interface{}, error) {
		return "mydata2", nil
	})
	cache.Get("mykey1", func() (interface{}, error) {
		return "mydata1pi", nil
	})

	cache.Get("mykey3", func() (interface{}, error) {
		return "mydata3", nil
	})
	data, err := cache.Get("mykey2", emptyFunc)
	if data != "" || err != nil {
		t.Fatalf("deleting error")
	}
}

package network

import (
	"reflect"
	"testing"
)

func TestResolveAddresses(t *testing.T) {
	oneResolvedAddr, err := ToUnifiedAddress("localhost:1000")
	if err != nil {
		t.Fatal(err)
	}

	testAddr := []string{
		"localhost:1000",
		"123.45.67.89:123",
	}
	expectedAddr := []string{
		oneResolvedAddr,
		testAddr[1],
	}

	resultAddr := resolveAddresses(testAddr)

	if !reflect.DeepEqual(resultAddr, expectedAddr) {
		t.Fatalf("Unexpected got %v, but expected %v", resultAddr, expectedAddr)
	}
}

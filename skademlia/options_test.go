package skademlia

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestOptions(t *testing.T) {
	keys, err := NewKeys(1, 1)
	if err != nil {
		t.Fatalf("error NewKeys(): %v", err)
	}

	c := NewClient(":0", keys, WithC1(1), WithC2(1),
		WithDialOptions(grpc.WithAuthority("authority")),
		WithPrefixDiffLen(64),
		WithPrefixDiffMin(16),
	)

	assert.Len(t, c.dopts, 1)
	assert.Equal(t, 64, c.prefixDiffLen)
	assert.Equal(t, 16, c.prefixDiffMin)
}

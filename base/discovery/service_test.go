package discovery_test

import (
	"github.com/perlin-network/noise/base/discovery"
	"github.com/perlin-network/noise/peer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestService(t *testing.T) {
	s := discovery.NewService(nil, peer.CreateID("", []byte{}))
	assert.NotNil(t, s)
}

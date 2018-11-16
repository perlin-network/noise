package discovery_test

import (
	"github.com/perlin-network/noise/base/discovery"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestService(t *testing.T) {
	s := discovery.NewService(nil, nil)
	assert.NotNil(t, s)
}

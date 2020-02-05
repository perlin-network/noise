package noise

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net"
	"testing"
	"testing/quick"
	"time"
)

func TestNodeOptions(t *testing.T) {
	a := func(a uint) bool {
		if a == 0 {
			a = 1
		}

		n, err := NewNode(WithNodeMaxDialAttempts(a))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.maxDialAttempts, a) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(a, &quick.Config{MaxCount: 10}))

	b := func(a uint) bool {
		if a == 0 {
			a = 128
		} else if a > 1000 {
			a = 1000
		}

		n, err := NewNode(WithNodeMaxInboundConnections(a))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.maxInboundConnections, a) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(b, &quick.Config{MaxCount: 10}))

	c := func(a uint) bool {
		if a == 0 {
			a = 128
		} else if a > 1000 {
			a = 1000
		}

		n, err := NewNode(WithNodeMaxOutboundConnections(a))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.maxOutboundConnections, a) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(c, &quick.Config{MaxCount: 10}))

	d := func(a time.Duration) bool {
		n, err := NewNode(WithNodeIdleTimeout(a))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.idleTimeout, a) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(d, &quick.Config{MaxCount: 10}))

	e := func(a bool) bool {
		var logger *zap.Logger
		if a {
			logger = zap.NewNop()
		}

		n, err := NewNode(WithNodeLogger(logger))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.NotNil(t, n.logger) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(e, &quick.Config{MaxCount: 10}))

	f := func(publicKey PublicKey, host net.IP, port uint16) bool {
		id := NewID(publicKey, host, port)

		n, err := NewNode(WithNodeID(id))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.id, id) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 10}))

	g := func(privateKey PrivateKey) bool {
		n, err := NewNode(WithNodePrivateKey(privateKey))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.privateKey, privateKey) {
			return false
		}

		if !assert.EqualValues(t, n.publicKey, privateKey.Public()) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(g, &quick.Config{MaxCount: 10}))

	h := func(host net.IP, port uint16, address string) bool {
		n, err := NewNode(WithNodeBindHost(host), WithNodeBindPort(port), WithNodeAddress(address))
		if !assert.NoError(t, err) {
			return false
		}

		if !assert.EqualValues(t, n.host, host) {
			return false
		}

		if !assert.EqualValues(t, n.port, port) {
			return false
		}

		if !assert.EqualValues(t, n.addr, address) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(h, &quick.Config{MaxCount: 10}))

	i := func(a uint) bool {
		n, err := NewNode(WithNodeNumWorkers(a))
		if !assert.NoError(t, err) {
			return false
		}

		if a > 0 && !assert.EqualValues(t, n.numWorkers, a) {
			return false
		}

		if a == 0 && !assert.EqualValues(t, n.numWorkers, 1) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(i, &quick.Config{MaxCount: 10}))
}

package network

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	keys     = ed25519.RandomKeyPair()
	host     = "localhost"
	protocol = "tcp"
	port     = uint16(12345)
)

func buildNetwork(port uint16) (*Network, error) {
	builder := NewBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(
		fmt.Sprintf("%s://%s:%d", protocol, host, port),
	)

	builder.AddPluginWithPriority(1, new(MockPlugin))

	return builder.Build()
}

func TestBuildNetwork(t *testing.T) {
	_, err := buildNetwork(port)

	if err != nil {
		t.Fatal(err)
	}
}

func TestSetters(t *testing.T) {
	net, err := buildNetwork(port)
	if err != nil {
		t.Fatal(err)
	}

	if net.Address != fmt.Sprintf("tcp://127.0.0.1:%d", port) { // Unified address.
		t.Fatalf("address is wrong: expected %s but got %s", fmt.Sprintf("tcp://127.0.0.1:%d", port), net.Address)
	}

	if !bytes.Equal(net.keys.PrivateKey, keys.PrivateKey) {
		t.Fatalf("private key is wrong")
	}

	if !bytes.Equal(net.keys.PublicKey, keys.PublicKey) {
		t.Fatalf("public key is wrong")
	}
}

func TestNoKeys(t *testing.T) {
	t.Parallel()

	builder := NewBuilder()
	builder.SetKeys(nil)
	_, err := builder.Build()
	if err == nil {
		t.Errorf("Build() = %+v, expected %+v", err, errors.New(ErrStrNoKeyPair))
	}
}

func TestBuilderAddress(t *testing.T) {
	t.Parallel()

	builder := NewBuilder()
	builder.SetAddress("")
	_, err := builder.Build()
	assert.NotEqual(t, nil, err)

	errMissingPort := errors.New("missing port in address")
	builder.SetAddress("localhost")
	_, err = builder.Build()
	if err == nil {
		t.Errorf("Build() = %+v, expected %+v", err, errMissingPort)
	}
}

func TestDuplicatePlugin(t *testing.T) {
	t.Parallel()

	builder := NewBuilder()
	_, err := builder.Build()

	assert.Equal(t, nil, err)
	assert.Equal(t, builder.pluginCount, 0, "should have no plugins")

	err = builder.AddPluginWithPriority(1, new(MockPlugin))
	assert.Equal(t, nil, err)

	err = builder.AddPluginWithPriority(1, new(MockPlugin))
	assert.NotEqual(t, nil, err)
}

func TestConnectionTimeout(t *testing.T) {
	t.Parallel()

	timeout := 5 * time.Second
	builder := NewBuilderWithOptions(ConnectionTimeout(timeout))
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.connectionTimeout, timeout, "connection timeout given should match found")
}

func TestSignaturePolicy(t *testing.T) {
	t.Parallel()

	signaturePolicy := ed25519.New()
	builder := NewBuilderWithOptions(SignaturePolicy(signaturePolicy))
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.signaturePolicy, signaturePolicy, "signature policy given should match found")
}

func TestHashPolicy(t *testing.T) {
	t.Parallel()

	hashPolicy := blake2b.New()
	builder := NewBuilderWithOptions(HashPolicy(hashPolicy))
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.hashPolicy, hashPolicy, "hash policy given should match found")
}

func TestWindowSize(t *testing.T) {
	t.Parallel()

	recvWindowSize := 2000
	sendWindowSize := 1000
	builder := NewBuilderWithOptions(
		RecvWindowSize(recvWindowSize),
		SendWindowSize(sendWindowSize),
	)
	net, err := builder.Build()
	if err != nil {
		t.Errorf("Build() = %+v, expected <nil>", err)
	}
	assert.Equal(t, net.opts.recvWindowSize, recvWindowSize, "recv window size given should match found")
	assert.Equal(t, net.opts.sendWindowSize, sendWindowSize, "send window size given should match found")
}

func TestWriteBufferSize(t *testing.T) {
	t.Parallel()

	writeBufferSize := 2048
	builder := NewBuilderWithOptions(
		WriteBufferSize(writeBufferSize),
	)
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.writeBufferSize, writeBufferSize, "write buffer size given should match found")
}

func TestWriteFlushLatency(t *testing.T) {
	t.Parallel()

	writeFlushLatency := 100 * time.Millisecond
	builder := NewBuilderWithOptions(
		WriteFlushLatency(writeFlushLatency),
	)
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.writeFlushLatency, writeFlushLatency, "write flush latency given should match found")
}

func TestWriteTimeout(t *testing.T) {
	t.Parallel()

	writeTimeout := 1 * time.Second
	builder := NewBuilderWithOptions(
		WriteTimeout(writeTimeout),
	)
	net, err := builder.Build()
	assert.Equal(t, nil, err)
	assert.Equal(t, net.opts.writeTimeout, writeTimeout, "write timeout given should match found")
}

func TestPeers(t *testing.T) {
	var nodes []*Network
	addresses := []string{"tcp://127.0.0.1:12345", "tcp://127.0.0.1:12346", "tcp://127.0.0.1:12347"}
	peers := [][2]int{{1, 2}, {0, 2}, {0, 1}}

	for i := 0; i < 3; i++ {
		net, err := buildNetwork(port + uint16(i))
		if err != nil {
			t.Fatal(err)
		}

		go net.Listen()
		net.BlockUntilListening()

		if i != 0 {
			net.Bootstrap(addresses[0])
		}
		nodes = append(nodes, net)
	}

	for x := 0; x < len(nodes); x++ {
		for y := 0; y < 2; y++ {
			if _, err := nodes[x].Client(addresses[peers[x][y]]); err != nil {
				t.Fatalf("nodes[%d] missing peer: %s", x, addresses[peers[x][y]])
			}
		}
	}
}

// Broadcast functions are tested through examples.

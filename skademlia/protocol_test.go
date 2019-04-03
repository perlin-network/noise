package skademlia

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fortytw2/leaktest"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/xnoise"
	"github.com/stretchr/testify/assert"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	C1 = 3
	C2 = 1
)

func overlay(t testing.TB, node *noise.Node, addr string) (*Protocol, *bufio.Reader) {
	t.Helper()

	keys, err := NewKeys(addr, C1, C2)
	assert.NoError(t, err)

	overlay := New(keys, xnoise.DialTCP)
	overlay.RegisterOpcodes(node)
	overlay.WithC1(C1)
	overlay.WithC2(C2)
	overlay.WithPrefixDiffMin(DefaultPrefixDiffMin)
	overlay.WithPrefixDiffLen(DefaultPrefixDiffLen)
	overlay.WithHandshakeTimeout(100 * time.Millisecond)

	writer := bytes.NewBuffer(nil)
	reader := bufio.NewReader(writer)
	overlay.Logger().SetOutput(writer)

	node.FollowProtocol(noise.NewProtocol(overlay.Protocol()))

	assert.NotZero(t, node.Opcode(OpcodePing))
	assert.NotZero(t, node.Opcode(OpcodeLookup))

	assert.Equal(t, overlay.c1, C1)
	assert.Equal(t, overlay.c2, C2)
	assert.Equal(t, overlay.prefixDiffMin, DefaultPrefixDiffMin)
	assert.Equal(t, overlay.prefixDiffLen, DefaultPrefixDiffLen)
	assert.Equal(t, overlay.handshakeTimeout, 100*time.Millisecond)
	assert.NotNil(t, overlay.logger)
	assert.NotNil(t, overlay.Logger())
	assert.NotNil(t, overlay.table)

	return overlay, reader
}

func TestProtocol(t *testing.T) {
	defer leaktest.Check(t)()

	alice, err := xnoise.ListenTCP(0)
	assert.NoError(t, err)

	bob, err := xnoise.ListenTCP(0)
	assert.NoError(t, err)

	charlie, err := xnoise.ListenTCP(0)
	assert.NoError(t, err)

	defer alice.Shutdown()
	defer bob.Shutdown()
	defer charlie.Shutdown()

	alicenet, alicelog := overlay(t, alice, net.JoinHostPort("127.0.0.1", strconv.Itoa(alice.Addr().(*net.TCPAddr).Port)))
	bobnet, boblog := overlay(t, bob, net.JoinHostPort("127.0.0.1", strconv.Itoa(bob.Addr().(*net.TCPAddr).Port)))
	charlienet, charlielog := overlay(t, charlie, net.JoinHostPort("127.0.0.1", strconv.Itoa(charlie.Addr().(*net.TCPAddr).Port)))

	aliceToBob, err := xnoise.DialTCP(alice, bobnet.keys.self.address)
	assert.NotNil(t, aliceToBob)
	assert.NoError(t, err)

	aliceToCharlie, err := xnoise.DialTCP(alice, charlienet.keys.self.address)
	assert.NotNil(t, aliceToCharlie)
	assert.NoError(t, err)

	aliceToBob.WaitFor(SignalAuthenticated)
	aliceToCharlie.WaitFor(SignalAuthenticated)

	alicenet.peersLock.Lock()
	numPeers := len(alicenet.peers)
	alicenet.peersLock.Unlock()

	assert.Equal(t, 2, numPeers)

	var bobToAlice *noise.Peer

	for bobToAlice == nil || bobToAlice.Addr().String() == alicenet.keys.self.address {
		bobToAlice = bob.Peers()[0]
	}

	bobToAlice.WaitFor(SignalAuthenticated)

	bobnet.peersLock.Lock()
	numPeers = len(bobnet.peers)
	bobnet.peersLock.Unlock()

	assert.Equal(t, 1, numPeers)

	var charlieToAlice *noise.Peer

	for charlieToAlice == nil || charlieToAlice.Addr().String() == alicenet.keys.self.address {
		charlieToAlice = charlie.Peers()[0]
	}

	charlieToAlice.WaitFor(SignalAuthenticated)

	charlienet.peersLock.Lock()
	numPeers = len(charlienet.peers)
	charlienet.peersLock.Unlock()

	assert.Equal(t, 1, numPeers)

	t.Run("can properly handshake", func(t *testing.T) {
		// Check that log messages get print correctly.

		bobPrinted := true

		line, _, err := alicelog.ReadLine()
		if err == nil {
			line := string(line)

			assert.Contains(t, line, "Registered to S/Kademlia")

			if bobPrinted = strings.Contains(line, bobnet.table.self.String()); !bobPrinted {
				assert.Contains(t, line, charlienet.table.self.String())
			} else {
				assert.Contains(t, line, bobnet.table.self.String())
			}
		}

		line, _, err = alicelog.ReadLine()
		if err == nil {
			line := string(line)

			assert.Contains(t, line, "Registered to S/Kademlia")

			if bobPrinted {
				assert.Contains(t, line, charlienet.table.self.String())
			} else {
				assert.Contains(t, line, bobnet.table.self.String())
			}
		}

		line, _, err = boblog.ReadLine()
		if err == nil {
			assert.Contains(t, string(line), fmt.Sprintf("Registered to S/Kademlia: %s", alicenet.table.self))
		}

		line, _, err = charlielog.ReadLine()
		if err == nil {
			assert.Contains(t, string(line), fmt.Sprintf("Registered to S/Kademlia: %s", alicenet.table.self))
		}
	})

	t.Run("evicts invalid peers", func(t *testing.T) {
		fakeKeys, err := NewKeys("fake_address", C1, C2)
		assert.NoError(t, err)

		assert.NoError(t, alicenet.Update(fakeKeys.ID()))
		assert.Equal(t, 2, len(alicenet.Peers(alice)))

		line, _, err := alicelog.ReadLine()
		assert.NoError(t, err)

		// We might have some of the logs appear multiple times. Buffer them away.
		for bytes.Contains(line, []byte("Registered to S/Kademlia")) {
			line, _, err = alicelog.ReadLine()
			assert.NoError(t, err)
		}

		assert.Contains(t, string(line), fmt.Sprintf("Peer %s could not be reached, and has been evicted.", fakeKeys.ID()))
	})

	t.Run("able to bootstrap", func(t *testing.T) {
		ids := bobnet.Bootstrap(bob)

		assert.Equal(t, 2, len(ids))

		for _, id := range ids {
			if id.checksum != charlienet.keys.self.checksum {
				assert.Equal(t, id.checksum, alicenet.keys.self.checksum)
			}

			if id.checksum != alicenet.keys.self.checksum {
				assert.Equal(t, id.checksum, charlienet.keys.self.checksum)
			}
		}

		time.Sleep(10 * time.Millisecond)

		assert.Len(t, alicenet.Peers(alice), 2)
		assert.Len(t, bobnet.Peers(alice), 2)
		assert.Len(t, charlienet.Peers(alice), 2)
	})

	t.Run("spam pings", func(t *testing.T) {
		assert.Len(t, alicenet.Peers(alice), 2)

		for i := 0; i < 10; i++ {
			id, err := alicenet.Ping(aliceToBob.Ctx())

			if !assert.NoError(t, err) || !assert.NotZero(t, id) {
				break
			}
		}

		assert.Len(t, alicenet.Peers(alice), 2)
	})

	t.Run("correctly executes eviction policy when table is full", func(t *testing.T) {
		fakeKeys, err := NewKeys("fake_address", C1, C2)
		assert.NoError(t, err)

		// Generate an ID in the same bucket.
		for {
			b1 := getBucketID(alicenet.table.self.checksum, bobnet.table.self.checksum)
			b2 := getBucketID(alicenet.table.self.checksum, fakeKeys.checksum)

			if b1 == b2 {
				break
			}

			fakeKeys, err = NewKeys("fake_address", C1, C2)
			assert.NoError(t, err)
		}

		assert.Len(t, alicenet.Peers(alice), 2)

		bucket := alicenet.table.buckets[getBucketID(alicenet.table.self.checksum, bobnet.table.self.checksum)]
		numPeers := bucket.Len()

		original := alicenet.table.getBucketSize()
		alicenet.table.setBucketSize(numPeers)

		assert.Error(t, alicenet.Update(fakeKeys.ID()))

		alicenet.table.setBucketSize(original)
		assert.Len(t, alicenet.Peers(alice), 2)
	})
}

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
	"testing"
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

	defer alice.Shutdown()
	defer bob.Shutdown()

	alicenet, alicelog := overlay(t, alice, net.JoinHostPort("127.0.0.1", strconv.Itoa(alice.Addr().(*net.TCPAddr).Port)))
	bobnet, boblog := overlay(t, bob, net.JoinHostPort("127.0.0.1", strconv.Itoa(bob.Addr().(*net.TCPAddr).Port)))

	aliceToBob, err := xnoise.DialTCP(alice, bobnet.keys.self.address)
	assert.NoError(t, err)

	var bobToAlice *noise.Peer

	t.Run("can properly handshake", func(t *testing.T) {
		aliceToBob.WaitFor(SignalAuthenticated)

		alicenet.peersLock.Lock()
		numPeers := len(alicenet.peers)
		alicenet.peersLock.Unlock()

		assert.Equal(t, 1, numPeers)

		for bobToAlice == nil || bobToAlice.Addr().String() == alicenet.keys.self.address {
			bobToAlice = bobnet.PeerByID(bob, alicenet.keys.self)
		}

		bobToAlice.WaitFor(SignalAuthenticated)

		bobnet.peersLock.Lock()
		numPeers = len(bobnet.peers)
		bobnet.peersLock.Unlock()

		assert.Equal(t, 1, numPeers)

		// Check that log messages get print correctly.
		line, _, err := alicelog.ReadLine()
		if err == nil {
			assert.Contains(t, fmt.Sprintf("Registered to S/Kademlia: %s", bobnet.table.self), string(line))
		}

		line, _, err = boblog.ReadLine()
		if err == nil {
			assert.Contains(t, fmt.Sprintf("Registered to S/Kademlia: %s", alicenet.table.self), string(line))
		}
	})

	t.Run("evicts invalid peers", func(t *testing.T) {
		fakeKeys, err := NewKeys("fake_address", C1, C2)
		assert.NoError(t, err)

		assert.NoError(t, alicenet.Update(fakeKeys.ID()))
		assert.Equal(t, 1, len(alicenet.Peers(alice)))

		line, _, err := alicelog.ReadLine()
		assert.NoError(t, err)

		// We might have some of the logs appear multiple times. Buffer them away.
		for bytes.Contains(line, []byte(fmt.Sprintf("Registered to S/Kademlia: %s", bobnet.table.self))) {
			line, _, err = alicelog.ReadLine()
			assert.NoError(t, err)
		}

		assert.Contains(t, fmt.Sprintf("Peer %s could not be reached, and has been evicted.", fakeKeys.ID()), string(line))
	})

	t.Run("able to bootstrap", func(t *testing.T) {
		peers := bobnet.Bootstrap(bob)

		assert.Equal(t, 1, len(peers))
		assert.Equal(t, peers[0].String(), alicenet.keys.self.String())
	})

	//t.Run("spam pings", func(t *testing.T) {
	//	assert.Len(t, alicenet.Peers(alice), 1)
	//
	//	for i := 0; i < 100; i++ {
	//		id, err := alicenet.Ping(aliceToBob.Ctx())
	//
	//		if !assert.NotZero(t, id) || !assert.NoError(t, err) {
	//			break
	//		}
	//	}
	//})

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

		original := alicenet.table.bucketSize
		alicenet.table.bucketSize = 1

		assert.Len(t, alicenet.Peers(alice), 1)

		// Lets ping our peer just to be sure that he is actually alive.
		id, err := alicenet.Ping(aliceToBob.Ctx())

		if err != nil {
			assert.Zero(t, id)
			assert.Len(t, alicenet.Peers(alice), 1)
		} else {
			assert.EqualValues(t, id, bobnet.table.self)
		}

		// The update function will ping our peer one more time. Since we are using
		// live TCP connections, it is possible the ping will fail and our fake ID
		// will be placed within the routing table.

		if err := alicenet.Update(fakeKeys.ID()); err != nil {
			assert.Len(t, alicenet.Peers(alice), 1)
		} else {
			assert.Len(t, alicenet.Peers(alice), 0)
		}

		alicenet.table.bucketSize = original
	})
}

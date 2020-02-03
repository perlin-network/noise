package gossip_test

import (
	"context"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/gossip"
	"github.com/perlin-network/noise/kademlia"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"sync"
	"testing"
)

func TestGossip(t *testing.T) {
	defer goleak.VerifyNone(t)

	nodes := make([]*noise.Node, 0, 16)
	overlays := make([]*kademlia.Protocol, 0, cap(nodes))

	seen := make(map[noise.PublicKey]struct{}, cap(nodes))
	cond := sync.NewCond(&sync.Mutex{})

	for i := 0; i < cap(nodes); i++ {
		node, err := noise.NewNode()
		assert.NoError(t, err)
		defer node.Close()

		overlay := kademlia.New()
		hub := gossip.New(overlay,
			gossip.WithEvents(
				gossip.Events{
					OnGossipReceived: func(sender noise.ID, data []byte) error {
						cond.L.Lock()
						seen[node.ID().ID] = struct{}{}
						cond.Signal()
						cond.L.Unlock()

						return nil
					},
				},
			),
		)

		node.Bind(
			overlay.Protocol(),
			hub.Protocol(),
		)

		assert.NoError(t, node.Listen())

		nodes = append(nodes, node)
		overlays = append(overlays, overlay)
	}

	leader, err := noise.NewNode()
	assert.NoError(t, err)
	defer leader.Close()

	overlay := kademlia.New()
	hub := gossip.New(overlay)

	leader.Bind(
		overlay.Protocol(),
		hub.Protocol(),
	)

	assert.NoError(t, leader.Listen())

	var wg sync.WaitGroup
	wg.Add(len(nodes) * (len(nodes) - 1))

	for i := range nodes {
		for j := range nodes {
			i, j := i, j

			if i == j {
				continue
			}

			go func() {
				_, err := nodes[i].Ping(context.TODO(), leader.Addr())
				assert.NoError(t, err)

				_, err = nodes[i].Ping(context.TODO(), nodes[j].Addr())
				assert.NoError(t, err)

				wg.Done()
			}()
		}
	}

	wg.Wait()

	for _, node := range nodes {
		for _, client := range append(node.Inbound(), node.Outbound()...) {
			client.WaitUntilReady()
		}
	}

	hub.Push(context.TODO(), []byte("hello!"))

	cond.L.Lock()
	for len(seen) != len(nodes) {
		cond.Wait()
	}
	cond.L.Unlock()
}

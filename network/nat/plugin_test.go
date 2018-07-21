package nat

import (
	"flag"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/perlin-network/noise/network"
	"github.com/stretchr/testify/assert"
)

func init() {
	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "4", "test")
	flag.Lookup("v").Value.Set(logLevel)
}

func TestRegisterPlugin(t *testing.T) {
	t.Parallel()

	b := network.NewBuilder()
	RegisterPlugin(b)
	n, err := b.Build()
	assert.Equal(t, nil, err)
	p, ok := n.Plugins.Get(PluginID)
	assert.Equal(t, true, ok)
	natPlugin := p.(*plugin)
	assert.NotEqual(t, nil, natPlugin)
}

func getPeers(m *sync.Map) []*network.PeerClient {
	peers := make([]*network.PeerClient, 0)
	m.Range(func(key, value interface{}) bool {
		client := value.(*network.PeerClient)

		peers = append(peers, client)

		return true
	})

	return peers
}

func TestNatConnect(t *testing.T) {
	t.Parallel()

	numNodes := 2
	nodes := make([]*network.Network, 0)
	for i := 0; i < numNodes; i++ {
		b := network.NewBuilder()
		port := network.GetRandomUnusedPort()
		addr := network.FormatAddress("tcp", "localhost", uint16(port))
		lis, err := network.NewTcpListener(addr)
		b.SetAddress(addr)
		RegisterPlugin(b)
		n, err := b.Build()
		assert.Equal(t, nil, err, "%+v", err)
		go n.Listen(lis)

		assert.Equal(t, nil, err)
		pInt, ok := n.Plugins.Get(PluginID)
		assert.Equal(t, true, ok)
		p := pInt.(*plugin)
		assert.NotEqual(t, nil, p)
		n.BlockUntilListening()
		nodes = append(nodes, n)
	}

	time.Sleep(1 * time.Second)

	nodes[1].Bootstrap(nodes[0].Address)
	peers := getPeers(nodes[1].Peers)
	for len(peers) < numNodes-1 {
		peers = getPeers(nodes[1].Peers)
		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, len(peers), 1)
}

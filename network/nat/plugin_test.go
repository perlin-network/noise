package nat

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"
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

func TestNatConnect(t *testing.T) {
	t.Parallel()

	numNodes := 2
	nodes := make([]*network.Network, 0)
	for i := 0; i < numNodes; i++ {
		b := network.NewBuilder()
		port := network.GetRandomUnusedPort()
		addr := network.FormatAddress("tcp", "localhost", uint16(port))
		b.SetAddress(addr)
		RegisterPlugin(b)
		b.AddPlugin(new(discovery.Plugin))
		n, err := b.Build()
		lis, err := network.NewTcpListener(addr)
		assert.Equal(t, nil, err, "%+v", err)
		go n.Listen(lis)

		assert.Equal(t, nil, err)
		pInt, ok := n.Plugins.Get(PluginID)
		assert.Equal(t, true, ok)
		p := pInt.(*plugin)
		assert.NotEqual(t, nil, p)
		nodes = append(nodes, n)
		n.BlockUntilListening()
		time.Sleep(100 * time.Millisecond)
	}

	nodes[1].Bootstrap(nodes[0].Address)
	pluginInt, ok := nodes[1].Plugin(discovery.PluginID)
	assert.Equal(t, true, ok)
	plugin := pluginInt.(*discovery.Plugin)
	routes := plugin.Routes
	peers := routes.GetPeers()
	for len(peers) < numNodes-1 {
		peers = routes.GetPeers()
		time.Sleep(50 * time.Millisecond)
	}

	assert.Equal(t, len(peers), 1)
}

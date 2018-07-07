package backoff

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/protobuf"
)

type Plugin struct {
	*network.Plugin

	net *network.Network
	// map[string]*Backoff
	backoffs sync.Map
}

var (
	PluginID        = (*Plugin)(nil)
	initialDelay    = 5 * time.Second
	limitIterations = 100
)

// Startup implements the plugin callback
func (p *Plugin) Startup(net *network.Network) {
	p.net = net
}

// PeerDisconnect implements the plugin callback
func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	addr := client.Address

	go p.startBackoff(addr)
}

// startBackoff uses an exponentially increasing timer to try to reconnect to a given address
func (p *Plugin) startBackoff(addr string) {
	time.Sleep(initialDelay)

	if _, exists := p.backoffs.Load(addr); exists {
		// don't activate if backoff is already active
		glog.Infof("backoff skipped for addr %s, already active\n", addr)
		return
	}
	// reset the backoff counter
	p.backoffs.Store(addr, DefaultBackoff())
	startTime := time.Now()
	for i := 0; i < limitIterations; i++ {
		s, active := p.backoffs.Load(addr)
		if !active {
			break
		}
		b := s.(*Backoff)
		if b.TimeoutExceeded() {
			// check if the backoff expired
			glog.Infof("backoff ended for addr %s, timed out after %s\n", addr, time.Now().Sub(startTime))
			break
		}
		// sleep for a bit before connecting
		d := b.NextDuration()
		glog.Infof("backoff reconnecting to %s in %s iteration %d", addr, d, i+1)
		time.Sleep(d)
		if p.checkConnected(addr) {
			// check that the connection is still empty before dialing
			break
		}
		// dial the client and see if it is successful
		c, err := p.net.Client(addr)
		if err != nil {
			continue
		}
		if !p.checkConnected(addr) {
			// check if successfully connected
			continue
		}
		if err := c.Tell(&protobuf.Ping{}); err != nil {
			// ping failed, not really connected
			continue
		}
		// success
		break
	}
	// clean up this backoff
	p.backoffs.Delete(addr)
}

// checkConnected is a helper function to check if the address is connected to the node
func (p *Plugin) checkConnected(addr string) bool {
	_, connected := p.net.Connections.Load(addr)
	return connected
}

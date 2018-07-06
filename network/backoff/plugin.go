package backoff

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
)

type Plugin struct {
	*network.Plugin

	// map[string]*Backoff
	backoffs sync.Map
}

var (
	PluginID        = (*Plugin)(nil)
	initialDelay    = 5 * time.Second
	limitIterations = 100
)

func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	addr := client.Address

	if _, exists := p.backoffs.Load(addr); exists {
		// don't activate if it already active
		glog.Infof("backoff skipped, already active\n")
		return
	}
	go func() {
		p.startBackoff(addr, client)
	}()
}

func (p *Plugin) startBackoff(addr string, client *network.PeerClient) {
	// this callback is called before the disconnect, so wait until disconnected
	time.Sleep(initialDelay)

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
			glog.Infof("backoff ended for addr %s, timed out after %s\n", addr, time.Now().Sub(startTime))
			break
		}
		d := b.NextDuration()
		glog.Infof("backoff reconnecting to %s in %s iteration %d", addr, d, i+1)
		time.Sleep(d)
		if p.checkConnected(client, addr) {
			// check that the connection is still empty before dialing
			break
		}
		if _, err := client.Network.Dial(client.Address); err != nil {
			client.Close()
			continue
		}
		if !p.checkConnected(client, addr) {
			// check if successfully connected
			continue
		}
		// success
		break
	}
	// clean up this back off
	p.backoffs.Delete(addr)
}

func (p *Plugin) checkConnected(client *network.PeerClient, addr string) bool {
	connected := false
	// check if the peer is still disconnected
	client.Network.Peers.Range(func(k string, pc *network.PeerClient) bool {
		// seems the peer is disconnected while pc.ID == nil
		if k == addr && pc.ID != nil {
			connected = true
		}
		return true
	})
	return connected
}
package backoff

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/network"
)

type Plugin struct {
	*network.Plugin

	sync.Mutex
	backoffs map[string]*Backoff
}

var PluginID = (*Plugin)(nil)

func (p *Plugin) Startup(net *network.Network) {
	if p.backoffs == nil {
		p.backoffs = map[string]*Backoff{}
	}
}

func (p *Plugin) Receive(ctx *network.MessageContext) error {
	addr := ctx.Sender().Address
	if _, exists := p.backoffs[addr]; exists {
		// if there is an active backoff, clear it
		delete(p.backoffs, addr)
	}
	return nil
}

func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	addr := client.Address

	if _, exists := p.backoffs[addr]; exists {
		// don't activate if it already active
		glog.Infof("backing off done already active\n")
		return
	}
	go func() {
		time.Sleep(5000 * time.Millisecond)
		// reset the backoff counter
		p.backoffs[addr] = DefaultBackoff()
		for {
			b, active := p.backoffs[addr]
			if !active {
				glog.Infof("backing off done already ended\n")
				break
			}
			glog.Infof("backing off: addr=%s b=%v\n", addr, b)
			if b.TimeoutExceeded() {
				glog.Infof("backing done timeout exceeded for add=%s\n", addr)
				delete(p.backoffs, addr)
				break
			}
			d := b.NextDuration()
			glog.Infof("backing off reconnecting to %s in %s", addr, d)
			time.Sleep(d)
			if _, err := client.Network.Dial(client.Address); err != nil {
				glog.Infof("backing off dial error %s to addr %s\n", err, addr)
				continue
			}
			peerConnected := false
			var peers []string
			client.Network.Peers.Range(func(k string, pc *network.PeerClient) bool {
				if k == addr {
					peerConnected = true
				}
				peers = append(peers, k)
				return true
			})
			if !peerConnected {
				glog.Infof("backing off still not connected to peer %s\n", addr)
				continue
			}
			glog.Infof("backing off done successfully reconnected to %s\n", addr)
			// success
			delete(p.backoffs, addr)
		}
	}()
}

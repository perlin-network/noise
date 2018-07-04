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
		glog.Infof("backing done already active\n")
		return
	}
	go func() {
		time.Sleep(1000 * time.Millisecond)
		// reset the backoff counter
		p.backoffs[addr] = DefaultBackoff()
		for {
			b, active := p.backoffs[addr]
			if !active {
				glog.Infof("backing done already ended\n")
				break
			}
			glog.Infof("backing off: addr=%s b=%v\n", addr, b)
			if b.TimeoutExceeded() {
				glog.Infof("backing done timeout exceeded for add=%s\n", addr)
				delete(p.backoffs, addr)
				break
			}
			peer, err := client.Network.Dial(client.Address)
			if err != nil {
				d := b.NextDuration()
				glog.Infof("%s, reconnecting to %s in %s", err, addr, d)
				time.Sleep(d)
				continue
			}
			if peer == nil {
				glog.Infof("backing error, peer was nil for addr=%s\n", addr)
				continue
			}
			glog.Infof("backing done successfully reconnected to %s\n", addr)
			// success
			delete(p.backoffs, addr)
		}
	}()
}

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
	isActive bool
	backoff  *Backoff
}

var PluginID = (*Plugin)(nil)

func (p *Plugin) Startup(net *network.Network) {
	if p.isActive {
		p.isActive = false
	}
}

func (p *Plugin) Receive(ctx *network.MessageContext) error {
	// no op
	return nil
}

func (p *Plugin) Cleanup(net *network.Network) {
	p.isActive = false
}

func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	// Delete peer if in routing table.

	if p.isActive {
		return
	}
	go func() {
		// reset the backoff counter
		p.backoff = DefaultBackoff()
		p.isActive = true
		for {
			if !p.isActive {
				break
			}
			if p.backoff.TimeoutExceeded() {
				p.isActive = false
				break
			}
			if _, err := client.Network.Dial(client.Address); err != nil {
				d := p.backoff.NextDuration()
				glog.Infof("%s, reconnecting in %s", err, d)
				time.Sleep(d)
				continue
			}
			// success
			p.isActive = false
		}
	}()
}

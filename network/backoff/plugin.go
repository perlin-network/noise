package backoff

import (
	"sync"
	"time"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/network"

	"github.com/golang/glog"
)

const (
	defaultPluginInitialDelay = 5 * time.Second
	defaultPluginMaxAttempts  = 100
	defaultPluginPriority     = 100
)

// Plugin is the backoff plugin
type Plugin struct {
	*network.Plugin

	// plugin options
	// initialDelay specifies initial backoff interval
	initialDelay time.Duration
	// maxAttempts specifies total number of retries
	maxAttempts int
	// priority specifies plugin priority
	priority int

	net      *network.Network
	backoffs sync.Map
}

// PluginOption are configurable options for the backoff plugin
type PluginOption func(*Plugin)

// WithInitialDelay specifies initial backoff interval
func WithInitialDelay(d time.Duration) PluginOption {
	return func(o *Plugin) {
		o.initialDelay = d
	}
}

// WithMaxAttempts specifies max attempts to retry upon client disconnect
func WithMaxAttempts(i int) PluginOption {
	return func(o *Plugin) {
		o.maxAttempts = i
	}
}

// WithPriority specifies plugin priority
func WithPriority(i int) PluginOption {
	return func(o *Plugin) {
		o.priority = i
	}
}

func defaultOptions() PluginOption {
	return func(o *Plugin) {
		o.initialDelay = defaultPluginInitialDelay
		o.maxAttempts = defaultPluginMaxAttempts
		o.priority = defaultPluginPriority
	}
}

var (
	_ network.PluginInterface = (*Plugin)(nil)
	// PluginID is used to check existence of the backoff plugin
	PluginID = (*Plugin)(nil)
)

// New returns a new backoff plugin with specified options
func New(opts ...PluginOption) *Plugin {
	p := new(Plugin)
	defaultOptions()(p)

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Startup implements the plugin callback
func (p *Plugin) Startup(net *network.Network) {
	p.net = net
}

// PeerDisconnect implements the plugin callback
func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	go p.startBackoff(client.Address)
}

// startBackoff uses an exponentially increasing timer to try to reconnect to a given address
func (p *Plugin) startBackoff(addr string) {
	time.Sleep(p.initialDelay)

	if _, exists := p.backoffs.Load(addr); exists {
		// don't activate if backoff is already active
		glog.Infof("backoff skipped for addr %s, already active\n", addr)
		return
	}
	// reset the backoff counter
	p.backoffs.Store(addr, DefaultBackoff())
	startTime := time.Now()
	for i := 0; i < p.maxAttempts; i++ {
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
		if p.net.ConnectionStateExists(addr) {
			// check that the connection is still empty before dialing
			break
		}
		// dial the client and see if it is successful
		c, err := p.net.Client(addr)
		if err != nil {
			continue
		}
		if !p.net.ConnectionStateExists(addr) {
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

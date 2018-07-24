package network

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/types"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
)

const (
	defaultDataShards   = 0
	defaultParityShards = 0
)

// KCPPlugin provides pluggable transport protocol
type KCPPlugin struct {
	*Transport

	opts kcpOptions
}

var (
	// KCPPluginID to reference transport plugin
	KCPPluginID                    = (*KCPPlugin)(nil)
	_           TransportInterface = (*KCPPlugin)(nil)
)

type kcpOptions struct {
	dataShards   int
	parityShards int
}

var defaultKCPOptions = kcpOptions{
	dataShards:   defaultDataShards,
	parityShards: defaultParityShards,
}

// A KCPOption sets kcpOptions for the kcp protocol
type KCPOption func(*kcpOptions)

// DataShards sets the data shards for kcp (default: 0).
func DataShards(n int) KCPOption {
	return func(o *kcpOptions) {
		o.dataShards = n
	}
}

// ParityShards sets the parity shards for kcp (default: 0).
func ParityShards(n int) KCPOption {
	return func(o *kcpOptions) {
		o.parityShards = n
	}
}

// NewListener creates a new transport protocol listener using given address
func (p *KCPPlugin) NewListener(addr string) (net.Listener, error) {
	addrInfo, err := types.ParseAddress(addr)
	if err != nil {
		return nil, err
	}

	lis, err := kcp.ListenWithOptions(fmt.Sprintf("localhost:%d", addrInfo.Port), nil, p.opts.dataShards, p.opts.parityShards)
	if err != nil {
		return nil, err
	}

	return lis, nil
}

// Listen starts listening on the transport protocol
func (p *KCPPlugin) Listen(net *Network) {
	lis, err := p.NewListener(p.address)
	if err != nil {
		glog.Errorf("transport: %+v", err)
		return
	}

	p.lis = lis

	// Handle new clients.
	go func() {
		for {
			if conn, err := p.lis.Accept(); err == nil {
				go net.Accept(conn)
			} else {
				glog.Errorf("transport: %+v", err)
			}
		}
	}()
}

// NewKCPTransport returns a new KCP protocol plugin
func NewKCPTransport(address string, opts ...KCPOption) (TransportInterface, error) {
	if len(address) > 5 && address[:6] != "kcp://" {
		return nil, errors.New("transport: address must begin with kcp://")
	}

	p := &Transport{
		address: address,
	}

	kcp := &KCPPlugin{
		p,
		defaultKCPOptions,
	}
	for _, o := range opts {
		o(&kcp.opts)
	}

	return kcp, nil
}

package base

import (
	"net"
	"sync"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

const MaxPublicVisibleAddressCandidates = 8

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type Dialer func(address string) (net.Conn, error)

type ConnectionAdapter struct {
	listener    net.Listener
	Dialer      Dialer
	idToAddress sync.Map

	reportedPubliclyVisibleAddresses      []*PubliclyVisibleAddress
	reportedPubliclyVisibleAddressesMutex sync.Mutex
}

type PubliclyVisibleAddress struct {
	address string
	count   uint64
}

func NewConnectionAdapter(listener net.Listener, dialer Dialer, node *protocol.Node) (*ConnectionAdapter, error) {
	adapter := &ConnectionAdapter{
		listener: listener,
		Dialer:   dialer,
	}
	node.SetConnectionAdapter(adapter)
	return adapter, nil
}

func (a *ConnectionAdapter) Dial(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	remoteAddr, err := a.lookupAddressByID(remote)
	if err != nil {
		return nil, err
	}

	conn, err := a.Dialer(remoteAddr)
	if err != nil {
		return nil, err
	}

	msgAdapter, err := NewMessageAdapterActive(a, conn, local, remote, a.getPubliclyVisibleAddress(), remoteAddr)
	if err != nil {
		return nil, err
	}
	a.updatePubliclyVisibleAddress(msgAdapter.Metadata()["localAddr"])
	return msgAdapter, nil
}

func (a *ConnectionAdapter) Accept(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	ch := make(chan protocol.MessageAdapter)
	go func() {
		defer close(ch)
		for {
			select {
			case <-c.Cancellation:
				return
			default:
			}

			conn, err := a.listener.Accept() // TODO: timeout
			if err != nil {
				log.Error().Err(err).Msg("unable to accept connection")
				continue
			}

			adapter, err := NewMessageAdapterPassive(a, conn, local, a.getPubliclyVisibleAddress())
			if err != nil {
				log.Error().Err(err).Msg("unable to start message adapter")
				continue
			}
			a.updatePubliclyVisibleAddress(adapter.Metadata()["localAddr"])

			ch <- adapter
		}
	}()
	return ch
}

func (a *ConnectionAdapter) getPubliclyVisibleAddress() string {
	a.reportedPubliclyVisibleAddressesMutex.Lock()
	var ret string
	if len(a.reportedPubliclyVisibleAddresses) > 0 {
		ret = a.reportedPubliclyVisibleAddresses[0].address
	}
	a.reportedPubliclyVisibleAddressesMutex.Unlock()
	return ret
}

// updatePubliclyVisibleAddress used to maintain the node's public address in case it's using UPnP
func (a *ConnectionAdapter) updatePubliclyVisibleAddress(address string) {
	a.reportedPubliclyVisibleAddressesMutex.Lock()
	defer a.reportedPubliclyVisibleAddressesMutex.Unlock()

	for i, pva := range a.reportedPubliclyVisibleAddresses {
		if pva.address == address {
			pva.count++
			p := i - 1
			// bubble up
			for p >= 0 {
				if a.reportedPubliclyVisibleAddresses[p].count < a.reportedPubliclyVisibleAddresses[p+1].count {
					t := a.reportedPubliclyVisibleAddresses[p]
					a.reportedPubliclyVisibleAddresses[p] = a.reportedPubliclyVisibleAddresses[p+1]
					a.reportedPubliclyVisibleAddresses[p+1] = t
					p--
				} else {
					break
				}
			}
			return
		}
	}

	if len(a.reportedPubliclyVisibleAddresses) > MaxPublicVisibleAddressCandidates-1 {
		a.reportedPubliclyVisibleAddresses = a.reportedPubliclyVisibleAddresses[:MaxPublicVisibleAddressCandidates-1]
	}

	// always keep the last received publicly visible address in storage
	// so it will always have a chance of being preferred.
	a.reportedPubliclyVisibleAddresses = append(a.reportedPubliclyVisibleAddresses, &PubliclyVisibleAddress{
		address: address,
		count:   1,
	})
}

func (a *ConnectionAdapter) AddPeerID(id []byte, addr string) error {
	a.idToAddress.Store(string(id), addr)
	return nil
}

func (a *ConnectionAdapter) GetPeerIDs() [][]byte {
	var results [][]byte
	a.idToAddress.Range(func(key, value interface{}) bool {
		results = append(results, ([]byte)(key.(string)))
		return true
	})
	return results
}

func (a *ConnectionAdapter) lookupAddressByID(id []byte) (string, error) {
	if v, ok := a.idToAddress.Load(string(id)); ok {
		return v.(string), nil
	} else {
		return "", errors.New("not found")
	}
}

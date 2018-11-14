package skademlia

import (
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/protocol"
)

var _ protocol.ConnectionAdapter = (*SKademliaConnectionAdapter)(nil)

type SKademliaConnectionAdapter struct {
	connection.AddressableConnectionAdapter
	RoutingTable
}

/*
func (a *SKademliaConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	remoteAddr, err := a.lookupAddressByID(remote)
	if err != nil {
		return nil, err
	}

	conn, err := a.dialer(remoteAddr)
	if err != nil {
		return nil, err
	}

	return startAddressableMessageAdapter(a, conn, local, remote, remoteAddr, false)
}

func (a *SKademliaConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
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

			adapter, err := startAddressableMessageAdapter(a, conn, local, nil, "", true)
			if err != nil {
				log.Error().Err(err).Msg("unable to start message adapter")
				continue
			}

			ch <- adapter
		}
	}()
	return ch
}
*/

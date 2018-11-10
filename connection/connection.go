package connection

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
)

const MaxPublicVisibleAddressCandidates = 8

var _ protocol.ConnectionAdapter = (*AddressableConnectionAdapter)(nil)

type Dialer func(address string) (net.Conn, error)

type AddressableConnectionAdapter struct {
	listener    net.Listener
	dialer      Dialer
	idToAddress sync.Map

	reportedPubliclyVisibleAddresses      []*PubliclyVisibleAddress
	reportedPubliclyVisibleAddressesMutex sync.Mutex
}

type PubliclyVisibleAddress struct {
	address string
	count   uint64
}

type AddressableMessageAdapter struct {
	conn              net.Conn
	local             []byte
	remote            []byte
	finalizerNotifier chan struct{}
}

func StartAddressableConnectionAdapter(
	listener net.Listener,
	dialer Dialer,
) (*AddressableConnectionAdapter, error) {
	return &AddressableConnectionAdapter{
		listener: listener,
		dialer:   dialer,
	}, nil
}

func (a *AddressableConnectionAdapter) MapIDToAddress(id []byte, addr string) {
	a.idToAddress.Store(string(id), addr)
}

func (a *AddressableConnectionAdapter) lookupAddressByID(id []byte) (string, error) {
	if v, ok := a.idToAddress.Load(string(id)); ok {
		return v.(string), nil
	} else {
		return "", errors.New("not found")
	}
}

func (a *AddressableConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
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

func (a *AddressableConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
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

func (a *AddressableConnectionAdapter) getPubliclyVisibleAddress() string {
	a.reportedPubliclyVisibleAddressesMutex.Lock()
	var ret string
	if len(a.reportedPubliclyVisibleAddresses) > 0 {
		ret = a.reportedPubliclyVisibleAddresses[0].address
	}
	a.reportedPubliclyVisibleAddressesMutex.Unlock()
	return ret
}

func (a *AddressableConnectionAdapter) updatePubliclyVisibleAddress(address string) {
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

func startAddressableMessageAdapter(connAdapter *AddressableConnectionAdapter, conn net.Conn, local, remote []byte, remoteAddr string, passive bool) (*AddressableMessageAdapter, error) {
	if len(local) > 255 || len(remote) > 255 {
		return nil, errors.New("local or remote id too long")
	}
	byteBuf := make([]byte, 1)

	if passive {
		remote = make([]byte, len(local))

		_, err := io.ReadFull(conn, remote)
		if err != nil {
			conn.Close()
			return nil, err
		}

		_, err = conn.Write(local)
		if err != nil {
			conn.Close()
			return nil, err
		}

		_, err = io.ReadFull(conn, byteBuf)
		if err != nil {
			conn.Close()
			return nil, err
		}

		pvaLen := int(byteBuf[0])
		if pvaLen > 0 {
			pvaBytes := make([]byte, pvaLen)
			_, err = io.ReadFull(conn, pvaBytes)
			if err != nil {
				conn.Close()
				return nil, err
			}
			pva := string(pvaBytes)
			connAdapter.updatePubliclyVisibleAddress(pva)
			//log.Debug().Msgf("Current publicly visible address: %s", connAdapter.getPubliclyVisibleAddress())
		}
	} else {
		_, err := conn.Write(local)
		if err != nil {
			conn.Close()
			return nil, err
		}
		recvRemote := make([]byte, len(local))
		_, err = io.ReadFull(conn, recvRemote)
		if err != nil {
			conn.Close()
			return nil, err
		}
		if !bytes.Equal(recvRemote, remote) {
			conn.Close()
			return nil, errors.Errorf("inconsistent remotes %s and %s", hex.EncodeToString(recvRemote), hex.EncodeToString(remote))
		}

		if len(remoteAddr) > 255 {
			conn.Close()
			return nil, errors.Errorf("remote address is too long")
		}

		_, err = conn.Write(append([]byte{byte(len(remoteAddr))}, []byte(remoteAddr)...))
		if err != nil {
			conn.Close()
			return nil, err
		}
	}

	adapter := &AddressableMessageAdapter{
		conn:              conn,
		local:             local,
		remote:            remote,
		finalizerNotifier: make(chan struct{}),
	}

	return adapter, nil
}

func (a *AddressableMessageAdapter) Close() {
	a.conn.Close()
	close(a.finalizerNotifier)
}

func (a *AddressableMessageAdapter) RemoteEndpoint() []byte {
	return a.remote
}

func (a *AddressableMessageAdapter) SendMessage(c *protocol.Controller, message []byte) error {
	lenBuf := make([]byte, 16)
	n := binary.PutUvarint(lenBuf, uint64(len(message)))
	_, err := a.conn.Write(lenBuf[:n])
	if err != nil {
		return err
	}
	_, err = a.conn.Write(message)
	if err != nil {
		return err
	}
	return nil
}

func (a *AddressableMessageAdapter) StartRecvMessage(c *protocol.Controller, callback protocol.RecvMessageCallback) {
	go runRecvWorker(a.finalizerNotifier, a.conn, callback)
}

func runRecvWorker(finalizerNotifier chan struct{}, conn net.Conn, callback protocol.RecvMessageCallback) {
	reader := bufio.NewReader(conn)

	for {
		// conn should also be closed as soon as gcFinalize() is called
		// so we do not need to check finalizerNotifier?
		n, err := binary.ReadUvarint(reader)
		if err != nil {
			break
		}

		// not so accurate since the message header takes a few bytes;
		// but it works just fine here.
		if n > protocol.MaxPayloadLen {
			log.Error().Msg("message too long")
			break
		}

		buf := make([]byte, int(n))
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			break
		}

		callback(buf)
	}

	callback(nil)
}

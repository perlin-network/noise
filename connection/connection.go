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
	"time"
)

var _ protocol.ConnectionAdapter = (*TCPConnectionAdapter)(nil)

type TCPConnectionAdapter struct {
	listener    net.Listener
	listenAddr  string
	dialTimeout time.Duration
	idToAddress sync.Map
}

// No outside references to TCPMessageAdapter should be kept. Otherwise, a resource leak will happen
// because TCPMessageAdapter uses a finalizer to clean-up itself.
type TCPMessageAdapter struct {
	conn              net.Conn
	local             []byte
	remote            []byte
	finalizerNotifier chan struct{}
}

func StartTCPConnectionAdapter(
	listenAddr string,
	dialTimeout time.Duration,
) (*TCPConnectionAdapter, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}

	return &TCPConnectionAdapter{
		listener:    listener,
		listenAddr:  listenAddr,
		dialTimeout: dialTimeout,
	}, nil
}

func (a *TCPConnectionAdapter) MapIDToAddress(id []byte, addr string) {
	a.idToAddress.Store(string(id), addr)
}

func (a *TCPConnectionAdapter) lookupAddressByID(id []byte) (string, error) {
	if v, ok := a.idToAddress.Load(string(id)); ok {
		return v.(string), nil
	} else {
		return "", errors.New("not found")
	}
}

func (a *TCPConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	remoteAddr, err := a.lookupAddressByID(remote)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout("tcp", remoteAddr, a.dialTimeout)
	if err != nil {
		return nil, err
	}

	return startTCPMessageAdapter(conn, local, remote, false)
}

func (a *TCPConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
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

			adapter, err := startTCPMessageAdapter(conn, local, nil, true)
			if err != nil {
				log.Error().Err(err).Msg("unable to start message adapter")
				continue
			}

			ch <- adapter
		}
	}()
	return ch
}

func startTCPMessageAdapter(conn net.Conn, local, remote []byte, passive bool) (*TCPMessageAdapter, error) {
	if len(local) > 255 || len(remote) > 255 {
		return nil, errors.New("local or remote id too long")
	}

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
	}

	adapter := &TCPMessageAdapter{
		conn:              conn,
		local:             local,
		remote:            remote,
		finalizerNotifier: make(chan struct{}),
	}

	return adapter, nil
}

func (a *TCPMessageAdapter) Close() {
	a.conn.Close()
	close(a.finalizerNotifier)
}

func (a *TCPMessageAdapter) RemoteEndpoint() []byte {
	return a.remote
}

func (a *TCPMessageAdapter) SendMessage(c *protocol.Controller, message []byte) error {
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

func (a *TCPMessageAdapter) StartRecvMessage(c *protocol.Controller, callback protocol.RecvMessageCallback) {
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

		if n > 1048576 {
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

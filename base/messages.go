package base

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
)

var _ protocol.MessageAdapter = (*MessageAdapter)(nil)

type MessageAdapter struct {
	conn              net.Conn
	local             []byte
	remote            []byte
	finalizerNotifier chan struct{}
}

func NewMessageAdapter(connAdapter *ConnectionAdapter, conn net.Conn, local, remote []byte, remoteAddr string, passive bool) (*MessageAdapter, error) {
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

	adapter := &MessageAdapter{
		conn:              conn,
		local:             local,
		remote:            remote,
		finalizerNotifier: make(chan struct{}),
	}

	return adapter, nil
}

func (a *MessageAdapter) Close() {
	a.conn.Close()
	close(a.finalizerNotifier)
}

func (a *MessageAdapter) RemoteEndpoint() []byte {
	return a.remote
}

func (a *MessageAdapter) SendMessage(c *protocol.Controller, message []byte) error {
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

func (a *MessageAdapter) StartRecvMessage(c *protocol.Controller, callback protocol.RecvMessageCallback) {
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

package main

import (
	"context"
	"encoding/hex"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"net"
	"os"
	"sync/atomic"
	"time"
)

const DialTimeout = 10 * time.Second

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, DialTimeout)
}

type CountService struct {
	protocol.Service
	MsgCount uint64
}

func (s *CountService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	atomic.AddUint64(&s.MsgCount, 1)
	return nil, nil
}

/*
  Usage:
	[terminal 1] go run main.go localhost:8000 wait
	[terminal 2] go run main.go localhost:8001 send (peerID from terminal 1) localhost:8000
*/
func main() {
	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		panic(err)
	}

	connAdapter, err := base.NewConnectionAdapter(listener, dialTCP)
	if err != nil {
		panic(err)
	}

	idAdapter := base.NewIdentityAdapter()
	kp := idAdapter.GetKeyPair()

	node := protocol.NewNode(
		protocol.NewController(),
		connAdapter,
		idAdapter,
	)
	node.Start()

	svc := &CountService{}

	node.AddService(svc)

	log.Info().Msgf("started, pubkey = %s", kp.PublicKeyHex())

	if os.Args[2] == "send" {
		peerID, err := hex.DecodeString(os.Args[3])
		if err != nil {
			panic(err)
		}
		remoteAddr := os.Args[4]
		connAdapter.AddPeerID(peerID, remoteAddr)

		go func() {
			for {
				node.Send(context.Background(), &protocol.Message{
					Sender:    kp.PublicKey,
					Recipient: peerID,
					Body: &protocol.MessageBody{
						Service: 42,
						Payload: []byte("Hello world!"),
					},
				})
				//node.ManuallyRemovePeer(peerID)
			}
		}()

	}

	for range time.Tick(10 * time.Second) {
		count := atomic.SwapUint64(&svc.MsgCount, 0)
		log.Info().Msgf("message count = %d", count)
	}

	select {}
}

package main

import (
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

	var msgCount uint64

	node.AddService(42, func(message *protocol.Message) {
		atomic.AddUint64(&msgCount, 1)
		//log.Info().Msgf("received payload from %s: %s", hex.EncodeToString(message.Sender), string(message.Body.Payload))
		/*node.Send(&protocol.Message {
			Sender: kp.PublicKey,
			Recipient: message.Sender,
			Body: message.Body,
		})*/
	})

	log.Info().Msgf("started, pubkey = %s", kp.PublicKeyHex())

	if os.Args[2] == "send" {
		peerID, err := hex.DecodeString(os.Args[3])
		if err != nil {
			panic(err)
		}
		remoteAddr := os.Args[4]
		connAdapter.MapIDToAddress(peerID, remoteAddr)

		go func() {
			for {
				node.Send(&protocol.Message{
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
		count := atomic.SwapUint64(&msgCount, 0)
		log.Info().Msgf("message count = %d", count)
	}

	select {}
}

package main

import (
	"encoding/hex"
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	connAdapter, err := connection.StartTCPConnectionAdapter(os.Args[1], 0)
	if err != nil {
		panic(err)
	}

	kp := ed25519.RandomKeyPair()
	idAdapter := identity.NewDefaultIdentityAdapter(kp)

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

		for {
			node.Send(&protocol.Message{
				Sender:    kp.PublicKey,
				Recipient: peerID,
				Body: &protocol.MessageBody{
					Service: 42,
					Payload: []byte("Hello world!"),
				},
			})
		}
	}

	for range time.Tick(10 * time.Second) {
		count := atomic.SwapUint64(&msgCount, 0)
		log.Info().Msgf("message count = %d", count)
	}

	select {}
}

package network

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type Server struct {
	network *Network
}

func createServer(network *Network) *Server {
	return &Server{
		network: network,
	}
}

// Handles new incoming peer connections and their messages.
func (s *Server) Stream(server protobuf.Noise_StreamServer) error {
	client := CreatePeerClient(s)
	defer client.close()

	go client.process()

	for {
		raw, err := server.Recv()

		// Should any errors occur reading packets, disconnect the peer.
		if err == io.EOF || err != nil {
			if client.id != nil {
				if s.network.Routes.PeerExists(*client.id) {
					s.network.Routes.RemovePeer(*client.id)
					log.Info("Peer " + client.id.Address + " has disconnected.")
				}
			}
			return nil
		}

		// Check if any of the message headers are invalid or null.
		if raw.Message == nil || raw.Sender == nil || raw.Sender.PublicKey == nil || len(raw.Sender.Address) == 0 || raw.Signature == nil {
			log.Debug("Received an invalid message (either no message, no sender, or no signature) from a peer.")
			continue
		}

		// Verify signature of message.
		if !crypto.Verify(raw.Sender.PublicKey, raw.Message.Value, raw.Signature) {
			continue
		}

		// Derive peer ID.
		val := peer.ID(*raw.Sender)

		// Just in case, set the peer ID only once.
		if client.id == nil {
			client.id = &val

			err := client.establishConnection()
			if err != nil {
				log.Debug(fmt.Sprintf("Failed to connect to peer %s err=[%+v]", client.id.Address, err))
				return err
			}
		} else if !client.id.Equals(val) {
			continue
		}

		// Unmarshal protobuf messages.
		var ptr ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(raw.Message, &ptr); err != nil {
			continue
		}

		msg := ptr.Message

		// Handle request/response.
		if raw.Nonce > 0 && raw.IsResponse {
			s.network.HandleResponse(raw.Nonce, msg)
		} else {
			// Forward it to mailbox of client.
			client.mailbox <- IncomingMessage{
				msg:   ptr.Message,
				nonce: raw.Nonce,
			}
		}
	}
	return nil
}

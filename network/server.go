package network

import (
	"io"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
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
	var client *PeerClient
	defer func() {
		if client != nil {
			client.close()
		}
	}()

	for {
		raw, err := server.Recv()

		// Should any errors occur reading packets, disconnect the peer.
		if err == io.EOF || err != nil {
			if client.Id != nil {
				if s.network.Routes.PeerExists(*client.Id) {
					s.network.Routes.RemovePeer(*client.Id)
					glog.Infof("Peer %s has disconnected.", client.Id.Address)
				}
			}
			break
		}

		// Check if any of the message headers are invalid or null.
		if raw.Message == nil || raw.Sender == nil || raw.Sender.PublicKey == nil || len(raw.Sender.Address) == 0 || raw.Signature == nil {
			glog.Info("Received an invalid message (either no message, no sender, or no signature) from a peer.")
			continue
		}

		// Verify signature of message.
		if !crypto.Verify(raw.Sender.PublicKey, raw.Message.Value, raw.Signature) {
			continue
		}

		// Derive peer ID.
		val := peer.ID(*raw.Sender)

		if client == nil {
			if cached, exists := s.network.Peers.Load(val.Address); exists && cached != nil {
				client = cached.(*PeerClient)
			} else {
				client = CreatePeerClient(s)
				s.network.Peers.Store(val.Address, client)
			}
		}

		// If peer ID has never been set, set it.
		if client.Id == nil {
			client.Id = &val

			err := client.establishConnection(client.Id.Address)

			// Could not connect to peer; disconnect.
			if err != nil {
				glog.Warningf("Failed to connect to peer %s err=[%+v]\n", client.Id.Address, err)
				return err
			}
		} else if !client.Id.Equals(val) {
			continue
		}

		// Update routing table w/ peer's ID.
		s.network.Routes.Update(val)

		// Unmarshal protobuf messages.
		var ptr ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(raw.Message, &ptr); err != nil {
			continue
		}

		msg := ptr.Message

		// Handle request/response.
		if raw.Nonce > 0 && raw.IsResponse {
			client.handleResponse(raw.Nonce, msg)
		} else {
			// Forward it to mailbox of Client.
			client.mailbox <- IncomingMessage{
				Message: ptr.Message,
				Nonce:   raw.Nonce,
			}
		}
	}
	return nil
}

package internal

import (
	"github.com/perlin-network/noise/actor"
	"github.com/perlin-network/noise/protobuf"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/dht"
)

type RouteActor struct {
	self peer.ID
	routes *dht.RoutingTable
}

// TODO: Send parameters which should contain easy-access to network.Network.
func (state *RouteActor) Receive(client protobuf.Noise_StreamClient, sender peer.ID, msg interface{}) {
	switch msg.(type) {
	case *protobuf.HandshakeRequest:
		// Update routing table w/ peer's ID.
		state.routes.Update(state.self)

		if client == nil {
			// Dial and send handshake response to peer.
			//client, err = state.network.Dial(raw.Sender.Address)
			//if err != nil {
			//	return
			//}
			//err = s.network.Tell(client, &protobuf.HandshakeResponse{})
			//if err != nil {
			//	return
			//}
			//
			//log.Info("Peer " + raw.Sender.Address + " has connected to you.")
		}
	case *protobuf.HandshakeResponse:
		// Update routing table w/ peer's ID.
		state.routes.Update(state.self)

		log.Info("Successfully bootstrapped w/ peer " + raw.Sender.Address + ".")
	case *protobuf.LookupNodeRequest:
		if client != nil {
			response := &protobuf.LookupNodeResponse{Peers: []*protobuf.ID{}}
			msg := msg.(*protobuf.LookupNodeRequest)

			// Update routing table w/ peer's ID.
			state.routes.Update(state.self)

			// Respond back with closest peers to a provided target.
			for _, id := range state.routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
				id := protobuf.ID(id)
				response.Peers = append(response.Peers, &id)
			}

			//s.network.Tell(client, response)
		}
	}
}

func CreateRouteActor(self peer.ID) func() actor.ActorTemplate {
	return func() actor.ActorTemplate {
		return &RouteActor{
			self: self,
			routes: dht.CreateRoutingTable(self),
		}
	}
}

package main

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"
)

const (
	serviceID    = 42
	host         = "localhost"
	numNodes     = 2
	sendingNodes = 1
)

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

type countService struct {
	protocol.Service
	MsgCount uint64
}

func (s *countService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != serviceID {
		return nil, nil
	}
	atomic.AddUint64(&s.MsgCount, 1)
	return nil, nil
}

func main() {
	var services []*countService
	var nodes []*protocol.Node
	var ports []int

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		// setup the node
		idAdapter := base.NewIdentityAdapter()
		node := protocol.NewNode(
			protocol.NewController(),
			idAdapter,
		)

		port := utils.GetRandomUnusedPort()
		ports = append(ports, port)
		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		// setup the connection adapter
		if _, err := base.NewConnectionAdapter(listener, dialTCP, node); err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		// create service
		service := &countService{}
		node.AddService(service)

		// Start listening for connections
		node.Start()

		nodes = append(nodes, node)
		services = append(services, service)
	}

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.GetIdentityAdapter().MyIdentity()
			srcNode.GetConnectionAdapter().AddRemoteID(peerID, fmt.Sprintf("%s:%d", host, ports[j]))
		}
	}

	// have every node send to the next one as quickly as possible
	for i := 0; i < sendingNodes; i++ {
		go func(senderIdx int) {
			receiver := nodes[(senderIdx+1)%numNodes].GetIdentityAdapter().MyIdentity()
			body := &protocol.MessageBody{
				Service: serviceID,
				Payload: []byte(fmt.Sprintf("From node %d to node %d", senderIdx, (senderIdx+1)%numNodes)),
			}
			for {
				nodes[senderIdx].Send(context.Background(), receiver, body)
			}
		}(i)
	}

	// dump the counts per second
	for range time.Tick(10 * time.Second) {
		var count uint64
		for _, svc := range services {
			atomic.AddUint64(&count, atomic.SwapUint64(&svc.MsgCount, 0))
		}
		log.Info().Msgf("message count = %d", count)
	}

	select {}
}

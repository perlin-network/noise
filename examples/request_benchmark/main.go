package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/utils"
)

const (
	opCode       = 42
	host         = "localhost"
	numNodes     = 2
	sendingNodes = 1
)

type countService struct {
	*noise.Noise
	MsgCount uint64
}

func (s *countService) Receive(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
	if message.Body.Service != opCode {
		return nil, nil
	}
	atomic.AddUint64(&s.MsgCount, 1)
	return nil, nil
}

func main() {
	var services []*countService

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		// setup the node
		config := &noise.Config{
			Host:            host,
			Port:            utils.GetRandomUnusedPort(),
			EnableSKademlia: false,
		}
		n, err := noise.NewNoise(config)
		if err != nil {
			panic(err)
		}

		// create service
		service := &countService{
			Noise: n,
		}
		service.OnReceive(noise.OpCode(opCode), service.Receive)

		services = append(services, service)
	}

	// Connect all the node routing tables
	for i, svc := range services {
		var peerIDs []noise.PeerID
		for j, other := range services {
			if i == j {
				continue
			}
			peerIDs = append(peerIDs, other.Self())
		}
		svc.Bootstrap(peerIDs...)
	}

	// have every node send to the next one as quickly as possible
	for i := 0; i < sendingNodes; i++ {
		go func(senderIdx int) {
			receiver := services[(senderIdx+1)%numNodes].Self().PublicKey
			body := &noise.MessageBody{
				Service: opCode,
				Payload: []byte(fmt.Sprintf("From node %d to node %d", senderIdx, (senderIdx+1)%numNodes)),
			}
			for {
				services[senderIdx].Send(context.Background(), receiver, body)
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

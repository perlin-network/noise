package main

import (
	"fmt"
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"math/rand"
	"sync/atomic"
	"time"
)

const NumInstances = 3
const StartPort = 7000

type Instance struct {
	address      string
	connAdapter  *connection.TCPConnectionAdapter
	node         *protocol.Node
	messageCount uint64
	keypair      *crypto.KeyPair
}

func StartInstance(port uint16) *Instance {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	connAdapter, err := connection.StartTCPConnectionAdapter(addr, 0)
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

	inst := &Instance{
		address:     addr,
		connAdapter: connAdapter,
		node:        node,
		keypair:     kp,
	}

	node.AddService(42, func(message *protocol.Message) {
		atomic.AddUint64(&inst.messageCount, 1)
	})

	node.Start()

	return inst
}

func (inst *Instance) ReadMessageCount() uint64 {
	return atomic.LoadUint64(&inst.messageCount)
}

func main() {
	instances := make([]*Instance, NumInstances)
	for i := 0; i < NumInstances; i++ {
		instances[i] = StartInstance(StartPort + uint16(i))
	}
	for i := 0; i < NumInstances; i++ {
		for j := 0; j < NumInstances; j++ {
			instances[i].connAdapter.MapIDToAddress(instances[j].keypair.PublicKey, instances[j].address)
		}
	}

	for i := 0; i < NumInstances; i++ {
		i := i
		go func() {
			current := instances[i]
			for {
				selectedN := rand.Intn(len(instances))
				if selectedN == i {
					continue
				}
				selected := instances[selectedN].keypair.PublicKey

				for {
					err := current.node.Send(&protocol.Message{
						Sender:    current.keypair.PublicKey,
						Recipient: selected,
						Body: &protocol.MessageBody{
							Service: 42,
							Payload: []byte("Hello world!"),
						},
					})
					if err == nil {
						break
					}
					time.Sleep(5 * time.Microsecond)
				}

				// simulate unstable connection
				if rand.Intn(1000) == 0 {
					current.node.ManuallyRemovePeer(selected)
				}
			}
		}()
	}

	lastMsgCount := make([]uint64, NumInstances)
	for range time.Tick(1 * time.Second) {
		newMsgCount := make([]uint64, NumInstances)
		for i := 0; i < NumInstances; i++ {
			newMsgCount[i] = instances[i].ReadMessageCount()
		}
		info := ""
		for i := 0; i < NumInstances; i++ {
			info += fmt.Sprintf("%d ", newMsgCount[i]-lastMsgCount[i])
		}
		log.Info().Msg(info)
		lastMsgCount = newMsgCount
	}
}

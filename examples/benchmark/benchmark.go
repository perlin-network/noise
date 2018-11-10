package main

import (
	"bytes"
	"fmt"
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"sync/atomic"
	"time"
)

const NumInstances = 20
const StartPort = 7000
const DialTimeout = 10 * time.Second

type Instance struct {
	address      string
	connAdapter  *connection.AddressableConnectionAdapter
	node         *protocol.Node
	messageCount uint64
	keypair      *crypto.KeyPair
}

type SimpleHandshakeProcessor struct{}

type SimpleHandshakeState struct {
	passive bool
}

/*
	ActivelyInitHandshake() ([]byte, interface{}, error) // (message, state, err)
	PassivelyInitHandshake() (interface{}, error) // (state, err)
	ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, DoneAction, error) // (message, doneAction, err)
*/

func (*SimpleHandshakeProcessor) ActivelyInitHandshake() ([]byte, interface{}, error) {
	return []byte("init"), &SimpleHandshakeState{passive: false}, nil
}

func (*SimpleHandshakeProcessor) PassivelyInitHandshake() (interface{}, error) {
	return &SimpleHandshakeState{passive: true}, nil
}

func (*SimpleHandshakeProcessor) ProcessHandshakeMessage(_state interface{}, payload []byte) ([]byte, protocol.DoneAction, error) {
	state := _state.(*SimpleHandshakeState)
	if state.passive {
		if bytes.Equal(payload, []byte("init")) {
			return []byte("ack"), protocol.DoneAction_SendMessage, nil
		} else {
			return nil, protocol.DoneAction_Invalid, errors.New("invalid handshake (passive)")
		}
	} else {
		if bytes.Equal(payload, []byte("ack")) {
			return nil, protocol.DoneAction_DoNothing, nil
		} else {
			return nil, protocol.DoneAction_Invalid, errors.New("invalid handshake (active)")
		}
	}
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, DialTimeout)
}

func StartInstance(port uint16) *Instance {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	connAdapter, err := connection.StartAddressableConnectionAdapter(listener, dialTCP)
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
	node.SetCustomHandshakeProcessor((*SimpleHandshakeProcessor)(nil))

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
	dropRate := uint32(10000) // 1/10000

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
				if rand.Intn(int(atomic.LoadUint32(&dropRate))) == 0 {
					current.node.ManuallyRemovePeer(selected)
				}
			}
		}()
	}

	lastMsgCount := make([]uint64, NumInstances)
	periodSecs := 5

	for range time.Tick(time.Duration(periodSecs) * time.Second) {
		newMsgCount := make([]uint64, NumInstances)
		for i := 0; i < NumInstances; i++ {
			newMsgCount[i] = instances[i].ReadMessageCount()
		}
		info := fmt.Sprintf("Drop rate=1/%d\t", atomic.LoadUint32(&dropRate))
		sum := uint64(0)
		for i := 0; i < NumInstances; i++ {
			sum += newMsgCount[i] - lastMsgCount[i]
		}
		info += fmt.Sprintf("Messages per second=%f\t", float64(sum)/float64(periodSecs))
		/*for i := 0; i < NumInstances; i++ {
			info += fmt.Sprintf("%d\t", newMsgCount[i]-lastMsgCount[i])
		}*/
		log.Info().Msg(info)
		lastMsgCount = newMsgCount
		if atomic.LoadUint32(&dropRate) < 10 {
			break
		}
		atomic.StoreUint32(&dropRate, atomic.LoadUint32(&dropRate)*3/4)
	}
}

package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"sync/atomic"
	"time"
)

const (
	NumInstances = 20
	StartPort    = 7000
	DialTimeout  = 10 * time.Second
)

var (
	_ protocol.HandshakeProcessor = (*SimpleHandshakeProcessor)(nil)
)

type Instance struct {
	protocol.Service
	address      string
	connAdapter  protocol.ConnectionAdapter
	node         *protocol.Node
	messageCount uint64
	keypair      *crypto.KeyPair
}

type SimpleHandshakeProcessor struct{}

type SimpleHandshakeState struct {
	passive bool
}

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

func StartInstance(port int) *Instance {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	idAdapter := base.NewIdentityAdapter()

	node := protocol.NewNode(
		protocol.NewController(),
		idAdapter,
	)
	node.SetCustomHandshakeProcessor((*SimpleHandshakeProcessor)(nil))

	connAdapter, err := base.NewConnectionAdapter(listener, dialTCP, node)
	if err != nil {
		panic(err)
	}

	inst := &Instance{
		address:     addr,
		connAdapter: connAdapter,
		node:        node,
		keypair:     idAdapter.GetKeyPair(),
	}

	node.AddService(inst)

	node.Start()

	return inst
}

func (s *Instance) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	atomic.AddUint64(&s.messageCount, 1)
	return nil, nil
}

func (inst *Instance) ReadMessageCount() uint64 {
	return atomic.LoadUint64(&inst.messageCount)
}

func main() {
	instances := make([]*Instance, NumInstances)
	for i := 0; i < NumInstances; i++ {
		instances[i] = StartInstance(StartPort + i)
	}
	for i := 0; i < NumInstances; i++ {
		for j := 0; j < NumInstances; j++ {
			instances[i].connAdapter.AddPeerID(instances[j].keypair.PublicKey, instances[j].address)
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
					err := current.node.Send(context.Background(),
						selected,
						&protocol.MessageBody{
							Service: 42,
							Payload: []byte("Hello world!"),
						})
					if err == nil {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
		}()
	}

	lastMsgCount := make([]uint64, NumInstances)
	periodSecs := 10
	dropRate := 10000 // 1/10000

	for range time.Tick(time.Duration(periodSecs) * time.Second) {
		newMsgCount := make([]uint64, NumInstances)
		for i := 0; i < NumInstances; i++ {
			newMsgCount[i] = instances[i].ReadMessageCount()
		}
		info := fmt.Sprintf("Drop rate=1/%d\t", dropRate)
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
		if dropRate < 10 {
			// program is done
			break
		}
		dropRate = dropRate * 3 / 4

		for i := 0; i < NumInstances; i++ {
			for j := 0; j < NumInstances; j++ {
				if j == i {
					continue
				}
				// simulate unstable connection
				if rand.Intn(dropRate) == 0 {
					instances[i].node.ManuallyRemovePeer(instances[j].keypair.PublicKey)
					instances[j].node.ManuallyRemovePeer(instances[i].keypair.PublicKey)
				}
			}
		}
	}
}

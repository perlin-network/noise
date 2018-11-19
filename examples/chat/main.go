package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/base/discovery"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	chatServiceID            = 44
	requestResponseServiceID = 45
)

var (
	reqNonce    = uint64(1)
	reqResponse sync.Map
)

type ChatNode struct {
	Node        *protocol.Node
	Address     string
	ConnAdapter protocol.ConnectionAdapter
}

func (n *ChatNode) ReceiveHandler(message *protocol.Message) {
	if message.Body.Service != chatServiceID {
		return
	}
	if len(message.Body.Payload) == 0 {
		return
	}
	var msg messages.ChatMessage
	if err := proto.Unmarshal(message.Body.Payload, &msg); err != nil {
		return
	}
	log.Info().Msgf("<%s> %s", n.Address, msg.Message)
}

func makeMessageBody(serviceID int, msg *protobuf.Message) *protocol.MessageBody {
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}
	body := &protocol.MessageBody{
		Service: uint16(serviceID),
		Payload: payload,
	}
	return body
}

type ChatRequestAdapter struct {
	Node *protocol.Node
}

func (c *ChatRequestAdapter) Request(ctx context.Context, target peer.ID, body *protobuf.Message) (*protobuf.Message, error) {
	body.RequestNonce = atomic.AddUint64(&reqNonce, 1)
	msg := &protocol.Message{
		Sender:    c.Node.GetIdentityAdapter().MyIdentity(),
		Recipient: target.PublicKey,
		Body:      makeMessageBody(requestResponseServiceID, body),
	}
	replyChan := make(chan *protobuf.Message, 1)
	reqResponse.Store(body.RequestNonce, replyChan)
	if err := c.Node.Send(msg); err != nil {
		return nil, err
	}
	select {
	case reply := <-replyChan:
		return reply, nil
	case <-time.After(3 * time.Second):
		return nil, errors.New("Timed out")
	}
	return nil, errors.New("Unexpected")
}

func (c *ChatRequestAdapter) Reply(ctx context.Context, target peer.ID, body *protobuf.Message) error {
	body.ReplyFlag = true
	msg := &protocol.Message{
		Sender:    c.Node.GetIdentityAdapter().MyIdentity(),
		Recipient: target.PublicKey,
		Body:      makeMessageBody(requestResponseServiceID, body),
	}
	return c.Node.Send(msg)
}

func main() {
	// process other flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	peersFlag := flag.String("peers", "", "peers to connect to in format: peerKeyHash1=host1:port1,peerKeyHash2=host2:port2,...")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")

	idAdapter := base.NewIdentityAdapter()

	log.Info().Msgf("Private Key: %s", idAdapter.GetKeyPair().PrivateKeyHex())
	log.Info().Msgf("Public Key: %s", idAdapter.GetKeyPair().PublicKeyHex())

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	connAdapter, err := base.NewConnectionAdapter(listener, func(addr string) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 10*time.Second)
	})
	if err != nil {
		panic(err)
	}

	node := &ChatNode{
		Node: protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		),
		Address:     addr,
		ConnAdapter: connAdapter,
	}

	node.Node.AddService(chatServiceID, node.ReceiveHandler)

	discoveryService := discovery.NewService(
		&ChatRequestAdapter{Node: node.Node},
		peer.CreateID(addr, idAdapter.GetKeyPair().PublicKey),
	)

	node.Node.AddService(discovery.DiscoveryServiceID, discoveryService.ReceiveHandler)

	if len(peers) > 0 {
		for _, peerKV := range peers {
			if len(peerKV) == 0 {
				// this is a blank parameter
				continue
			}
			p := strings.Split(peerKV, "=")
			peerID, err := hex.DecodeString(p[0])
			if err != nil {
				panic(err)
			}
			remoteAddr := p[1]
			connAdapter.AddPeerID(peer.CreateID(remoteAddr, peerID))
		}
	}

	node.Node.Start()

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')

		// skip blank lines
		if len(strings.TrimSpace(input)) == 0 {
			continue
		}

		log.Info().Msgf("<%s> %s", addr, input)

		chatMsg := &messages.ChatMessage{
			Message: input,
		}
		bytes, _ := chatMsg.Marshal()
		msg := &protobuf.Message{
			Message: bytes,
		}
		node.Node.Broadcast(makeMessageBody(chatServiceID, msg))
	}
}

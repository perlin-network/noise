package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	chatServiceID = 44
)

var (
	reqNonce    = uint64(1)
	reqResponse sync.Map
)

type ChatService struct {
	protocol.Service
	Address string
}

func (n *ChatService) Receive(ctx context.Context, request *protocol.Message) (*protocol.MessageBody, error) {
	if request.Body.Service != chatServiceID {
		return nil, nil
	}
	if len(request.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	var pm protobuf.Message
	if err := proto.Unmarshal(request.Body.Payload, &pm); err != nil {
		return nil, err
	}
	var mc messages.ChatMessage
	if err := proto.Unmarshal(pm.Message, &mc); err != nil {
		return nil, err
	}
	log.Info().Msgf("<%s> %s", n.Address, mc.Message)
	return nil, nil
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

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
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

	node := protocol.NewNode(
		protocol.NewController(),
		idAdapter,
	)

	if _, err := base.NewConnectionAdapter(listener, dialTCP, node); err != nil {
		panic(err)
	}

	service := &ChatService{
		Address: addr,
	}

	node.AddService(service)

	node.Start()

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
			node.GetConnectionAdapter().AddPeerID(peerID, remoteAddr)
		}
	}

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
		node.Broadcast(context.Background(), makeMessageBody(chatServiceID, msg))
	}
}

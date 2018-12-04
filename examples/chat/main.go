package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/dht"
	"github.com/perlin-network/noise/skademlia/peer"
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
	var mc messages.ChatMessage
	if err := proto.Unmarshal(request.Body.Payload, &mc); err != nil {
		return nil, err
	}
	log.Info().Msgf("<%s> %s", n.Address, mc.Message)
	return nil, nil
}

func makeMessageBody(serviceID int, msg *messages.ChatMessage) *protocol.MessageBody {
	payload, err := msg.Marshal()
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
	peersFlag := flag.String("peers", "", "peers to connect to in format: publicKey1=host1:port1,publicKey2=host2:port2,...")
	privateKeyFlag := flag.String("private_key", "", "use an existing public key generated from this private key parameter")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")
	privateKeyHex := *privateKeyFlag

	var idAdapter *skademlia.IdentityAdapter
	if len(privateKeyHex) > 0 {
		kp, err := crypto.FromPrivateKey(ed25519.New(), privateKeyHex)
		if err != nil {
			panic(err)
		}
		idAdapter, err = skademlia.NewIdentityFromKeypair(kp, skademlia.DefaultC1, skademlia.DefaultC2)
		if err != nil {
			panic(err)
		}
	} else {
		idAdapter = skademlia.NewIdentityAdapterDefault()
	}

	log.Info().Msgf("Private Key: %s", idAdapter.GetKeyPair().PrivateKeyHex())
	log.Info().Msgf("Public Key: %s", idAdapter.MyIdentityHex())

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	node := protocol.NewNode(
		protocol.NewController(),
		idAdapter,
	)

	if _, err := skademlia.NewConnectionAdapter(listener, dialTCP, node, addr); err != nil {
		panic(err)
	}

	service := &ChatService{
		Address: addr,
	}

	node.AddService(service)

	node.Start()

	if len(peers) > 0 {
		var peerIDs []peer.ID
		for _, peerKV := range peers {
			if len(peerKV) == 0 {
				// this is a blank parameter
				continue
			}
			p := strings.Split(peerKV, "=")
			peerPubKey, err := hex.DecodeString(p[0])
			if err != nil {
				panic(err)
			}
			remoteAddr := p[1]
			peerIDs = append(peerIDs, dht.NewID(peerPubKey, remoteAddr))
		}

		node.GetConnectionAdapter().(*skademlia.ConnectionAdapter).Bootstrap(peerIDs...)
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
		node.Broadcast(context.Background(), makeMessageBody(chatServiceID, chatMsg))
	}
}

package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"flag"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/examples/chat/messages"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	chatOpCode = 44
)

var (
	reqNonce    = uint64(1)
	reqResponse sync.Map
)

// ChatService inherits the noise interface
type ChatService struct {
	*noise.Noise
}

func receive(ctx context.Context, request *noise.Message) (*noise.MessageBody, error) {
	if request.Body.Service != chatOpCode {
		return nil, nil
	}
	if len(request.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	var mc messages.ChatMessage
	if err := proto.Unmarshal(request.Body.Payload, &mc); err != nil {
		return nil, err
	}
	log.Info().Msgf("<%s> %s", hex.EncodeToString(request.Sender)[0:16], mc.Message)
	return nil, nil
}

// makeMessageBody is a helper to serialize the message type
func makeMessageBody(serviceID int, msg *messages.ChatMessage) *noise.MessageBody {
	payload, err := msg.Marshal()
	if err != nil {
		return nil
	}
	body := &noise.MessageBody{
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
	peersFlag := flag.String("peers", "", "peers to connect to in format: NodeID=host:port (required if not the first node)")
	privateKeyFlag := flag.String("private_key", "", "use an existing public key generated from this private key parameter (optional)")
	flag.Parse()

	port := *portFlag
	host := *hostFlag
	peers := strings.Split(*peersFlag, ",")
	privateKeyHex := *privateKeyFlag

	// setup the node
	config := &noise.Config{
		Host:            host,
		Port:            port,
		EnableSKademlia: true,
	}

	if len(privateKeyHex) > 0 {
		config.PrivateKeyHex = privateKeyHex
	}

	// setup the node
	n, err := noise.NewNoise(config)
	if err != nil {
		panic(err)
	}
	svc := &ChatService{
		Noise: n,
	}

	// print the identity so you can use the public key for the next node
	log.Info().Msgf("PrivateKey: %s", svc.Config().PrivateKeyHex)
	log.Info().Msgf("NodeID: %s", hex.EncodeToString(svc.Self().PublicKey))

	// register the recieve callback
	svc.OnReceive(noise.OpCode(chatOpCode), receive)

	if len(peers) > 0 {
		// bootstrap the node to an existing cluster
		var peerIDs []noise.PeerID
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
			peerIDs = append(peerIDs, noise.CreatePeerID(peerPubKey, remoteAddr))
		}

		svc.Bootstrap(peerIDs...)
	}

	// broadcast any stdin inputs
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')

		// skip blank lines
		if len(strings.TrimSpace(input)) == 0 {
			continue
		}

		log.Info().Msgf("<%s> %s", hex.EncodeToString(svc.Self().PublicKey)[0:16], input)

		body := makeMessageBody(chatOpCode, &messages.ChatMessage{
			Message: input,
		})
		svc.Broadcast(context.Background(), body)
	}
}

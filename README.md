# Noise

[![GoDoc][1]][2] [![Discord][7]][8] [![MIT licensed][5]][6] [![Build Status][9]][10] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
[5]: https://img.shields.io/badge/license-MIT-blue.svg
[6]: LICENSE
[7]: https://img.shields.io/discord/458332417909063682.svg
[8]: https://discord.gg/dMYfDPM
[9]: https://travis-ci.org/perlin-network/noise.svg?branch=master
[10]: https://travis-ci.org/perlin-network/noise
[11]: https://goreportcard.com/badge/github.com/perlin-network/noise
[12]: https://goreportcard.com/report/github.com/perlin-network/noise
[13]: https://codecov.io/gh/perlin-network/noise/branch/master/graph/badge.svg
[14]: https://codecov.io/gh/perlin-network/noise


<img align="right" width=400 src="media/chat.gif">

**noise** is an opinionated, easy-to-use P2P network stack for
*decentralized applications, and cryptographic protocols* written in
[Go](https://golang.org/) by Perlin Network.

**noise** is made to be robust, developer-friendly, performant, secure, and
cross-platform across multitudes of devices by making use of well-tested,
production-grade dependencies.

## Features

- Modular design with interfaces for establishing connections, verifying identity/authorization, and sending/receiving messages.
- Real-time, bidirectional streaming between peers via TCP and
  [Protobufs](https://developers.google.com/protocol-buffers/).
- NAT traversal/automated port forwarding (NAT-PMP, UPnP).
- [NaCL/Ed25519](https://tweetnacl.cr.yp.to/) scheme for peer identities and
  signatures.
- [S/Kademlia](https://ieeexplore.ieee.org/document/4447808) peer identity and discovery.
- Request/Response and Messaging RPC.
- Logging via [zerolog](https://github.com/rs/zerolog/log).
- Plugin system via services.

## Setup

### Dependencies

 - [Protobuf compiler](https://github.com/google/protobuf/releases) (protoc)
 - [Go 1.11](https://golang.org/dl/) (go)

```bash
# enable go modules: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# download the dependencies to vendor folder for tools
go mod vendor

# generate necessary code files
go get -u github.com/gogo/protobuf/protoc-gen-gogofaster
# tested with version v1.1.1 (636bf0302bc95575d69441b25a2603156ffdddf1)
go get -u github.com/golang/mock/mockgen
# tested with v1.1.1 (c34cdb4725f4c3844d095133c6e40e448b86589b)
go generate ./...

# run an example
[terminal 1] go run examples/chat/main.go -port 3000 -private_key aefd86bdfefe4e2eca563782682d7612a856191b48844687fec1c8a22dc70f220da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1
[terminal 2] go run examples/chat/main.go -port 3001 -peers 0da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1=localhost:3000
[terminal 3] go run examples/chat/main.go -port 3002 -peers 0da80160d6b3686d66a4ad8ac692a322043b0239302c5037988d4bb1e41830f1=localhost:3000

# run test cases
go test -v -count=1 -race ./...

# run test cases short
go test -v -count=1 -race -short ./...
```

## Usage

Noise is designed to be modular and splits the networking, node identity, and message sending/receiving into separate interfaces.

Noise provides out-of-the-box node configuration. A new Noise configuration randomly generates public/private keys using the Ed25519 signature scheme.

```go
// Create a new Noise config which generate a new Ed25519 keypair.
config := &noise.Config{
    PrivateKeyHex: idAdapter.GetKeyPair().PrivateKeyHex(),
}
n, _ := noise.NewNoise(config)

// Create a Noise config from an existing hex-encoded private key.
config := &noise.Config{
    PrivateKeyHex: "4d5333a68e3a96d0ad935cb6546b97bbb0c0771acf76c868a897f65dad0b7933e1442970cce57b7a35e1803e0e8acceb04dc6abf8a73df52e808ab5d966113ac",
}
n, _ := noise.NewNoise(config)

// Print out loaded public/private keys.
log.Info().
    Str("private_key", n.Config().PrivateKeyHex()).
    Msg("")
log.Info().
    Str("public_key", hex.EncodeToString(n.Self().PublicKey)).
    Msg("")
log.Info().
    Str("node_id", hex.EncodeToString(n.Self().Id)).
    Msg("")
```

To change the host and port the node listens to, set the `Host` and `Port` settings in the Noise config.

```go
// setup the node with a different host and port
config := &noise.Config{
    Host: "localhost",
    Port: 3000,
}
n, _ := noise.NewNoise(config)

// add services for the node here.
svc := &BasicService{
    Noise, n,
}

// Register callbacks
svc.OnReceive(svc.OpCode, svc.Receive)

// Connect or bootstrap to other nodes
svc.Bootstrap(peers...)
```

See `examples/getting_started` for a full working example to get started with.

## Services

Services are a way to interface with the lifecycle of your network.


```go
type YourAwesomeService struct {
	*noise.Noise
	Mailbox chan *messages.BasicMessage
}

func (state *YourAwesomeService) Startup(node *noise.Node)              {}
func (state *YourAwesomeService) Receive(ctx context.Context, request *noise.Message) (*noise.MessageBody, error)  { return nil }
func (state *YourAwesomeService) Cleanup(node *noise.Node)              {}
func (state *YourAwesomeService) PeerConnect(id []byte)    {}
func (state *YourAwesomeService) PeerDisconnect(id []byte) {}
```

Services reference a noise node as follows and register callbacks:

```go
n, err := noise.NewNoise(&noise.Config{})

// Create service from noise node.
service := &BasicService{
    Noise: n,
    Mailbox: make(chan *messages.BasicMessage, 1),
}

// register the service callback
service.OnReceive(service.OpCode, func(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
	return nil, nil
})
```

## S/Kademlia

Noise natively supports the [S/Kademlia]((https://ieeexplore.ieee.org/document/4447808)) protocol for peer identification and routing. Enable S/Kademlia in your application by setting the Noise config.

```go
n, err := noise.NewNoise(&noise.Config{
    EnableSKademlia: true,
})
```

## Handling Messages

Messages in **noise** are passed as bytes. This means, you can serialize/deserialize your messages in whichever way you wish. In our examples, we use messages that are serialized/deserialized as
[protobufs](https://developers.google.com/protocol-buffers/).

Here's a simple example:

```go
// helper for converting strings to messages
func makeMessageBody(value string) *noise.MessageBody {
    msg := &messages.BasicMessage{
        Message: value,
    }
    payload, err := proto.Marshal(msg)
    if err != nil {
        return nil
    }
    body := &noise.MessageBody{
        Service: opCode,
        Payload: payload,
    }
    return body
}

...

// broadcast your message to all connected peers
service.Broadcast(context.Background(), makeMessageBody("hello world"))
```

Check out our documentation and look into the `examples/` directory to find out
more.

## Contributions

We at Perlin love reaching out to the open-source community and are open to
accepting issues and pull-requests.

For all code contributions, please ensure they adhere as close as possible to
the following guidelines:

1. **Strictly** follows the formatting and styling rules denoted
   [here](https://github.com/golang/go/wiki/CodeReviewComments).
2. Commit messages are in the format `module_name: Change typed down as a sentence.`
   This allows our maintainers and everyone else to know what specific code
   changes you wish to address.
    - `protocol: Made Noise interface simpler.`
    - `examples/basic: Fixed test to conform to new code.`
3. Consider backwards compatibility.

If you...

1. love the work we are doing,
2. want to work full-time with us,
3. or are interested in getting paid for working on open-source projects

... **we're hiring**.

To grab our attention, just make a PR and start contributing.

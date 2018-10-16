# Noise

[![GoDoc][1]][2] [![Discord][7]][8] [![Powered][3]][4] [![MIT licensed][5]][6] [![Build Status][9]][10] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
[3]: https://img.shields.io/badge/KCP-Powered-blue.svg
[4]: https://github.com/skywind3000/kcp
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

- Real-time, bidirectional streaming between peers via
  [KCP](https://github.com/xtaci/kcp-go)/TCP and
  [Protobufs](https://developers.google.com/protocol-buffers/).
- NAT traversal/automated port forwarding (NAT-PMP, UPnP).
- [NaCL/Ed25519](https://tweetnacl.cr.yp.to/) scheme for peer identities and
  signatures.
- Kademlia DHT-inspired peer discovery.
- Request/Response and Messaging RPC.
- Logging via [zerolog](https://github.com/rs/zerolog/log).
- Plugin system.

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
[terminal 1] vgo run examples/chat/main.go -port 3000
[terminal 2] vgo run examples/chat/main.go -port 3001 -peers tcp://localhost:3000
[terminal 3] vgo run examples/chat/main.go -port 3002 -peers tcp://localhost:3000

# run test cases
vgo test -v -count=1 -race ./...

# run test cases short
vgo test -v -count=1 -race -short ./...
```

## Usage

A peer's cryptographic public/private keys are randomly generated/loaded with
1 LoC in mind.

```go
// Randomly generate a Ed25519 keypair.
keys := ed25519.RandomKeyPair()

// Load a private key through a hex-encoded string.
keys := crypto.FromPrivateKey(ed25519.New(), "4d5333a68e3a96d0ad935cb6546b97bbb0c0771acf76c868a897f65dad0b7933e1442970cce57b7a35e1803e0e8acceb04dc6abf8a73df52e808ab5d966113ac")

// Load a private key through a provided 64-length byte array (for Ed25519 keypair).
keys := crypto.FromPrivateKeyBytes(ed25519.New(), []byte{ ...}...)

// Print out loaded public/private keys.
log.Info().
    Str("private_key", keys.PrivateKeyHex()).
    Msg("")
log.Info().
    Str("public_key", keys.PublicKeyHex()).
    Msg("")
```

You may use the loaded keys to sign/verify messages that are loaded as byte
arrays.

```go
msg := []byte{ ... }

// Sign a message.
signature, err := keys.Sign(ed25519.New(), blake2b.New(), msg)
if err != nil {
    panic(err)
}

log.Info().
    Str("signature", hex.EncodeToString(signature)).
    Msg("")

// Verify a signature.
verified := crypto.Verify(ed25519.New(), blake2b.New(), keys.PublicKey, msg, signature)

log.Info().Msgf("Is the signature valid? %b", verified)
```

Now that you have your keys, we can start listening and handling messages from
incoming peers.

```go
builder := network.NewBuilder()

// Set the address which noise will listen on and peers will use to connect to
// you. For example, set the host part to `localhost` if you are testing
// locally, or your public IP address if you are connected to the internet
// directly.
builder.SetAddress("tcp://localhost:3000")

// Alternatively...
builder.SetAddress(network.FormatAddress("tcp", "localhost", 3000))

// Set the cryptographic keys used for your network.
builder.SetKeys(keys)

// ... add plugins or set any more options here.

// Build the network.
net, err := builder.Build()
if err != nil {
    panic(err)
}

// Have the server start listening for peers.
go net.Listen()

// Connect to some peers and form a peer cluster automatically with built-in
// peer discovery.
net.Bootstrap("tcp://localhost:3000", "tcp://localhost:3001")

// Alternatively..
net.Bootstrap([]string{"tcp://localhost:3000", "tcp://localhost:3001"}...)
```

If you have any code you want to execute only once the node is ready to listen
for peers, just run:

```go
net.BlockUntilListening()
```

... in any goroutine you desire. The goroutine will block until the server is
ready to start listening.

See `examples/getting_started` for a full working example to get started with.

## Plugins

Plugins are a way to interface with the lifecycle of your network.


```go
type YourAwesomePlugin struct {
    network.Plugin
}

func (state *YourAwesomePlugin) Startup(net *network.Network)              {}
func (state *YourAwesomePlugin) Receive(ctx *network.PluginContext) error  { return nil }
func (state *YourAwesomePlugin) Cleanup(net *network.Network)              {}
func (state *YourAwesomePlugin) PeerConnect(client *network.PeerClient)    {}
func (state *YourAwesomePlugin) PeerDisconnect(client *network.PeerClient) {}
```

They are registered through `network.Builder` through the following:

```go
builder := network.NewBuilder()

// Add plugin.
builder.AddPlugin(new(YourAwesomePlugin))
```

**noise** comes with three plugins: `discovery.Plugin`, `backoff.Plugin` and
`nat.Plugin`.

```go
// Enables peer discovery through the network.
// Check documentation for more info.
builder.AddPlugin(new(discovery.Plugin))

// Enables exponential backoff upon peer disconnection.
// Check documentation for more info.
builder.AddPlugin(new(backoff.Plugin))

// Enables automated NAT traversal/port forwarding for your node.
// Check documentation for more info.
nat.RegisterPlugin(builder)
```

Make sure to register `discovery.Plugin` if you want to make use of automatic
peer discovery within your application.

## Handling Messages

All messages that pass through **noise** are serialized/deserialized as
[protobufs](https://developers.google.com/protocol-buffers/). If you want to
use a new message type for your application, you must first register your
message type.

```go
opcode.RegisterMessageType(opcode.Opcode(1000), &MyNewProtobufMessage{})
```

On a spawned `us-east1-b` Google Cloud (GCP) cluster comprised of 8
`n1-standard-1` (1 vCPU, 3.75GB memory) instances, **noise** is able to sign,
send, receive, verify, and process a total of ~10,000 messages per second.

Once you have modeled your messages as protobufs, you may process and receive
them over the network by creating a plugin and overriding the
`Receive(ctx *PluginContext)` method to process specific incoming message types.

Here's a simple example:

```go
// An example chat plugin that will print out a formatted chat message.
type ChatPlugin struct{ *network.Plugin }

func (state *ChatPlugin) Receive(ctx *network.PluginContext) error {
    switch msg := ctx.Message().(type) {
        case *messages.ChatMessage:
            log.Info().Msgf("<%s> %s", ctx.Client().ID.Address, msg.Message)
    }
    return nil
}

// Register plugin to *network.Builder.
builder.AddPlugin(new(ChatPlugin))
```

Through a `ctx *network.PluginContext`, you can access flexible methods to
customize how you handle/interact with your peer network. All messages are
signed and verified with one's cryptographic keys.

```go
// Reply with a message should the incoming message be a request.
err := ctx.Reply(message here)
if err != nil {
    return err
}

// Get an instance of your own nodes ID.
self := ctx.Self()

// Get an instance of the peers ID which sent the message.
sender := ctx.Sender()

// Get access to an instance of the peers client.
client := ctx.Client()

// Get access to your own nodes network instace.
net := ctx.Network()
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
    - `network: Added in message broadcasting methods.`
    - `builders/network: Added in new option to address PoW in generating peer IDs.`
3. Consider backwards compatibility. New methods are perfectly fine, though
   changing the `network.Builder` pattern radically for example should only be
   done should there be a good reason.

If you...

1. love the work we are doing,
2. want to work full-time with us,
3. or are interested in getting paid for working on open-source projects

... **we're hiring**.

To grab our attention, just make a PR and start contributing.

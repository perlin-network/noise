# Nodes

Nodes are self-explanatory: they are the central hub of any p2p application that listens for peers, accepts incoming connections from peers, and coordinates around alive peer connections.

You can create multiple nodes within a single application, though most of the time you would most probably just stick to creating one.

To make management of nodes simple within Noise, all node-level logic/operations are encapsulated and accessible under a single entity: `*noise.Node`.

```go
import "github.com/Yayg/noise"

func main() {
	// Instantiate a default set of node parameters.
	params := noise.DefaultParams()
	params.Port = uint16(3000)
	
	// Instantiate a new node that listens for peers on port 3000.
	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	
	// Start listening for incoming peers.
	go node.Listen()
	
	select{}
}
```

Instantiating a `*noise.Node` is a matter of passing in a set of configuration options, with which we provide to you a nice set of defaults.

```go
func DefaultParams() parameters {
	return parameters{
		Host:           "127.0.0.1",
		Port: 0
		Transport:      transport.NewTCP(),
		Metadata:       map[string]interface{}{},
		MaxMessageSize: 1048576,

		SendMessageTimeout:    3 * time.Second,
		ReceiveMessageTimeout: 3 * time.Second,

		SendWorkerBusyTimeout: 3 * time.Second,
	}
}
```

## Setup

Before instantiating a node however, you may optionally change up the configuration options however you want. Noise provides a large array of options you may play around with.

```go
type parameters struct {
	Host               string
	Port, ExternalPort uint16

	NAT       nat.Provider
	Keys      identity.Keypair
	Transport transport.Layer

	Metadata map[string]interface{}

	MaxMessageSize uint64

	SendMessageTimeout    time.Duration
	ReceiveMessageTimeout time.Duration

	SendWorkerBusyTimeout time.Duration
}
```

As an example, you may forcefully set `Host` to have your node listen for peers under a specified host address.

Or you may choose to set the internal port your node listens on by setting `Port`, or otherwise set an external port which you expect your computer to be able to openly accept connections on by setting `ExternalPort`.

By default, should the `ExternalPort` option not be set, `ExternalPort` will be set to whatever value `Port` is on node instantiation.

Should `Port` not be specified, depending on the transport layer chosen (at the very least, for TCP), a randomly available port will be assigned to the node upon instantiation.

The next couple of pages goes over how to configure a couple of the available options above.

## Metadata

Every single `*noise.Node` instance is capable of holding custom metadata that you choose to pass around throughout your p2p application.

The exposed functions for manipulating custom metadata is akin to the `sync.Map` API in Go's standard library.

Should you share any metadata across multiple goroutines, be sure to use some sync primitives to safely share data around.

```go
// Given a `*noise.Node` instance `node`,
var node *noise.Node

someValue := node.LoadOrStore("some key", uint32(64))
// `someValue` is an interface{} type that may be safely casted to uint32

queriedValue := node.Get("some key")
// `queriedValue` is an interface{} type that may be safely casted to uint32

fmt.Println("Must be true:", someValue == queriedValue)

node.Set("some key", true)

fmt.Println("The value for key `some key` is now:", node.Get("some key"))

node.Delete("some key")

fmt.Println("The value for key `some key` is now nil:", node.Get("some key"))
```

## Cleanup

After you are done with a node, you may gracefully stop a node by invoking the `node.Kill()` function.

The function will block the current goroutine until all workers related to a node have been put to a complete stop.

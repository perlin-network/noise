# Peers

Nodes in **noise** come with the ability to be able to dial and connect to other peers.

After taking some time to configure and instantiate your node, you can dial/connect to a peer like so:

```go
package main

import (
	"github.com/Yayg/noise"
)

func main() {
	params := noise.DefaultParams()
	params.Port = uint16(3000)
	
	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}
	
	// Start listening for incoming peers.
	go node.Listen()
	
	// Dial peer at address 127.0.0.1:3001.
	peer, err := node.Dial("127.0.0.1:3001")
	if err != nil {
		panic("failed to dial peer located at 127.0.0.1:3001!")
	}
	
	// ... do whatever you want with `peer` here.
	
	select{}
}
```

The node function `Dial(addres string)` returns an instance of `*noise.Peer` which provides you a large number of convenience functions for interacting/dealing with a peer.

An error will return should there be issues connecting/dialing a peer.

> **Note:** You may have multiple `*noise.Peer` instances connected to the exact same computer/address. They are unique amongst one another, and simply represent but a single connection instance to a computer.

The next page will go over briefly how to send/receive message given a `*noise.Peer` instance.

## Intercepting Events

Every single time a peer connects to your node, or a connection is successfully established against a peer, you may pass to a `*noise.Node` instance a callback function to execute.

```go
var node *noise.Node

node.OnPeerConnected(func(node *noise.Node, peer *noise.Peer) error {
	// This function gets called every time a peer connects
	// to you!
	return nil
})

node.OnPeerDialed(func(node *noise.Node, peer *noise.Peer) error {
	// This function gets called every time you successfully
	// connect to a peer!
	return nil
})

node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
	// This function gets called every time a peer connects
	// to you, or you successfully connect to a peer!
	return nil
})
```

Just like every other callback function passed to Noise, be sure that no blocking (ex: infinite loop) occurs in any callback function or else Noise will deadlock.

Some reasons for why you would want to intercept any of these events include:

1. instantiating metadata for your peer,
2. spawning new worker goroutines that run in parallel with every single peer,
3. and sending/receiving messages upon successful instantiation of a `*noise.Peer`.

## Metadata

Every single `*noise.Peer` instance is capable of holding custom metadata that you choose to pass around throughout your p2p application.

The exposed functions for manipulating custom metadata is akin to the `sync.Map` API in Go's standard library.

Should you share any metadata across multiple goroutines, be sure to use some sync primitives to safely share data around.

```go
// Given a `*noise.Peer` instance `peer`,
var peer *noise.Peer

someValue := peer.LoadOrStore("some key", uint32(64))
// `someValue` is an interface{} type that may be safely casted to uint32

queriedValue := peer.Get("some key")
// `queriedValue` is an interface{} type that may be safely casted to uint32

fmt.Println("Must be true:", someValue == queriedValue)

peer.Set("some key", true)

fmt.Println("The value for key `some key` is now:", peer.Get("some key"))

peer.Delete("some key")

fmt.Println("The value for key `some key` is now nil:", peer.Get("some key"))
```

## Cleanup

After you are done with a peer, you may gracefully stop the peer by invoking the `peer.Disconnect()` function.

The function will block the current goroutine until all workers related to a peer have been put to a complete stop.

You may additionally specify a callback function to be called when a specified peer disconnects to cleanup any resources/variables left behind of a peer.

```go
var peer *noise.Peer

peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
	// handle disconnect logic here...
	return nil
})

// We can also have a disconnect callback registered on every single peer.
var node *noise.Node

node.OnPeerDisconnected(func(node *noise.Node, peer *noise.Peer) error {
	// handle disconnect logic here...
	return nil
})

// The above is just syntactical sugar for the following code.
node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
	peer.OnDisconnect(func(node *noise.Node, peer *noise.peer) error {
		// handle disconnect logic here..
		return nil
	})
	
	return nil
})
```


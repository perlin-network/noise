# I/O

After having established a connection with a peer and registering a couple of `noise.Message` types, you are now ready to start sending and receiving some messages via Noise.

Noise was designed with the intent of making message I/O be as simple as possible. More specifically, writing message sending/receiving code should _look_ and _feel_ synchronous when in reality Noise handles all of the asynchronous/concurrent work for you.

For a bit of background, from the very moment a peer is successfully dialed, or a peer connects to your node, two goroutines are spawned.

A send worker is spawned responsible for linearizing and send messages over the wire. 

A receive worker is spawned responsible for deciphering the contents of raw network packets into `noise.Opcode` and `noise.Message` instances that may be intercepted.

## Sending a message

In order to send a message, you would instantiate a message you wish to send over the wire and use either of the following methods:

```go
var peer *noise.Peer
var msg noise.Message

// Send a message, and block the current goroutine until the message is successfully sent.
err := peer.SendMessage(msg)
if err != nil {
	panic("failed to send message")
}

// Send a message and create a channel which may be read for broadcasting errors at a later time.
err := <-peer.SendMessageAsync(msg)
if err != nil {
	panic("failed to send message")
}
```

`SendMessage(noise.Message) error` sends a message whose type is registered with Noise to a specified peer. Calling this function will block the current goroutine until the message is successfully sent.

On the other hand, using `SendMessageAsync(noise.Message) <-chan error` will not block the current goroutine, and instead simply queues the message up and lets you read from the channel for broadcasting errors anytime throughout your p2p application.

Both methods above are guaranteed to block until your message to be sent gets successfully queued into the send worker.

It is guaranteed that by sending message through either of the means above, all messages are sent in a linearized order.

All functions will return errors if:

1. it takes too long to send a message,
2. the message requested to be sent is not actually registered to Noise,
3. or the send worker responsible for linearizing the order of sent messages is too busy for too long of a period of time.

It is guaranteed that `SendMessageAsync` will at most only emit one error; you may choose to directly `close()` the channel after receiving a single message should you rather not have Go's garbage collector manually close the channel for you.

To figure out how to adjust the timeouts which define how long is 'too long' as to when a message is considered to have failed to be delivered, check out the `Messages` section in `Nodes`.

## Receiving a message

In order to receive a message, you would specify the opcode of the message you expect to receive from a given peer like so:

```go
import "time"

var peer *noise.Peer
var someMessagesOpcode noise.Opcode

// Receive an expected message, but panic should we be waiting for the message for
// too long.
select {
    case expectedMessage := <-peer.Receive(someMessagesOpcode):
    	fmt.Println("Got the message we were looking for:", expectedMessage)
    case <-time.After(3 * time.Second):
    	panic("We waited for 3 seconds and still didn't get the message we wanted :(")
}
```

`Receive(noise.Opcode) <-chan noise.Message` was intentionally designed to return a channel that may be read from multiple times to allow for complete flexibility on how messages should be received from a peer.

You may timeout the receiving of a message should we be waiting for a message from a peer for too long, or create an infinite loop acting as an event loop that expects, receives, and processes multiple messages designated by their opcodes at once.

```go
import "github.com/Yayg/noise"
import "time"

var peer *noise.Peer
var opcode1, opcode2, opcode3 noise.Opcode

// An example of an infinite loop that awaits and handles messages from a designated
// peer.
func receiveWorkerLoop(peer *noise.Peer, kill <-chan struct{}) {
    for {
        select {
        case <-kill:
        	return
        case msg := <-peer.Receive(opcode1):
            // handle opcode 1 here...
        case msg := <-peer.Receive(opcode2):
            // handle opcode 2 here...
        case msg := <-peer.Receive(opcode3):
            // handle opcode 3 here...
        default:
            // maybe do something else if we did not receive a message?
        }
    }
}

func main() {
	var node *noise.Node
	
	// ...
	// ... setup node here
	// ...
	
	// Have the receive worker loop run in a new goroutine on every single
	// newly connected/dialed peer.
	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		kill := make(chan struct{})
		go receiveWorkerLoop(peer, kill)
		
		// Maybe after 5 seconds, kill the receive worker loop?
	    go func() {
	    	<-time.After(5 * time.Seconds)
	    	
	    	close(kill)
	    }()
		
		return nil
	})
}
```

## Atomic Operations

One important feature Noise provides is being able to perform atomic operations over the network upon the recipient of a message.

Examples of such atomic operations include blocking the recipient of any messages from a designated peer except for a specific message.
 
This is is extremely useful for establishing synchronization points/acknowledgements between peers that from both sides, which allow us to know that they both completed some sort of action such as completing some handshake protocol scheme at the same time.

```go
import "time"

var peer *noise.Peer
var opcodeACK noise.Opcode

type ACK struct { noise.Message }

noise.RegisterMessage(noise.NextAvailableOpcode(), (*ACK)(nil))

// Send an ACK message to establish a synchronization point.
err := peer.SendMessage(ACK{})
if err != nil {
	panic("failed to send ACK to peer")
}

// Upon the recipient of an ACK message, block the recipient of any
// other messages from this peer until `Unlock()` gets called.
locker := peer.LockOnReceive(b.opcodeACK)
defer locker.Unlock()

select {
	case <-time.After(3 * time.Second):
		return errors.Wrap(protocol.DisconnectPeer, "timed out waiting for AEAD ACK")
	case <-peer.Receive(b.opcodeACK):
}

// We have entered the 'critical section' of our protocol that is safely executed
// by our node and our peer. We can do stuff like setup encryption/decryption
// schemes where we encrypt every message we send, and decrypt every message
// we receive from now on.
//
// It is guaranteed that within this critical section, we will NOT receive
// or handle any other messages.

peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
	// ... do decryption here.
	return msg, nil
})

peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
	// ... do encryption here.
	return msg, nil
})

// Do whatever else you want here until `locker.Unlock()` gets called.
```
# Messages

**noise** provides first-class support for choosing how you want message to be serialized/deserialized as they are sent/received over the network.

You can choose to cryptographically encrypt/decrypt every single message, append cryptographic signatures to every single message in its footer, or even compress/decompress every single message; the possibilities are endless.

## Setup

A message at the bare minimum comprises of two components: an opcode, and its contents.

The `opcode` is a single unsigned byte that designates what a message contents (its string of bytes) really mean to your application. Think of it like a static type system where you may have in total \\( 2^8 \\) kinds of types.

The `opcode <-> message` pairing may be specified entirely up to the user in code like so:

```go
import "github.com/Yayg/noise"

var _ noise.Message = (*RandomMessage)(nil)

type RandomMessage struct{}

func (RandomMessage) Read(reader payload.Reader) (noise.Message, error) {
	return RandomMessage{}, nil
}

func (RandomMessage) Write() []byte {
	return []byte("This is the message contents of a single message!")
}

// Register type `RandomMessage` to opcode 16.
noise.RegisterMessage(noise.Opcode(16), (*RandomMessage)(nil))
```

One thing that can be taken away by the code above is that messages in Noise implement the interface `noise.Message`.

```go
// To have Noise send/receive messages of a given type, said type must implement the
// following Message interface.
//
// Noise by default encodes messages as bytes in little-endian order, and provides
// utility classes to assist with serializing/deserializing arbitrary Go types into
// bytes efficiently.
//
// By exposing raw network packets as bytes to users, any additional form of
// serialization or message packing or compression scheme or cipher scheme may
// be bootstrapped on top of any particular message type registered to Noise.
type Message interface {
	Read(reader payload.Reader) (Message, error)
	Write() []byte
}
```

Another little tidbit that can be taken away by the code above is that you register said mappings using the `noise.Register(opcode noise.Opcode, messageType noise.Message)` function, with `messageType` in this case being a nil pointer of your message.

Should you attempt to send a message that is not registered, your application will panic as a safety precaution.

If a peer sends you a message that is not registered, said peer will be automatically disconnected away from you as a safety precaution.

If you believe something strange might be occuring with how opcodes are being mapped to message types, you can print them out easily by calling `noise.DebugOpcodes()` at any time.

## Next Available Opcode

For convenience, there exists a `noise.NextAvailableOpcode()` function that provides you an opcode that has yet to be registered to Noise.

You may use it like so:

```go
// Register type `RandomMessage` to the next available opcode.
noise.RegisterMessage(noise.NextAvailableOpcode(), (*RandomMessage)(nil))
```

For the most part by the way, it is highly recommend you only register opcodes upon initialization of your node.

Doing it while a node is live in your application can potentially cause strange sorts of data races, so it is highly recommended you don't play around with opcode to message type pairings in realtime!

## Message to Opcode Conversion

After registering your `opcode <-> message` pairing to Noise, you can use a couple of helper functions to derive the opcode of a given message type, and vice versa.

```go
import "fmt"

message := RandomMessage{}

// Get the opcode of your message.
opcode, err := noise.OpcodeFromMessage(message)
if err != nil {
	panic("message not registered to noise")
}

// The statement above and below are equivalent!
opcode, err = noise.OpcodeFromMessage((*RandomMessage)(nil))
if err != nil {
	panic("message not registered to noise")
}

// Print out the opcode of type `RandomMessage`.
fmt.Println("The opcode of `RandomMessage` is:", opcode)

// And vice-versa.
messageType, err := noise.MessageFromOpcode(opcode)
if err != nil {
	panic("message not registered to noise")
}

fmt.Println("The message type associated to opcode", opcode, "is:", messageType)
```

These mappings are what Noise relies on to be able to distinguish what kinds of message are being sent/received over the wire.

## Serialization/Deserialization

Up to 97% of the contents of every single network packet emitted by your node over the wire can be completely customized.

You could choose to represent all messages as byte-order little-endian messages, or even make use of `protobuf` or `msgpack` as your p2p applications message serialization/deserialization format.

The only limitation which Noise has in terms of configuring how messages look over the wire is that the `opcode` byte (_yes, a single unsigned 8-bit integer_), and your messages `content` (_a string of bytes_) have to be contiguously next to teach other.

To put it more visually, a message's payload is well-defined as:

### Message Format

```bash
[.. message header ..]
.
[.. START PAYLOAD SECTION ..]
[opcode (unsigned 8-bit integer)]
[contents ([]byte)]
[.. END PAYLOAD SECTION   ..]
.
[.. message footer ..]
```

Noise allows you to have complete control over the contents and encoding/parsing of a messages header/footer section, alongside the encoding/parsing of a messages content section.

Simply pass a callback function to your `*noise.Node` instance, which will be fed in with all of yur nodes incoming/outgoing data such that you may intercept all data feed accordingly.

You may intercept all incoming/outgoing messages on a per-peer level, such that you can choose to enforce encryption/decryption only for a particular set of peers!

```go
import "github.com/Yayg/noise"

func main() {
	params := noise.DefaultParams()
	
	node, err := noise.NewNode(params)
    if err != nil {
        panic(err)
    }
	
	// Register a callback every single time a peer is initialized.
	node.OnPeerInit(onInitPeer)
}

func onInitPeer(node *noise.Node, peer *noise.Peer) error {
    peer.BeforeMessageReceived(func (node *noise.Node, peer *Peer, incoming []byte) ([]byte, error) {
        // Every single time you receive a message from a specific peer,
        // this function will be called with the `contents` your peer
        // sent you!
        
        // You can setup some decryption that will take place
        // on `incoming` here.
        
        // Just leave the incoming data as it is, and return no error.
        // If an error occurs, a warning will be printed and the peer
        // will be disconnected.
        return incoming, nil
    })
    
	peer.BeforeMessageSent(func (node *noise.Node, peer *Peer, outgoing []byte) ([]byte, error) {
		// Every single time you send a message to a specific peer,
		// this function will be called with the `contents` of your
		// message to be sent!
		
		// You can setup some encryption that will take place
		// on `outgoing` here.
		
		// Just leave the outgoing data as it is, and return no error.
		// If an error occurs, a warning will be printed and the peer
		// will be disconnected.
		return outgoing, nil
	}
	
	// Some additional functions to be aware of that lets you handle
	// the encoding/parsing of the header/footer section of all messages!
	peer.OnEncodeHeader(...)
	peer.OnEncodeFooter(...)
	peer.OnDecodeHeader(...)
	peer.OnDecodeFooter(...)
	
	return nil
}
```

Note that everything performed inside any of the callback functions should not be blocking (i.e. infinite loops), otherwise Noise will deadlock as all callbacks are called synchronously.

This overall serves as a nice teaser on the nature of how one can hook/intercept specific networking-related operations via Noise.

For more information regarding the different kinds of network operations that may be intercepted, check out the `Peer` section.

### Wire Format

Over the wire, a message is then prefixed with an unsigned 64-bit variable-sized integer denoting the length of the message as follows:

```bash
[length of message as unsigned 64-bit variable-sized integer]
.
.
[.. message header ..]
.
[.. START PAYLOAD SECTION ..]
[opcode (unsigned 8-bit integer)]
[contents ([]byte)]
[.. END PAYLOAD SECTION   ..]
.
[.. message footer ..]
```

To prevent payload buffer over-run attacks, a configuration option is provided on instantiating a node to set the max message size `MaxMessageSize`.

## Serialization/Deserialization

Noise emphasizes on performance, and thus by default does not require developers to have to make use of a message serialization/deserialization scheme such as `protobuf` or `msgpack` from the get-go.

The `payload` package in Noise provides tools to easily assist developers to precisely define how Noise-registered messages are serialized into bytes, and thereafter deserialized into message instances.

In particular, the tools provided in the `payload` package by default allows developers to quickly convert statically typed data into little-endian ordered bytes.

Reading/writing raw little-endian bytes is simple, quick and efficient, and gives developers peace-of-mind knowing that they have complete control over every single byte of their message before it is transmitted over the wire.

### `payload` package

By default, the `payload` package prefixes all strings and byte arrays with their respective lengths.

Booleans are represented as single bytes, and all integers are little-endian.

Noise instantiates `payload.Reader` and `payload.Writer` instances every single time a message is received/sent respectively.

You may choose to omit having to use the `payload` package at any time, and directly plug-n-play `protobuf` or `msgpack` to specify how your message types are serialized/deserialized should you desire.

```go
// Here's an example on how to manually specify how the message type `TestMessage`
// is formatted in terms of little-endian ordered bytes.

import (
	"github.com/Yayg/noise"
	"github.com/Yayg/noise/payload"
)

var _ noise.Message = (*TestMessage)(nil)

type TestMessage struct {
	someBytes []byte
	someString string
	someByte byte
	someUint16 uint16
	someUint32 uint32
	someUint64 uint64
}

// It's best you manually handle and return errors inside each respective
// Read()/Write() function! This is just for demo code cleanup purposes.
func check(err error) {
	if err != nil {
		panic(err)
    }
}

func (t *TestMessage) Read(reader payload.Reader) (noise.Message, error) {
	var err error
	
	t.someBytes, err = reader.ReadBytes()
	check(err)
	
	t.someString, err = reader.ReadString()
	check(err)
	
	t.someByte, err = reader.ReadByte()
	check(err)
	
	t.someUint16, err = reader.ReadUint16()
	check(err)
	
	t.someUint32, err = reader.ReadUint32()
	check(err)
	
	t.someUint64, err = reader.ReadUint64()
	check(err)
	
	return t, nil
}

func (t *TestMessage) Write() []byte {
	return payload.NewWriter(nil).
		WriteBytes(t.someBytes).
		WriteString(t.someString).
		WriteByte(t.someByte).
		WriteUint16(t.someUint16).
		WriteUint32(t.someUint32).
		WriteUint64(t.someUint64).
		Bytes()
}

// Register the message type to Noise.
noise.RegisterMessage(noise.NextAvailableOpcode(), (*TestMessage)(nil))
```

```go
// Let's assume there exists a protobuf-defined message called `ProtobufMessage`.
//
// We can have Noise recognize it as a noise.Message by implementing
// both Read() and Write() functions for `ProtobufMessage`!

import "github.com/golang/protobuf/proto"

var _ noise.Message = (*ProtobufMessage)(nil)

// It's best you manually handle and return errors inside each respective
// Read()/Write() function! This is just for demo code cleanup purposes.
func check(err error) {
	if err != nil {
		panic(err)
    }
}

func (t *ProtobufMessage) Read(reader payload.Reader) (noise.Message, error) {
	bytes, err := reader.ReadBytes()
	check(err)
	
	err = proto.Unmarshal(bytes, t)
	check(err)
	
	return t, nil
}

func (t *ProtobufMessage) Write() []byte {
    bytes, err := proto.Marshal(t)
    check(err)
    
	return payload.NewWriter(nil).
		WriteBytes(bytes).
		Bytes()
}

// Register the message type to Noise.
noise.RegisterMessage(noise.NextAvailableOpcode(), (*ProtobufMessage)(nil))
```

## Timeouts

One bit that is always good to have complete control over is the ability to enforce timeouts for fundamental networking operations.

For the time being (_open to suggestions!_), timeouts may be set for:

1. retrieving and processing of a single message (`ReceiveMessageTimeout`),
2. on attempting to send a message to a peer (`SendMessageTimeout`),
3. and on waiting for the send queue worker to be available (`SendWorkerBusyTimeout`).
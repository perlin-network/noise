package noise_test

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise"
	"strings"
	"sync"
)

// ChatMessage is an example struct that is registered on example nodes, and serialized/deserialized on-the-fly.
type ChatMessage struct {
	content string
}

// Marshal serializes a chat message into bytes.
func (m ChatMessage) Marshal() []byte {
	return []byte(m.content)
}

// Unmarshal deserializes a slice of bytes into a chat message, and returns an error should deserialization
// fail, or the slice of bytes be malformed.
func UnmarshalChatMessage(buf []byte) (ChatMessage, error) {
	return ChatMessage{content: strings.ToValidUTF8(string(buf), "")}, nil
}

// This example demonstrates messaging with registering Go types to be serialized/deserialized on-the-wire provided
// marshal/unmarshal functions, how to decode serialized messages received from a peer, and how to send serialized
// messages.
func Example_codecMessaging() {
	// Let there be Alice and Bob.

	alice, err := noise.NewNode()
	if err != nil {
		panic(err)
	}

	bob, err := noise.NewNode()
	if err != nil {
		panic(err)
	}

	// Gracefully release resources for Alice and Bob at the end of the example.

	defer alice.Close()
	defer bob.Close()

	// Register the ChatMessage type to Alice and Bob so they know how to serialize/deserialize
	// them.

	alice.RegisterMessage(ChatMessage{}, UnmarshalChatMessage)
	bob.RegisterMessage(ChatMessage{}, UnmarshalChatMessage)

	var wg sync.WaitGroup

	// When Alice gets a ChatMessage from Bob, print it out.

	alice.Handle(func(ctx noise.HandlerContext) error {
		obj, err := ctx.DecodeMessage()
		if err != nil {
			return nil
		}

		msg, ok := obj.(ChatMessage)
		if !ok {
			return nil
		}

		fmt.Printf("Got a message from Bob: '%s'\n", msg.content)

		wg.Done()

		return nil
	})

	// When Bob gets a message from Alice, print it out.

	bob.Handle(func(ctx noise.HandlerContext) error {
		obj, err := ctx.DecodeMessage()
		if err != nil {
			return nil
		}

		msg, ok := obj.(ChatMessage)
		if !ok {
			return nil
		}

		fmt.Printf("Got a message from Alice: '%s'\n", msg.content)

		wg.Done()

		return nil
	})

	// Have Alice and Bob start listening for new peers.

	if err := alice.Listen(); err != nil {
		panic(err)
	}

	if err := bob.Listen(); err != nil {
		panic(err)
	}

	// Have Alice send Bob a ChatMessage with 'Hi Bob!'

	if err := alice.SendMessage(context.TODO(), bob.Addr(), ChatMessage{content: "Hi Bob!"}); err != nil {
		panic(err)
	}

	// Wait until Bob receives the message from Alice.

	wg.Add(1)
	wg.Wait()

	// Have Bob send Alice a ChatMessage with 'Hi Alice!'

	if err := bob.SendMessage(context.TODO(), alice.Addr(), ChatMessage{content: "Hi Alice!"}); err != nil {
		panic(err)
	}

	// Wait until Alice receives the message from Bob.

	wg.Add(1)
	wg.Wait()

	// Output:
	// Got a message from Alice: 'Hi Bob!'
	// Got a message from Bob: 'Hi Alice!'
}

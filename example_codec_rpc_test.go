package noise_test

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise"
)

// This example demonstrates sending registered serialized Go types as requests, decoding registered serialized
// Go types from peers, and sending registered serialized Go types as responses.
func Example_codecRPC() {
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

	// When Bob gets a request from Alice, print it out and respond to Alice with 'Hi Alice!'.

	bob.Handle(func(ctx noise.HandlerContext) error {
		if !ctx.IsRequest() {
			return nil
		}

		req, err := ctx.DecodeMessage()
		if err != nil {
			return nil
		}

		fmt.Printf("Got a message from Alice: '%s'\n", req.(ChatMessage).content)

		return ctx.SendMessage(ChatMessage{content: "Hi Alice!"})
	})

	// Have Alice and Bob start listening for new peers.

	if err := alice.Listen(); err != nil {
		panic(err)
	}

	if err := bob.Listen(); err != nil {
		panic(err)
	}

	// Have Alice send Bob a ChatMessage request with the message 'Hi Bob!'

	res, err := alice.RequestMessage(context.TODO(), bob.Addr(), ChatMessage{content: "Hi Bob!"})
	if err != nil {
		panic(err)
	}

	// Print out the ChatMessage response Bob got from Alice.

	fmt.Printf("Got a message from Bob: '%s'\n", res.(ChatMessage).content)

	// Output:
	// Got a message from Alice: 'Hi Bob!'
	// Got a message from Bob: 'Hi Alice!'
}

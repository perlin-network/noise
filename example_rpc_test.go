package noise_test

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise"
)

// This example demonstrates how to send/handle RPC requests across peers, how to listen for incoming peers, how
// to check if a message received is a request or not, how to reply to a RPC request, and how to cleanup node
// instances after you are done using them.
func Example_rPC() {
	// Let there be nodes Alice and Bob.

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

	// When Bob gets a message from Alice, print it out and respond to Alice with 'Hi Alice!'

	bob.Handle(func(ctx noise.HandlerContext) error {
		if !ctx.IsRequest() {
			return nil
		}

		fmt.Printf("Got a message from Alice: '%s'\n", string(ctx.Data()))

		return ctx.Send([]byte("Hi Alice!"))
	})

	// Have Alice and Bob start listening for new peers.

	if err := alice.Listen(); err != nil {
		panic(err)
	}

	if err := bob.Listen(); err != nil {
		panic(err)
	}

	// Have Alice send Bob a request with the message 'Hi Bob!'

	res, err := alice.Request(context.TODO(), bob.Addr(), []byte("Hi Bob!"))
	if err != nil {
		panic(err)
	}

	// Print out the response Bob got from Alice.

	fmt.Printf("Got a message from Bob: '%s'\n", string(res))

	// Output:
	// Got a message from Alice: 'Hi Bob!'
	// Got a message from Bob: 'Hi Alice!'
}

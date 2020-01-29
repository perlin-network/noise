package noise_test

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise"
	"sync"
)

// This example demonstrates how to send and receive raw bytes across peers, how to listen for incoming peers, and
// how to cleanup node instances after you are done using them.
func Example_messaging() {
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

	var wg sync.WaitGroup

	// When Alice gets a message from Bob, print it out.

	alice.Handle(func(ctx noise.HandlerContext) error {
		fmt.Printf("Got a message from Bob: '%s'\n", string(ctx.Data()))
		wg.Done()
		return nil
	})

	// When Bob gets a message from Alice, print it out.

	bob.Handle(func(ctx noise.HandlerContext) error {
		fmt.Printf("Got a message from Alice: '%s'\n", string(ctx.Data()))
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

	// Have Alice send Bob 'Hi Bob!'

	if err := alice.Send(context.TODO(), bob.Addr(), []byte("Hi Bob!")); err != nil {
		panic(err)
	}

	// Wait until Bob receives the message from Alice.

	wg.Add(1)
	wg.Wait()

	// Have Bob send Alice 'Hi Alice!'

	if err := bob.Send(context.TODO(), alice.Addr(), []byte("Hi Alice!")); err != nil {
		panic(err)
	}

	// Wait until Alice receives the message from Bob.

	wg.Add(1)
	wg.Wait()

	// Output:
	// Got a message from Alice: 'Hi Bob!'
	// Got a message from Bob: 'Hi Alice!'
}

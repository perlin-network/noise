package main

import (
	"fmt"
	"github.com/perlin-network/noise/xnoise"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Spawn 2 nodes: Alice and Bob.
	alice, err := xnoise.ListenTCP(0)
	check(err)

	bob, err := xnoise.ListenTCP(0)
	check(err)

	// Have Alice and Bob accept messages under opcodeTest.
	const opcodeTest byte = 0x01

	bob.Handle(opcodeTest, nil)

	// Have Alice dial Bob.
	aliceToBob, err := xnoise.DialTCP(alice, bob.Addr().String())
	check(err)

	// Wait until Bob successfully dials with Alice.
	bobsPeers := bob.Peers()
	for len(bobsPeers) == 0 {
		bobsPeers = bob.Peers()
	}

	// The only peer which Bob is connected to must be Alice.
	bobToAlice := bobsPeers[0]

	// Have Alice send a 'hello world!' to Bob under opcodeTest.
	check(aliceToBob.Send(opcodeTest, []byte("hello world!")))

	// Have Bob print out a single message from Alice that is under opcodeTest.
	fmt.Println("Alice said:", string((<-bobToAlice.Recv(opcodeTest))))
}

# Noise

[![GoDoc][1]][2] [![Discord][7]][8] [![MIT licensed][5]][6] [![Build Status][9]][10] [![Go Report Card][11]][12] [![Coverage Statusd][13]][14]

[1]: https://godoc.org/github.com/perlin-network/noise?status.svg
[2]: https://godoc.org/github.com/perlin-network/noise
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

**noise** is a secure, decentralized p2p extension to gRPC for 
[Go](https://golang.org/) by [the Perlin team](https://perlin.net)

**noise** extends gRPC to support overlay networks, perfect-forward-secrecy, encrypted channels, key establishment protocols.
All the necessary tools to build a robust and secure decentralized applications on top of gRPC.

By using gRPC, there are a few advantages right out the box:
1) High performance transport: HTTP/2 provides high speed communication protocol that can take advantage of bi-directional streaming, multiplexing and more.
2) Efficient serialization: Protobuf focuses on performance and simplicity.
3) Lesser boilerplate code: Using code generators, it reduces the amount of server and client code you have to write.

<hr/>

**noise** provides protocol building blocks such as:
1) handshake protocol implementations (Elliptic-Curve Diffie Hellman),
2) peer routing/discovery protocol implementations (S/Kademlia),
3) message broadcasting protocol implementations (S/Kademlia),
4) overlay network protocol implementations (S/Kademlia),
5) cryptographic identity schemes (Ed25519 w/ EdDSA signatures),
6) authenticated encryption schemes (AES-256 GCM AEAD).
7) and NAT traversal support (NAT-PMP, UPnP).

<br/>

```proto
// Payload and service definition
syntax = "proto3";

package main;

message Text {
    string message = 1;
}

service Chat {
    rpc Stream(stream Text) returns (Text) {}
}
```

<br/>

```go
// Server implementation
type chatHandler struct{}

func (chatHandler) Stream(stream Chat_StreamServer) error {
    for {
        txt, err := stream.Recv()

        if err != nil {
            return err
        }

        // Get the the peer
        p, ok := peer.FromContext(stream.Context())
        if !ok {
            panic("cannot get peer from context")
        }

        // Get the the peer's auth information
        info := noise.InfoFromPeer(p)
        if info == nil {
            panic("cannot get info from peer")
        }
		
        // Get the skademlia ID
        id := info.Get(skademlia.KeyID)
        if id == nil {
            panic("cannot get id from peer")
        }

        fmt.Printf("%s> %s\n", id, txt.Message)
	}
}

// Start the server and connect to a peer
func main() {
    flag.Parse()

    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        panic(err)
    }

    fmt.Println("Listening for peers on port:", listener.Addr().(*net.TCPAddr).Port)

    keys, err := skademlia.NewKeys(1, 1)
    if err != nil {
        panic(err)
    }

    addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))

    client := skademlia.NewClient(addr, keys, skademlia.WithC1(C1), skademlia.WithC2(C2))
    client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))

    // Start the server
    go func() {
        server := client.Listen()
        
        // Register the chat server
        RegisterChatServer(server, &chatHandler{})

        if err := server.Serve(listener); err != nil {
            panic(err)
        }
    }()
	
    // Dial peer located at address 127.0.0.1:3001
    if _, err := client.Dial("127.0.0.1:3001"); err != nil {
        panic(err)
    }
	
    client.Bootstrap()

    // Send a single message to the peers, knowing it's encrypted over the wire.
    conns := client.ClosestPeers()
    for _, conn := range conns {
        chat := NewChatClient(conn)

        stream, err := chat.Stream(context.Background())
        if err != nil {
            continue
        }

        if err := stream.Send(&Text{Message: "Hello peer!"}); err != nil {
            continue
        }
    }
}
```

## Setup

### Installation
Make sure to have at the bare minimum [Go 1.11](https://golang.org/dl/) installed before incorporating **noise** into your project and Go module enabled.

Install the protocol buffer implementation [https://github.com/protocolbuffers/protobuf](https://github.com/protocolbuffers/protobuf) and [gogo/protobuf](https://github.com/golang/protobuf).

To generate code for your protocol buffers.
```
protoc --gogofaster_out=plugins=grpc:. myproto.proto
```

### Import into your project

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run this inside your projects directory.
go get github.com/perlin-network/noise
```

### Run the examples

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run an example creating a cluster of 3 peers automatically
# discovering one another.
[terminal 1] go run examples/chat/main.go
Listening for peers on port: 3000
[terminal 2] go run examples/chat/main.go 127.0.0.1:3000
Listening for peers on port: 3001
[terminal 3] go run examples/chat/main.go 127.0.0.1:3001
Listening for peers on port: 3003

# Optionally run test cases.
go test -v -count=1 -race ./...
```

## Documentation

For more information, you can refer to the following:

- [Noise](docs/noise.md)

- [NAT Traversal](docs/nat.md) 


## We're hiring!

Here at [Perlin](https://perlin.net), we spend days and weeks debating, tinkering, and researching what is out there in academia to bring to industries truly resilient, open-source, secure, economic, and decentralized software to empower companies, startups, and users.
                                                        
Our doors are open to academics that have a knack for distributed systems, engineers that want to explore unknown waters, frontend developers that want to make and evangelize the next generation of customer-facing applications, and graphics designers that yearn to instrument together greater user experiences for decentralized applications.

## Contributions

First of all, _thank you so much_ for taking part in our efforts for creating a p2p networking stack that can meet everyones needs without sacrificing developer productivity!

All code contributions to _noise_ should comply with all idiomatic Go standards listed [here](https://github.com/golang/go/wiki/CodeReviewComments).

All commit messages should be in the format:

```bash
module_name_1, module_name_2: description of the changes you made to the two
    modules here as a sentence
```

Be sure to use only imperative, present tense within your commit messages and optionally include motivation for your changes _two lines breaks_ away from your commit message.

This allows other maintainers and contributors to know which modules you are modifying/creating within the code/docs repository.

Lastly, be sure to consider backwards compatibility.

Additionally, if you'd like to talk to us or any of the team in real-time, be sure to join our [Discord server](https://discord.gg/dMYfDPM)!

We are heavily active, ready to answer any questions/assist you with any code/doc contributions at almost any time.

## License
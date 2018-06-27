# Noise

An opinionated, easy-to-use P2P network stack for decentralized applications and cryptographic protocols by Perlin Network.

Made in Golang with a focus on easy-interoperability with other programming languages, devices, network protocols, and cryptography libraries.

Noise follows the DRY (don't-repeat-yourself) principle for its choices of technology. Especially when dealing with cryptosystems and networking, DRY as a principle plays an important role.

## Features

- Real-time, bidirectional streaming between peers via. gRPC and Protobufs.
- NaCL/Ed25519 scheme for peer identities and signatures.
- Kademlia-inspired peer discovery.

## Usage

```
# install vgo tooling
go get -u golang.org/x/vgo

# download the dependencies to vendor folder and run main.go
vgo mod -vendor
[terminal 1] vgo run main.go -port 3000
[terminal 2] vgo run main.go -port 3001 peers localhost:3000
[terminal 3] vgo run main.go -port 3002 peers localhost:3000

# build and run the main.go noise binary
vgo build
./noise
```

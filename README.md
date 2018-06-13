# Noise

An opinionated, easy-to-use P2P network stack for decentralized applications and cryptographic protocols by Perlin Network.

Made in Golang with a focus on easy-interoperability with other programming languages, devices, network protocols, and cryptography libraries.

Noise follows the DRY (don't-repeat-yourself) principle for its choices of technology. Especially when dealing with cryptosystems and networking, DRY as a principle plays an important role.

## Features

- Real-time, bidirectional streaming between peers via. gRPC and Protobufs.
- Business logic and concurrency modeling via. actor model.
- NaCL/Ed25519 scheme for peer identities and signatures.
- Kademlia-inspired peer discovery.
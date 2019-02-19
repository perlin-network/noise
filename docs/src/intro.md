# Introduction

Peer-to-peer (P2P) networking is something that is just _damn hard_ to get right.

The thing is, there has been no standardized solution to developing P2P applications that has been widely adopted by any community; with prior open-sourced attempts seeming like some mangled code pulled out of a poor magicians hat.

As a result, we see a lot of projects going for improper P2P networking solutions such as:

- **gRPC**, which was intended to be used for micro-services. **NOT** as a networking stack for trustless P2P applications.
- **Consul, Etcd, and even ZooKeeper,** which was intended to be used for service discovery in a _trusted, centralized network_.

And on the other hand, we see hodgepodges of code such as:

- **libp2p,** which feels oddly modular and far too verbose in all the wrong ways as though it was ripped out of IPFS.

In response, we now have a wide array of projects in the decentralized space, rolling out with their own immature networking stacks that are festered with bugs and low-level detail that is making the development of decentralized apps/cryptographic protocols just a huge conglomerated mess.

## What is `noise`?

**noise** was born out of the frustration that networking libraries typically trade off developer productivity in exchange for performance, security, and expressivity.

It is difficult to bring along a clear separation between a p2p applications networking protocol, and the underlying networking boilerplate code that comes with it.

As a result, the final product after months of work invested into building a p2p application is that its source code inevitably becomes garbled, hard-to-debug, and incredibly unintuitive to the eyes of any developer.

At [Perlin](https://perlin.net), we have faced this exact same issue time and time again building mission-critical p2p applications.

There does not exist a networking library out right now that has the capability of cleanly allowing a developer to declaratively implement complex, abstract networking protocols which p2p applications inevitably comprise of.

Libraries/frameworks that claim otherwise tend to abstract away the low-level networking details that a p2p application must absolutely have control of.    

This led us to building and open-sourcing noise: a networking stack that lets you declaratively derive complex structure out of a noisy network of nodes.

In developing Noise, our goals are three-fold:

- **Developer Ergonomics.** Developers should never feel forced to read from and write with, an ugly code mess tied to thousands of unnecessary dependencies with abhorrent documentation while writing secure, high-performance, and low-latency networking systems.

    Thousands of decentralized projects produce insightful, yet highly non-reusable code selfishly stuck to the abstractions of their own projects code-base for the sake of showing off their own coding prowess.

- **Don’t-Repeat-Yourself (DRY).** When dealing with security and performance, Noise is opinionated in having developers work with friendly and concise abstractions over battle-tested, high performance, and secure technologies such as Ed25519 signatures, TCP, zerolog, and protobufs.

    Tons of decentralized projects roll out their own custom bug-prone wire serialization formats, networking protocols, cryptosystems and signature schemes which only makes the learning curve for developers more tedious in taking apart/contributing to such projects.

- **Keep-It-Simple-Stupid (KISS).** Developers should feel comfortable in diving through Noise’s code-base, writing decentralized applications with a minimal amount of boilerplate while being backed by sufficient amounts of examples.

    Developers should not have to write 200 lines of code while digging through crappy documentation for the sake of writing a hello world application.

After iterating through hundreds of improvement and requests, we hope you enjoy using **noise** for your next p2p application.

— The Perlin Team
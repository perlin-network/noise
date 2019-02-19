# Setup

Make sure to have at the bare minimum [Go 1.11](https://golang.org/dl/) installed before incorporating **noise** into your project.

After installing _Go_, you may choose to either:

1. directly incorporate noise as a library dependency to your project,

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run this inside your projects directory.
go get github.com/perlin-network/noise
```

2. or checkout the source code on Github and run any of the following commands below.

```bash
# Be sure to have Go modules enabled: https://github.com/golang/go/wiki/Modules
export GO111MODULE=on

# Run an example creating a cluster of 3 peers automatically
# discovering one another.
[terminal 1] go run examples/chat/main.go -p 3000
[terminal 2] go run examples/chat/main.go -p 3001 127.0.0.1:3000
[terminal 3] go run examples/chat/main.go -p 3002 127.0.0.1:3001

# Optionally run test cases.
go test -v -count=1 -race ./...
```
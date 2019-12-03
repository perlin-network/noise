proto:
	protoc --gogofaster_out=plugins=grpc:. skademlia/rpc.proto
	protoc --gogofaster_out=plugins=grpc:. examples/benchmark_rpc/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/skademlia_rpc/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/skademlia_stream/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/chat/test.proto

fmt:
	go fmt ./...

lint:
#	https://github.com/golangci/golangci-lint#install
	golangci-lint -c .golangci.yml run

test:
	go test -timeout 10m -v -bench -race ./...

check: fmt lint test

license:
	addlicense -l mit -c Perlin $(PWD)
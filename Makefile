proto:
	protoc --gogofaster_out=plugins=grpc:. skademlia/rpc.proto
	protoc --gogofaster_out=plugins=grpc:. examples/benchmark_rpc/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/skademlia_rpc/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/skademlia_stream/test.proto
	protoc --gogofaster_out=plugins=grpc:. examples/chat/test.proto
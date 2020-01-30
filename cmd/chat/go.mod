module github.com/perlin-network/noise/cmd/chat

replace github.com/perlin-network/noise => ../../

go 1.13

require (
	github.com/perlin-network/noise v0.0.0-00010101000000-000000000000
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.13.0
)

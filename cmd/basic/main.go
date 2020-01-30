package main

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
	"go.uber.org/zap"
	"os"
	"os/signal"
)

func main() {
	logger, err := zap.NewDevelopment(zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	node, err := noise.NewNode(noise.WithNodeLogger(logger), noise.WithNodeBindPort(9000))
	if err != nil {
		panic(err)
	}
	defer node.Close()

	overlay := kademlia.New()
	node.Bind(overlay.Protocol())

	if err := node.Listen(); err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

package network

import (
	"github.com/xtaci/smux"
	"time"
)

func muxConfig() *smux.Config {
	config := smux.DefaultConfig()
	config.KeepAliveTimeout = 1 * time.Second
	config.KeepAliveInterval = 250 * time.Millisecond

	return config
}
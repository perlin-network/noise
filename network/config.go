package network

import (
	"github.com/xtaci/smux"
	"time"
)

func muxConfig() *smux.Config {
	config := smux.DefaultConfig()
	config.KeepAliveTimeout = 5 * time.Second
	config.KeepAliveInterval = 1 * time.Millisecond

	return config
}

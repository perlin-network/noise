package network

import (
	"github.com/xtaci/smux"
	"time"
)

func muxConfig() *smux.Config {
	config := smux.DefaultConfig()
	config.KeepAliveTimeout = 10 * time.Second
	config.KeepAliveInterval = 2 * time.Second

	return config
}

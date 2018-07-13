package network

import (
	"time"

	"github.com/xtaci/smux"
)

func muxConfig() *smux.Config {
	config := smux.DefaultConfig()
	config.KeepAliveTimeout = 10 * time.Second
	config.KeepAliveInterval = 2 * time.Second

	return config
}

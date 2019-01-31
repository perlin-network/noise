package timeout

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"time"
)

// Enforce initiates the timeout dispatcher to call a function should it take too long to perform
// a handshake.
func Enforce(peer *noise.Peer, key string, timeoutDuration time.Duration, fn func()) {
	if timeoutDuration != 0 {
		timeoutDispatcher := make(chan struct{}, 1)
		peer.Set(key, timeoutDispatcher)

		go func() {
			select {
			case <-timeoutDispatcher:
			case <-time.After(timeoutDuration):
				fn()
			}

			peer.Delete(key)
			close(timeoutDispatcher)
		}()
	}
}

// Clear dispatches to the timeout dispatcher that the peer has successfully completed its handshake.
func Clear(peer *noise.Peer, key string) error {
	if dispatcher := peer.Get(key); dispatcher != nil {
		dispatcher.(chan struct{}) <- struct{}{}
	} else {
		return errors.New("no timeout dispatcher was registered to the peer")
	}
	return nil
}

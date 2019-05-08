package noise

import (
	"sync"
	"time"
)

var timerPool sync.Pool

func acquireTimer(d time.Duration) *time.Timer {
	v := timerPool.Get()

	if v == nil {
		return time.NewTimer(d)
	}

	t := v.(*time.Timer)

	if t.Reset(d) {
		return time.NewTimer(d)
	}

	return t

}

func releaseTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	timerPool.Put(t)
}

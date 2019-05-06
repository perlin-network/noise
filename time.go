package noise

import (
	"sync"
	"time"
)

var timerPool sync.Pool

func acquireTimer() *time.Timer {
	v := timerPool.Get()
	if v == nil {
		return time.NewTimer(time.Hour * 24)
	}
	t := v.(*time.Timer)
	resetTimer(t, time.Hour*24)
	return t
}

func releaseTimer(t *time.Timer) {
	stopTimer(t)
	timerPool.Put(t)
}

func resetTimer(t *time.Timer, d time.Duration) {
	stopTimer(t)
	t.Reset(d)
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

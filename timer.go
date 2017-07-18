package act

import (
	"time"
)

//
// Timer
//
type Timer struct {
	ticker *time.Ticker
}

func (pid *Pid) SendAfterWithStop(data Term, timeoutMs uint32) *Timer {

	ticker := time.NewTicker(time.Duration(timeoutMs) * time.Millisecond)

	go func() {

		select {
		case <-ticker.C:
			pid.Cast(data)
		}

		ticker.Stop()

	}()

	return &Timer{ticker}
}

func (t *Timer) Stop() {
	if t == nil {
		return
	}

	t.ticker.Stop()
}

func (pid *Pid) SendAfter(data Term, timeoutMs uint32) {
	go func() {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		pid.Cast(data)
	}()
}

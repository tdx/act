package act

import (
	"time"
)

//
// Timer
//
type Timer struct {
	ticker   *time.Ticker
	stopChan chan<- bool
}

func (pid *Pid) SendAfterWithStop(data Term, timeoutMs uint32) *Timer {

	stop := make(chan bool, 1)
	ticker := time.NewTicker(time.Duration(timeoutMs) * time.Millisecond)

	go func() {

		select {
		case <-ticker.C:
			pid.Cast(data)
		case <-stop:
		}

		ticker.Stop()

	}()

	return &Timer{ticker, stop}
}

func (t *Timer) Stop() {
	if t == nil {
		return
	}

	defer func() {
		recover()
	}()

	close(t.stopChan)
}

func (pid *Pid) SendAfter(data Term, timeoutMs uint32) {
	go func() {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		pid.Cast(data)
	}()
}

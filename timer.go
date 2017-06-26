package act

import (
	"time"
)

//
// Timer
//
type Timer chan bool

func (pid *Pid) SendAfterWithStop(data Term, timeoutMs int32) Timer {
	stop := make(chan bool)

	go func() {
		ticker := time.NewTicker(time.Duration(timeoutMs) * time.Millisecond)

		select {
		case <-ticker.C:
			pid.Cast(data)
		case <-stop:
		}

		ticker.Stop()
	}()

	return stop
}

func (timer Timer) Stop() {
	close(timer)
}

func (pid *Pid) SendAfter(data Term, timeoutMs int32) {
	go func() {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		pid.Cast(data)
	}()
}

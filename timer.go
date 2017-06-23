package act

import (
	"time"
)

//
// Timer
//
type Timer chan bool

func (pid *Pid) SendAfter(data Term, timeoutMs int32) Timer {
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

func (timer Timer) TimerStop() {
	close(timer)
}

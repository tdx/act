package act

import (
	"sync"
	"time"
)

//
// Timer
//
type Timer struct {
	sync.Mutex
	stopChan chan bool
	active   bool
}

func (pid *Pid) SendAfterWithStop(data Term, timeoutMs uint32) *Timer {
	stop := make(chan bool)

	timer := Timer{stopChan: stop, active: true}

	go func() {
		ticker := time.NewTicker(time.Duration(timeoutMs) * time.Millisecond)

		select {
		case <-ticker.C:
			pid.Cast(data)
			timer.Stop()
		case <-stop:
		}

		ticker.Stop()
	}()

	return &timer
}

func (timer *Timer) Stop() {
	if timer == nil {
		return
	}

	timer.Lock()
	defer timer.Unlock()

	if timer.active {
		close(timer.stopChan)
		timer.active = false
	}
}

func (pid *Pid) SendAfter(data Term, timeoutMs uint32) {
	go func() {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		pid.Cast(data)
	}()
}

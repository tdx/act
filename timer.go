package act

import (
	"time"
)

//
// Timer to send event to pid
//
type Timer struct {
	timer *time.Timer
}

//
// SendAfterWithStop returns stoppable timer, after timeoutMs sends data event
// to pid
//
func (pid *Pid) SendAfterWithStop(data Term, timeoutMs uint32) *Timer {

	d := time.Duration(timeoutMs) * time.Millisecond
	timer := time.AfterFunc(d, func() { pid.Cast(data) })

	return &Timer{timer: timer}
}

//
// Stop stops the timer
//
func (t *Timer) Stop() {
	if t == nil {
		return
	}

	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
}

//
// SendAfter sends data event to pid after timeoutMs
//
func (pid *Pid) SendAfter(data Term, timeoutMs uint32) {
	go func() {
		time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
		pid.Cast(data)
	}()
}

package act

import (
	"testing"
	"time"
)

//
// Timers
//
func TestTimer(t *testing.T) {
	start_server(t)

	_, err := pid.Call(cmdStartTimer)
	if err != nil {
		t.Errorf("start timer failed: %s", err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err := inc(pid)
	if err != nil {
		t.Errorf("inc() failed: %s", err.Error())
	}

	if r != 101 {
		t.Errorf("inc() %d != 101", r)
	}

	pid.Stop()
}

func TestWaitTimer(t *testing.T) {
	start_server(t)

	_, err := pid.Call(cmdStartStoppableTimer)
	if err != nil {
		t.Errorf("start timer failed: %s", err)
	}

	time.Sleep(time.Duration(700) * time.Millisecond)

	r, err := inc(pid)
	if err != nil {
		t.Errorf("inc() failed: %s", err.Error())
	}

	if r != 101 {
		t.Errorf("inc() %d != 101", r)
	}

	pid.Stop()
}

func TestTimerStop(t *testing.T) {
	start_server(t)

	_, err := pid.Call(cmdStartStoppableTimer)
	if err != nil {
		t.Errorf("start timer failed: %s", err)
	}

	time.Sleep(time.Duration(200) * time.Millisecond)

	_, err = pid.Call(cmdStopTimer)
	if err != nil {
		t.Errorf("stop timer failed: %s", err)
	}

	// second stop must not be failed
	_, err = pid.Call(cmdStopTimer)
	if err != nil {
		t.Errorf("stop timer failed: %s", err)
	}

	r, err := inc(pid)
	if err != nil {
		t.Errorf("inc() failed: %s", err.Error())
	}

	if r != 11 {
		t.Errorf("inc() %d != 11", r)
	}

	pid.Stop()
}

package act

import (
	"sync/atomic"
	"testing"
	"time"
)

//
// Timers
//
func TestGoTimer(t *testing.T) {
	var gotEvent int64

	d := time.Duration(300) * time.Millisecond
	start := time.Now()

	sendFunc := func() {

		atomic.AddInt64(&gotEvent, 1)

		end := time.Now()
		t.Logf("%s timer fired: time %s\n", end, end.Sub(start))
	}

	t.Logf("%s shedule timer, %s\n", start, d)

	time.AfterFunc(d, sendFunc)

	time.Sleep(time.Duration(1000) * time.Millisecond)

	if gotEvent < 1 {
		t.Errorf("%s timer not fired: %#v", time.Now(), gotEvent)
	}
}

func TestTimer(t *testing.T) {
	startServer(t)

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
		t.Errorf("%s inc() %d != 101", time.Now(), r)
	}

	pid.Stop()
}

func TestWaitTimer(t *testing.T) {
	startServer(t)

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
		t.Errorf("%s inc() %d != 101", time.Now(), r)
	}

	pid.Stop()
}

func TestTimerStop(t *testing.T) {
	startServer(t)

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
		t.Errorf("%s inc() %d != 11", time.Now(), r)
	}

	pid.Stop()
}

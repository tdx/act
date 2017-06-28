package act

import (
	"fmt"
	"testing"
	"time"
)

//
// Test gen server
//
type gs struct {
	GenServerImpl
	i     int
	crash bool
	timer Timer
}

type reqInc struct {
	i int
}

const (
	cmdTest        string = "testCast"
	cmdStop        string = "stop"
	cmdCrash       string = "crash"
	cmdCrashInStop string = "stopCrash"
	cmdCallNoReply string = "callNoReply"

	cmdStartTimer          string = "startTimer"
	cmdStartStoppableTimer string = "startStoppableTimer"
	cmdStopTimer           string = "stopTimer"
)

var pid *Pid

func start(i int) (pid *Pid, err error) {

	s := new(gs)
	s.i = i

	pid, err = Spawn(s)

	return
}

func start_fail() (pid *Pid, err error) {

	s := new(gs)
	pid, err = Spawn(s, true)

	return
}

func start_prefix(prefix, name string) (*Pid, error) {

	s := new(gs)

	pid, err := SpawnPrefixName(s, prefix, name)

	return pid, err
}

func run_server() (*Pid, error) {

	s := new(gs)

	pid, err := Spawn(s)

	return pid, err
}

func inc(pid *Pid) (int, error) {
	r, err := pid.Call(&reqInc{})

	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case *reqInc:
		return r.i, nil
	}

	return 0, fmt.Errorf("unexpected answer: %#v", r)
}

// ----------------------------------------------------------------------------
// GenServer interface callbacks
// ----------------------------------------------------------------------------
func (s *gs) Init(args ...interface{}) (
	action GenInitAction, stopReason string) {

	if len(args) > 0 {
		switch stop := args[0].(type) {
		case bool:
			if stop == true {
				return GenInitStop, "simulate init failed"
			}
		}
	}

	return GenInitOk, ""
}

func (s *gs) HandleCall(req Term, from From) (
	action GenCallAction, reply Term, stopReason string) {

	switch req := req.(type) {
	case *reqInc:
		s.i++
		req.i = s.i

		return GenCallReply, req, ""

	case string:
		if req == cmdCrash {
			var i int = 1

			return GenCallReply, i / (i - 1), ""

		} else if req == cmdCallNoReply {
			go func() {
				time.Sleep(time.Duration(2) * time.Second)
				Reply(from, "ok")
			}()

			return GenCallNoReply, "", ""

		} else if req == cmdStartTimer {
			s.Self().SendAfter(cmdTest, 300)

			return GenCallReply, "ok", ""

		} else if req == cmdStartStoppableTimer {
			s.timer = s.Self().SendAfterWithStop(cmdTest, 500)

			return GenCallReply, "ok", ""

		} else if req == cmdStopTimer {
			s.timer.Stop()

			return GenCallReply, "ok", ""
		}
	}

	return GenCallStop,
		fmt.Sprintf("HandleCall, unexpected: %#v\n", req), "unexpected call"
}

func (s *gs) HandleCast(req Term) (
	action GenCastAction, stopReason string) {

	switch req := req.(type) {
	case string:
		if req == cmdTest {
			s.i = 100

		} else if req == cmdStop {
			return GenCastStop, cmdStop

		} else if req == cmdCrash {
			var i int = 1
			s.i = i / (i - 1)

		} else if req == cmdCrashInStop {
			s.crash = true
		}
	}

	return GenCastNoreply, ""
}

func (s *gs) Terminate(reason string) {
	// fmt.Printf("Terminate: %s, need crash: %v\n", reason, s.crash)

	if s.crash {
		var t map[string]int

		t["crash"] = 1
	}
}

// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------
func start_server(t *testing.T) {
	var err error
	pid, err = start(10)
	if err != nil {
		t.Fatalf("create server failed: %s", err.Error())
	}
}

func TestInitStop(t *testing.T) {
	var err error
	pid, err = start_fail()
	if err == nil {
		t.Fatal("start server must fail")
	}
}

func TestInit(t *testing.T) {
	start_server(t)
}

func TestCall(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	r, err := inc(pid)
	if err != nil {
		t.Errorf("inc() failed: %s", err.Error())
	}

	if r != 11 {
		t.Errorf("inc() != %d", r)
	}

	r, err = inc(pid)
	if err != nil {
		t.Errorf("inc() failed: %s", err.Error())
	}

	if r != 12 {
		t.Errorf("inc() %d != 12", r)
	}
}

func TestCallNoReply(t *testing.T) {
	r, err := pid.Call(cmdCallNoReply)
	if err != nil {
		t.Error("CallNoReply failed: %s", err)
	}

	if r != "ok" {
		t.Errorf("call(long reply) %#v, want ok", r)
	}
}

func TestCast(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	err := pid.Cast(cmdTest)
	if err != nil {
		t.Errorf("cast failed: %s", err.Error())
	}
}

func TestStop(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	err := pid.Stop()
	if err != nil {
		t.Errorf("stop failed: %s", err.Error())
	}
}

func TestStopStopped(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	err := pid.Stop()
	if err == nil {
		t.Error("stop stopped server error")
	}
}

func TestCallStopped(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	// any param
	_, err := pid.Call(cmdTest)
	if err == nil {
		t.Error("call stopped server error")
	}
}

func TestCastStopped(t *testing.T) {
	if pid == nil {
		t.Error("no server")
		return
	}

	// any param
	err := pid.Cast(cmdTest)
	if err == nil {
		t.Error("cast stopped server error")
	}
}

func TestCallStop(t *testing.T) {
	start_server(t)

	// unexpected call -> stop server
	_, err := pid.Call(cmdStop)
	if err != nil {
		t.Errorf("call failed: %s", err.Error())
	}

	// server must be stopped here
	_, err = pid.Call(cmdStop)
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestCastStop(t *testing.T) {
	start_server(t)

	// send stop command
	err := pid.Cast(cmdStop)
	if err != nil {
		t.Errorf("cast failed: %s", err.Error())
	}

	time.Sleep(time.Duration(2) * time.Second)

	err = pid.Cast(cmdStop)
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestNilPid(t *testing.T) {
	var badPid *Pid

	_, err := badPid.Call(cmdTest)
	if err == nil {
		t.Errorf("call on nil pid must fail")
	}

	err = badPid.Cast(cmdTest)
	if err == nil {
		t.Errorf("cast on nil pid must fail")
	}

	err = badPid.Stop()
	if err == nil {
		t.Errorf("stop on nil pid must fail")
	}
}

//
// Crash gen server process
//
func TestCrashInCall(t *testing.T) {
	start_server(t)

	// send crash command
	_, err := pid.Call(cmdCrash)
	if err == nil {
		t.Error("must return crash error")
	}

	// server must be stopped here
	_, err = pid.Call(cmdCrash)
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestCrashInCast(t *testing.T) {
	start_server(t)

	// send crash command
	err := pid.Cast(cmdCrash)
	if err != nil {
		t.Error("cast crash: %s", err)
	}
}

func TestCrashInStop(t *testing.T) {
	start_server(t)

	// send crash command
	err := pid.Cast(cmdCrashInStop)
	if err != nil {
		t.Error("stop crash: %s", err)
	}

	// crash
	pid.Stop()
}

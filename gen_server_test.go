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
	i          int
	crash      bool
	timer      *Timer
	gotTimeout bool
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
	cmdGetTimeout  string = "getTimeout"

	cmdStartTimer          string = "startTimer"
	cmdStartStoppableTimer string = "startStoppableTimer"
	cmdStopTimer           string = "stopTimer"

	cmdCallBadReply string = "badCallReply"
	cmdCastBadReply string = "badCastReply"

	cmdInitTimeout        string = "initTimeout"
	cmdCallTimeout        string = "callTimeout"
	cmdCallNoReplyTimeout string = "callNoReplyTimeout"
	cmdCastTimeout        string = "castTimeout"

	cmdLongCall string = "cmdLongCall"
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

func start_fail2() (pid *Pid, err error) {

	s := new(gs)
	pid, err = Spawn(s, "bad reply")

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
func (s *gs) Init(args ...interface{}) Term {

	if len(args) > 0 {
		switch arg := args[0].(type) {
		case bool:
			if arg == true {
				return &GsInitStop{"simulate init failed"}
			}
		case string:
			if arg == "bad reply" {
				return "init bad reply"
			} else if arg == cmdInitTimeout {
				return &GsInitOkTimeout{300}
			}
		}
	}

	return GsInitOk
}

func (s *gs) HandleCall(req Term, from From) Term {

	switch req := req.(type) {
	case *reqInc:
		s.i++
		req.i = s.i

		return &GsCallReply{req}

	case string:
		if req == cmdCrash {
			var i int = 1

			return &GsCallReply{i / (i - 1)}

		} else if req == cmdCallNoReply {
			go func() {
				time.Sleep(time.Duration(2) * time.Second)
				Reply(from, "ok")
			}()

			return GsCallNoReply

		} else if req == cmdStartTimer {
			s.Self().SendAfter(cmdTest, 300)

			return GsCallReplyOk

		} else if req == cmdStartStoppableTimer {
			s.timer = s.Self().SendAfterWithStop(cmdTest, 500)

			return GsCallReplyOk

		} else if req == cmdStopTimer {
			s.timer.Stop()

			return GsCallReplyOk

		} else if req == cmdCallBadReply {

			return false

		} else if req == cmdGetTimeout {

			return &GsCallReply{s.gotTimeout}

		} else if req == cmdCallTimeout {

			s.gotTimeout = false

			return &GsCallReplyTimeout{"ok", 300}

		} else if req == cmdCallNoReplyTimeout {
			go func() {
				time.Sleep(time.Duration(100) * time.Millisecond)
				Reply(from, "ok")
			}()

			return &GsCallNoReplyTimeout{300}

		} else if req == cmdLongCall {

			time.Sleep(time.Duration(6) * time.Second)

			return GsCallReplyOk
		}
	}

	return &GsCallStop{
		Reason: "unexpected call",
		Reply:  fmt.Sprintf("HandleCall, unexpected: %#v\n", req)}
}

func (s *gs) HandleCast(req Term) Term {

	switch req := req.(type) {
	case GsTimeout:
		// fmt.Printf("%s HandleCast: timeout\n", time.Now())
		s.gotTimeout = true

	case string:
		if req == cmdTest {
			// fmt.Printf("%s HandleCast: %s\n", time.Now(), cmdTest)
			s.i = 100

		} else if req == cmdStop {
			return &GsCastStop{cmdStop}

		} else if req == cmdCrash {
			var i int = 1
			s.i = i / (i - 1)

		} else if req == cmdCrashInStop {
			s.crash = true

		} else if req == cmdCastBadReply {

			return false

		} else if req == cmdCastTimeout {

			s.gotTimeout = false

			return &GsCastNoReplyTimeout{300}
		}
	default:
		fmt.Printf("%s HandleCast: unexpected request: %#v\n", time.Now(), req)
	}

	return GsCastNoReply
}

func (s *gs) Terminate(reason string) {
	// fmt.Printf("pid # %d: Terminate: %s, need crash: %v\n",
	// 	s.Self().Id(), reason, s.crash)

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

func start_timeout(t *testing.T) {
	var err error

	s := new(gs)
	pid, err = Spawn(s, cmdInitTimeout)

	if err != nil {
		t.Fatalf("create server failed: %s", err.Error())
	}
}

func start_server_opts(b *testing.B, opts *Opts) {
	var err error

	s := new(gs)
	pid, err = SpawnOpts(s, opts)

	if err != nil {
		b.Fatalf("create server failed: %s", err.Error())
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

	// crash, stpped in any case
	pid.Stop()
}

//
// bad init call, cast reply -> process stopped
//
func TestBadInitReply(t *testing.T) {

	pid, err := start_fail2()

	if err == nil {
		t.Error("server must not be started")
	}

	_, err = pid.Call(cmdTest)
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestBadCallReply(t *testing.T) {
	start_server(t)

	_, err := pid.Call(cmdCallBadReply)
	if err == nil {
		t.Error("call must fail")
	}

	_, err = pid.Call(cmdTest)
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestBadCastReply(t *testing.T) {
	start_server(t)

	err := pid.Cast(cmdCastBadReply)
	if err != nil {
		t.Error(err)
	}

	_, err = pid.Call(cmdTest) // call timeout
	if err == nil {
		t.Error("server must be stopped")
	}
}

func TestBadCastReply2(t *testing.T) {
	start_server(t)

	err := pid.Cast(cmdCastBadReply)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {
		go func() {
			_, err = pid.Call(cmdTest) // call timeout
			if err == nil {
				t.Error("server must be stopped")
			}
		}()
	}

	time.Sleep(time.Duration(1) * time.Second)
}

func TestBadCastReply3(t *testing.T) {
	start_server(t)

	err := pid.Cast(cmdCastBadReply)
	if err != nil {
		t.Error(err)
	}

	// send on closed channel
	err = pid.Stop()
	if err == nil {
		t.Error("server must be stopped")
	}
}

//
// Timeout
//
func TestInitTimeout(t *testing.T) {
	start_timeout(t)

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != true {
		t.Errorf("%s init timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCallTimeout(t *testing.T) {
	start_server(t)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("getTimeout must return false")
	}

	_, err = pid.Call(cmdCallTimeout)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err = pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != true {
		t.Errorf("%s call timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCallNoReplyTimeout(t *testing.T) {
	start_server(t)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("getTimeout must return false")
	}

	_, err = pid.Call(cmdCallNoReplyTimeout)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err = pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != true {
		t.Errorf("%s call timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCastTimeout(t *testing.T) {
	start_server(t)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("getTimeout must return false")
	}

	err = pid.Cast(cmdCastTimeout)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err = pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != true {
		t.Errorf("%s cast timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCancelInitTimeout(t *testing.T) {
	start_timeout(t)

	// any call/cast must cancel timeout
	_, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Errorf("%s cancel timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCancelCallTimeout(t *testing.T) {
	start_server(t)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("getTimeout must return false")
	}

	_, err = pid.Call(cmdCallTimeout)
	if err != nil {
		t.Error(err)
	}

	// any call/cast must cancel timeout
	err = pid.Cast(cmdTest)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err = pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Errorf("%s call timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

func TestCancelCastTimeout(t *testing.T) {
	start_server(t)

	r, err := pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("getTimeout must return false")
	}

	err = pid.Cast(cmdCastTimeout)
	if err != nil {
		t.Error(err)
	}

	// any call/cast must cancel timeout
	err = pid.Cast(cmdTest)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Duration(500) * time.Millisecond)

	r, err = pid.Call(cmdGetTimeout)
	if err != nil {
		t.Error(err)
	}

	if r != false {
		t.Error("%s cast timeout failed: %#v", time.Now(), r)
	}

	pid.Stop()
}

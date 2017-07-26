package act

import (
	"errors"
	"fmt"
	"time"
	// "runtime"
)

//
// No process error
//
type gsNoProcError int

func (e gsNoProcError) Error() string {
	return "no_proc"
}

func IsNoProcError(err error) bool {
	if _, ok := err.(gsNoProcError); ok {
		return true
	}

	return false
}

//
// gen_server's return types
//

type GsTimeout int

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------
//
// ok
// {ok, Timeout}
// {stop, Reason}
//
type gsInitOk int

type GsInitOkTimeout struct {
	Timeout uint32
}

type GsInitStop struct {
	Reason string
}

// ---------------------------------------------------------------------------
// Cast
// ---------------------------------------------------------------------------
//
// noreply
// {noreply, Timeout}
// {stop, Reason}
//
type gsCastNoReply int

type GsCastNoReplyTimeout struct {
	Timeout uint32
}

type GsCastStop struct {
	Reason string
}

// ---------------------------------------------------------------------------
// Call
// ---------------------------------------------------------------------------
//
// {reply, Reply}
// {reply, Reply, Timeout}
// noreply
// {noreply, Timeout}
// {stop, Reason, Reply}
//

// reply "ok"
type gsCallReplyOk int

type GsCallReply struct {
	Reply Term
}

type GsCallReplyTimeout struct {
	Reply   Term
	Timeout uint32
}

type gsCallNoReply int

type GsCallNoReplyTimeout struct {
	Timeout uint32
}

type GsCallStop struct {
	Reason string
	Reply  Term
}

const (
	defaultCallTimeoutMs uint32 = 5000

	replyOk string = "ok"

	gsTimeout     GsTimeout     = 0
	GsInitOk      gsInitOk      = 1
	GsCastNoReply gsCastNoReply = 2
	GsCallNoReply gsCallNoReply = 3
	GsCallReplyOk gsCallReplyOk = 4
	GsNoProcError gsNoProcError = 5
)

//
// Types to communicate with gen_server
//

// Cast arg
type genReq struct {
	data Term
}

type From chan<- Term

// Call arg
type genCallReq struct {
	data      Term
	replyChan From
}

// Stop arg
type stopReq struct {
	reason    string
	replyChan chan<- bool
}

//
// GenServer interface
//
type GenServer interface {
	Init(args ...interface{}) (result Term)
	HandleCall(req Term, from From) (result Term)
	HandleCast(req Term) (result Term)
	Terminate(reason string)

	// private
	setPid(pid *Pid)
	setPrefix(prefix string)
	setName(name interface{})
}

// ProcessLoop executes during whole time of process life.
// It receives incoming messages from channels and handle it
// using methods of implementation
func GenServerLoop(
	gs GenServer,
	prefix string,
	name interface{},
	initChan chan Term,
	pid *Pid,
	args ...interface{}) {

	var timer *Timer
	var replyCall chan<- Term
	var replyStop chan<- bool
	inCall := false
	inStop := false
	inTerminate := false

	defer func() {

		UnregisterPrefix(prefix, name)
		timer.Stop()
		pid.flushMessages(prefix, name)
		pid.closeChannels(prefix, name)

		if r := recover(); r != nil {

			fmt.Printf("pid #%d/%s/%v: GenServer recovered: %#v\n",
				pid.Id(), prefix, name, r)
			// trace := make([]byte, 1024)
			// count := runtime.Stack(trace, true)
			// fmt.Printf("pid #%d/%s/%s: Stack of %d bytes: %s\n",
			// 	pid.Id(), prefix, name, count, trace)

			if !inTerminate {
				gs.Terminate(fmt.Sprintf("crashed: %#v", r))
			}

			if inCall {
				replyCall <- fmt.Errorf("crashed: %#v", r)
			}

			if inStop {
				replyStop <- true
			}
		}

	}()

	gs.setPid(pid)
	gs.setPrefix(prefix)
	gs.setName(name)

	result := gs.Init(args...)

	nLog("init result: %#v", result)

	switch r := result.(type) {

	case gsInitOk:
		initChan <- result

	case *GsInitOkTimeout:
		initChan <- result
		timer = pid.SendAfterWithStop(gsTimeout, r.Timeout)

	case *GsInitStop:
		initChan <- result
		return

	default:
		initChan <- fmt.Errorf("Init bad reply: %#v", r)
		return
	}

	for {

		inCall = false
		inStop = false
		inTerminate = false

		select {
		case m := <-pid.inChan:

			timer.Stop()

			switch m := m.(type) {

			// Call
			case *genCallReq:

				inCall = true
				replyCall = m.replyChan

				nLog("call message: %#v", m)
				result := gs.HandleCall(m.data, m.replyChan)
				nLog("call result: %#v", result)

				inCall = false

				switch result := result.(type) {

				case *GsCallReply:
					m.replyChan <- result.Reply

				case gsCallReplyOk:
					m.replyChan <- replyOk

				case *GsCallReplyTimeout:
					m.replyChan <- result.Reply
					timer = pid.SendAfterWithStop(gsTimeout, result.Timeout)

				case gsCallNoReply:

				case *GsCallNoReplyTimeout:
					timer = pid.SendAfterWithStop(gsTimeout, result.Timeout)

				case *GsCallStop:
					inTerminate = true
					m.replyChan <- result.Reply
					gs.Terminate(result.Reason)
					return

				default:
					reply := fmt.Sprintf("HandleCall bad reply: %#v", result)
					m.replyChan <- errors.New(reply)
					inTerminate = true
					gs.Terminate(reply)
					return
				}

			// Cast
			case *genReq:

				nLog("cast message: %#v", m)
				result := gs.HandleCast(m.data)
				nLog("cast result: %#v", result)

				switch result := result.(type) {

				case gsCastNoReply:

				case *GsCastNoReplyTimeout:
					timer = pid.SendAfterWithStop(gsTimeout, result.Timeout)

				case *GsCastStop:
					inTerminate = true
					gs.Terminate(result.Reason)
					return

				default:
					inTerminate = true
					gs.Terminate(
						fmt.Sprintf("HandleCast bad reply: %#v", result))
					return
				}
			}

		// Stop
		case m := <-pid.stopChan:

			timer.Stop()

			inStop = true
			replyStop = m.replyChan

			nLog("stop message: %s", m.reason)
			inTerminate = true
			gs.Terminate(m.reason)

			m.replyChan <- true

			inStop = false

			return
		} // select
	} // for
}

// ---------------------------------------------------------------------------
func Reply(replyTo From, data Term) {
	replyTo <- data
}

func (pid *Pid) Call(data Term) (reply Term, err error) {

	defer func() {
		if r := recover(); r != nil {
			reply = nil
			err = fmt.Errorf("pid #%d: call recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil && pid.inChan != nil {

		var replyTerm Term

		replyChan := make(chan Term, 1)
		pid.inChan <- &genCallReq{data, replyChan}
		replyTerm = <-replyChan

		// server stopped
		if replyTerm == nil {
			return nil, GsNoProcError
		}

		// call crashed ?
		switch err := replyTerm.(type) {
		case error:
			return nil, err
		}

		return replyTerm, nil
	}

	return nil, GsNoProcError
}

func (pid *Pid) Cast(data Term) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: cast recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil && pid.inChan != nil {
		pid.inChan <- &genReq{data}

		return nil
	}

	return GsNoProcError
}

func (pid *Pid) Stop() error {
	return pid.StopReason("stop")
}

func (pid *Pid) StopReason(reason string) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: stop recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil && pid.stopChan != nil {

		replyChan := make(chan bool)
		pid.stopChan <- &stopReq{reason, replyChan}
		<-replyChan

		return nil
	}

	return GsNoProcError
}

// ---------------------------------------------------------------------------
func (pid *Pid) closeChannels(prefix string, name interface{}) {
	if pid == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("pid #%d/%s/%v: closeChannels recovered: %#v\n",
				pid.Id(), prefix, name, r)
		}
	}()

	close(pid.inChan)
	close(pid.stopChan)
}

func (pid *Pid) flushMessages(prefix string, name interface{}) {
	if pid == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("pid #%d/%s/%v: flushMessages recovered: %#v\n",
				pid.Id(), prefix, name, r)
		}
	}()

	for {
		select {
		case m := <-pid.inChan:
			switch m := m.(type) {
			case *genCallReq:
				fmt.Printf("%s flushMessages: pid #%d/%s/%s: %#v\n",
					time.Now(), pid.Id(), prefix, name, m)
				close(m.replyChan)
			}
		default:
			return
		}
	}
}

package act

import (
	"errors"
	"fmt"
	// "runtime"
	"time"
)

//
// No process error
//
type gsNoProcError int

func (e gsNoProcError) Error() string {
	return "no_proc"
}

//
// IsNoProcError checks if error is of type of gsNoProcError
//
func IsNoProcError(err error) bool {
	if _, ok := err.(gsNoProcError); ok {
		return true
	}

	return false
}

//
// gen_server's return types
//

//
// GsTimeout is sent to the process if the inactivity timer has expired
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

//
// GsInitOkTimeout is returned from the Init callback to indicate
// that the process initialization is successful and an inactivity
// timer must be set
//
type GsInitOkTimeout struct {
	Timeout uint32
}

//
// GsInitStop is returned from the Init callback to indicat that
// the process must be stopped
//
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

//
// GsCastNoReplyTimeout is returned from the HandleCast callback to indicate
// that an inactivity timer must be set
//
type GsCastNoReplyTimeout struct {
	Timeout uint32
}

//
// GsCastStop is returned from the HandleCast callback to indicate that
// the process must be stopped
//
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

type gsCallReplyOk int

//
// GsCallReply is returned from the HandleCall callback to indicate that
// the process returns result in Reply
//
type GsCallReply struct {
	Reply Term
}

//
// GsCallReplyTimeout is returned from the HandleCall callback to indicate that
// the process returns result in Reply and an inactivity timer must be set
//
type GsCallReplyTimeout struct {
	Reply   Term
	Timeout uint32
}

type gsCallNoReply int

//
// GsCallNoReplyTimeout is returned from the HandleCall callback to indicate
// that an inactivity timer must be set. Result to caller returned with Reply()
//
type GsCallNoReplyTimeout struct {
	Timeout uint32
}

//
// GsCallStop is returned from the HandleCall callback to indicate that
// the process must be stopped
//
type GsCallStop struct {
	Reason string
	Reply  Term
}

const (
	replyOk string = "ok"

	gsTimeout GsTimeout = 0
	// GsInitOk is returned from Init callback to indicate that initialization
	// of process is successful
	GsInitOk gsInitOk = 1
	// GsCastNoReply is returned from HandleCast callback to indicate that no
	// result to reply to caller
	GsCastNoReply gsCastNoReply = 2
	// GsCallNoReply is returned from HandleCall callback to indicate that no
	// result to reply to caller
	GsCallNoReply gsCallNoReply = 3
	// GsCallReplyOk is standard reply from HandleCall
	GsCallReplyOk gsCallReplyOk = 4
	// GsNoProcError can be returned from Cast or Call if process identificated
	// by pid is not exists
	GsNoProcError gsNoProcError = 5
)

//
// Types to communicate with gen_server
//

// Cast arg
type genReq struct {
	data Term
}

// From type for Reply() method
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

// GenServerLoop executes during whole time of process life.
// It receives incoming messages from channels and handle it
// using methods of implementation
func (a *act) GenServerLoop(
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

		a.UnregisterPrefix(prefix, name)
		timer.Stop()
		pid.flushMessages(prefix, name)
		pid.closeChannels(prefix, name)

		if r := recover(); r != nil {

			now := time.Now().Truncate(time.Microsecond)

			fmt.Printf("%s pid #%d/%s/%v: GenServer recovered: %#v\n",
				now, pid.Id(), prefix, name, r)

			// trace := make([]byte, 512)
			// count := runtime.Stack(trace, true)
			// fmt.Printf("%s pid #%d/%s/%s: Stack of %d bytes: %s\n",
			// 	now, pid.Id(), prefix, name, count, trace)

			if !inTerminate {
				inTerminate = true
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

			// channel closed
			if m == nil {
				return
			}

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
					m.replyChan <- result.Reply
					inTerminate = true
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

			// channel closed
			if m == nil {
				return
			}

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

//
// Reply sends reply to caller
//
func Reply(replyTo From, data Term) {
	replyTo <- data
}

//
// Call makes a synchronous call to the process
//
func (pid *Pid) Call(data Term) (reply Term, err error) {

	defer func() {
		if r := recover(); r != nil {
			reply = nil
			err = fmt.Errorf("pid #%d: call recovered: %#v", pid.Id(), r)
		}
	}()

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

//
// Cast makes an asynchronous call to the process
//
func (pid *Pid) Cast(data Term) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: cast recovered: %#v", pid.Id(), r)
		}
	}()

	pid.inChan <- &genReq{data}

	return nil
}

//
// Stop makes synchronous stop request to the process
//
func (pid *Pid) Stop() error {
	return pid.StopReason("stop")
}

//
// StopReason makes synchronous stop request to the process
// Reason is the reason to stop the process
//
func (pid *Pid) StopReason(reason string) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: stop recovered: %#v", pid.Id(), r)
		}
	}()

	replyChan := make(chan bool)
	pid.stopChan <- &stopReq{reason, replyChan}
	<-replyChan

	return nil
}

// ---------------------------------------------------------------------------
func (pid *Pid) closeChannels(prefix string, name interface{}) {

	defer func() {
		if r := recover(); r != nil {
			now := time.Now().Truncate(time.Microsecond)

			fmt.Printf("%s pid #%d/%s/%v: closeChannels recovered: %#v\n",
				now, pid.Id(), prefix, name, r)

			// trace := make([]byte, 512)
			// count := runtime.Stack(trace, true)
			// fmt.Printf("%s pid #%d/%s/%s: closeChannels stack of %d bytes: %s\n",
			// 	now, pid.Id(), prefix, name, count, trace)
		}
	}()

	close(pid.inChan)
	close(pid.stopChan)
}

func (pid *Pid) flushMessages(prefix string, name interface{}) {

	defer func() {
		if r := recover(); r != nil {
			now := time.Now().Truncate(time.Microsecond)

			fmt.Printf("%s pid #%d/%s/%v: flushMessages recovered: %#v\n",
				now, pid.Id(), prefix, name, r)

			// trace := make([]byte, 512)
			// count := runtime.Stack(trace, true)
			// fmt.Printf("%s pid #%d/%s/%s: flushMessages stack of %d bytes: %s\n",
			// 	now, pid.Id(), prefix, name, count, trace)
		}
	}()

	for len(pid.inChan) > 0 {
		select {
		case m := <-pid.inChan:
			switch m := m.(type) {
			case *genCallReq:
				fmt.Printf("%s flushMessages: pid #%d/%s/%s: %#v\n",
					time.Now().Truncate(time.Microsecond),
					pid.Id(), prefix, name, m)
				close(m.replyChan)
			}
		default:
			break
		}
	}

	for len(pid.stopChan) > 0 {
		select {
		case m := <-pid.stopChan:
			if m != nil {
				fmt.Printf("%s flushMessages: pid #%d/%s/%s: %#v\n",
					time.Now().Truncate(time.Microsecond),
					pid.Id(), prefix, name, m)
				close(m.replyChan)
			}
		default:
			break
		}
	}
}

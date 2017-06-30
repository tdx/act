package act

import (
	"errors"
	"fmt"
	// "runtime"
	"time"
)

const (
	defaultCallTimeoutMs uint32 = 5000
)

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------
//
// ok
// {ok, Timeout}
// {stop, Reason}
//
type GsInitOk struct {
}

type GsInitOkTimeout struct {
	timeout uint32
}

type GsInitStop struct {
	reason string
}

// ---------------------------------------------------------------------------
// Cast
// ---------------------------------------------------------------------------
//
// noreply
// {noreply, Timeout}
// {stop, Reason}
//
type GsCastNoReply struct {
}

type GsCastNoReplyTimeout struct {
	timeout uint32
}

type GsCastStop struct {
	reason string
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
type GsCallReply struct {
	reply Term
}

type GsCallReplyTimeout struct {
	reply   Term
	timeout uint32
}

type GsCallNoReply struct {
}

type GsCallNoReplyTimeout struct {
	timeout uint32
}

type GsCallStop struct {
	reason string
	reply  Term
}

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

	// returns process-related options
	// chan-size : int
	Options() (options map[string]interface{})

	// private
	setPid(pid *Pid)
	setPrefix(prefix string)
	setName(name string)
}

// ProcessLoop executes during whole time of process life.
// It receives incoming messages from channels and handle it
// using methods of implementation
func GenServerLoop(
	gs GenServer,
	prefix, name string,
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

		pid.flushMessages()
		pid.closeChannels()
		timer.Stop()

		if r := recover(); r != nil {

			fmt.Printf("pid #%d/%s/%s: GenServer recovered: %#v\n",
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

		UnregisterPrefix(prefix, name)
	}()

	gs.setPid(pid)
	gs.setPrefix(prefix)
	gs.setName(name)

	result := gs.Init(args...)

	nLog("init result: %#v", result)

	switch r := result.(type) {
	case *GsInitOk:
		initChan <- result
	case *GsInitOkTimeout:
		initChan <- result
		timer = pid.SendAfterWithStop("timeout", r.timeout)
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
					m.replyChan <- result.reply
				case *GsCallReplyTimeout:
					m.replyChan <- result.reply
					timer = pid.SendAfterWithStop("timeout", result.timeout)
				case *GsCallNoReply:
				case *GsCallNoReplyTimeout:
					timer = pid.SendAfterWithStop("timeout", result.timeout)
				case *GsCallStop:
					inTerminate = true
					m.replyChan <- result.reply
					gs.Terminate(result.reason)
					return
				default:
					reply := fmt.Sprintf("HanelCall bad reply: %#v", result)
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
				case *GsCastNoReply:
				case *GsCastNoReplyTimeout:
					timer = pid.SendAfterWithStop("timeout", result.timeout)
				case *GsCastStop:
					inTerminate = true
					gs.Terminate(result.reason)
					return
				default:
					inTerminate = true
					gs.Terminate(
						fmt.Sprintf("HanelCast bad reply: %#v", result))
					return
				}
			}

		// Stop
		case m := <-pid.stopChan:

			timer.Stop()

			inStop = true
			replyStop = m.replyChan

			nLog("stop message")
			inTerminate = true
			gs.Terminate("stop")

			m.replyChan <- true

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

		replyChan := make(chan Term)
		pid.inChan <- &genCallReq{data, replyChan}

		var replyTerm Term

		ticker := time.NewTicker(
			time.Duration(defaultCallTimeoutMs) * time.Millisecond)
		defer ticker.Stop()

		select {
		case replyTerm = <-replyChan:
		case <-ticker.C:
			close(replyChan)
			return nil, errors.New("timeout")
		}

		// server stopped
		if replyTerm == nil {
			return nil, errors.New(NoProc)
		}

		// call crashed ?
		switch err := replyTerm.(type) {
		case error:
			return nil, err
		}

		return replyTerm, nil
	}

	return nil, errors.New(NoProc)
}

func (pid *Pid) Cast(data Term) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: cast recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil {
		if pid.inChan != nil {
			pid.inChan <- &genReq{data}

			return nil
		}
	}

	return errors.New(NoProc)
}

func (pid *Pid) Stop() (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pid #%d: stop recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil && pid.stopChan != nil {

		replyChan := make(chan bool)
		pid.stopChan <- &stopReq{replyChan}
		<-replyChan

		return nil
	}

	return errors.New(NoProc)
}

// ---------------------------------------------------------------------------
func (pid *Pid) closeChannels() {
	if pid != nil {
		close(pid.inChan)
		close(pid.stopChan)
	}
}

func (pid *Pid) flushMessages() {
	if pid == nil {
		return
	}

	n := len(pid.inChan)
	for i := 0; i < n; n++ {
		select {
		case m := <-pid.inChan:
			switch m := m.(type) {
			case *genCallReq:
				close(m.replyChan)
			}
		}
	}
}

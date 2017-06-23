package act

import (
	"errors"
	"fmt"
	"runtime"
)

type GenInitAction int
type GenCastAction int
type GenCallAction int

const (
	// init
	GenInitOk   GenInitAction = 0
	GenInitStop GenInitAction = 1

	// cast
	GenCastNoreply GenCastAction = 2
	GenCastStop    GenCastAction = 3

	// call
	GenCallReply   GenCallAction = 4
	GenCallNoReply GenCallAction = 5
	GenCallStop    GenCallAction = 6
)

/*
  ok
  {stop, Reason}
*/
type genInitReply struct {
	action GenInitAction
	reason string
}

/*
  noreply
  {stop, Reason}
*/
type genReq struct {
	data Term
}

type From chan Term

/*
  noreply
  {reply, Reply}
  {stop, Reason}
*/
type genCallReq struct {
	data      Term
	replyChan From
}

type stopReq struct {
	replyChan chan bool
}

// GenServer interface
type GenServer interface {
	Init(args ...interface{}) (action GenInitAction, stopReason string)

	HandleCast(req Term) (action GenCastAction, stopReason string)

	HandleCall(req Term, from From) (
		action GenCallAction, reply Term, stopReason string)

	HandleInfo(msg Term)

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
	initCh chan *genInitReply,
	pid *Pid,
	args ...interface{}) {

	defer func() {
		if r := recover(); r != nil {

			trace := make([]byte, 1024)
			count := runtime.Stack(trace, true)
			fmt.Printf("pid #%d/%s/%s: GenServer recovered: %#v\n",
				pid.Id(), prefix, name, r)
			fmt.Printf("pid #%d/%s/%s: Stack of %d bytes: %s\n",
				pid.Id(), prefix, name, count, trace)

			gs.Terminate(fmt.Sprintf("crashed: %#v", r))
		}

		UnregisterPrefix(prefix, name)
		pid.closePidChannels()
	}()

	gs.setPid(pid)
	gs.setPrefix(prefix)
	gs.setName(name)

	action, reason := gs.Init(args...)

	nLog("init action: %#v, reason: %#v", action, reason)

	initCh <- &genInitReply{action, reason}
	if action == GenInitStop {
		return
	}

	for {
		select {
		case m := <-pid.inChan:

			switch m := m.(type) {
			case *genCallReq:
				nLog("call message: %#v", m)
				action, reply, reason := gs.HandleCall(m.data, m.replyChan)
				nLog("call action: %#v, reply: %#v", action, reply)

				if action != GenCallNoReply {
					m.replyChan <- reply
				}

				if action == GenCallStop {
					gs.Terminate(reason)
					return
				}
			case *genReq:
				nLog("cast message: %#v", m)
				action, reason := gs.HandleCast(m.data)
				nLog("cast action: %#v", action)

				if action == GenCastStop {
					gs.Terminate(reason)
					return
				}
			}

		case m := <-pid.stopChan:

			nLog("stop message")
			gs.Terminate("stop")
			m.replyChan <- true

			return
		} // select
	} // for
}

// ---------------------------------------------------------------------------
func (pid *Pid) Call(data Term) (reply Term, err error) {

	defer func() {
		if r := recover(); r != nil {
			reply = nil
			err = fmt.Errorf("pid #%d: call recovered: %#v", pid.Id(), r)
		}
	}()

	if pid != nil {
		if pid.inChan != nil {
			replyChan := make(chan Term)
			pid.inChan <- &genCallReq{data, replyChan}
			replyTerm := <-replyChan

			return replyTerm, nil
		} else {
			return nil, errors.New("inChan is nil")
		}
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
		} else {
			return errors.New("inChan is nil")
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

	if pid != nil {
		if pid.stopChan != nil {
			replyChan := make(chan bool)
			pid.stopChan <- &stopReq{replyChan}
			<-replyChan
			// close(pid.stopChan)

			return nil
		} else {
			return errors.New("stopChan is nil")
		}
	}

	return errors.New(NoProc)
}

package act

import (
	"errors"
	"flag"
	"fmt"
	"log"
)

type Term interface{}
type Tuple []Term
type List []Term
type Atom string

type Pid struct {
	id       uint64
	inChan   chan interface{}
	stopChan chan *stopReq
}

type makePidResp struct {
	pid *Pid
	err error
}

type makePidReq struct {
	prefix  string
	name    string
	replyTo chan<- makePidResp
}

type regNameReq struct {
	prefix  string
	name    string
	pid     *Pid
	replyTo chan<- bool
}

type unregNameReq struct {
	prefix  string
	name    string
	replyTo chan<- bool
}

type whereNameReq struct {
	prefix  string
	name    string
	replyTo chan<- *Pid
}

type regMap map[string]*Pid
type wherePrefixReq struct {
	prefix  string
	replyTo chan<- regMap
}

type registryChan struct {
	makePidChan     chan makePidReq
	regNameChan     chan regNameReq
	unregNameChan   chan unregNameReq
	whereNameChan   chan whereNameReq
	wherePrefixChan chan wherePrefixReq
}

type act struct {
	serial     uint64
	registry   *registryChan
	registered map[string]regMap
}

const NoProc string = "no_proc"

// ---------------------------------------------------------------------------
var nTrace bool
var env *act

func init() {
	flag.BoolVar(&nTrace, "trace", false, "trace actors")

	registry := &registryChan{
		makePidChan:     make(chan makePidReq),
		regNameChan:     make(chan regNameReq),
		unregNameChan:   make(chan unregNameReq),
		whereNameChan:   make(chan whereNameReq),
		wherePrefixChan: make(chan wherePrefixReq),
	}

	env = &act{
		registry:   registry,
		registered: make(map[string]regMap),
	}

	// without prefix
	env.registered[""] = make(regMap)

	env.run()
}

func nLog(f string, a ...interface{}) {
	if nTrace {
		log.Printf(f, a...)
	}
}

// ---------------------------------------------------------------------------
// Spawn new GenServer process
// ---------------------------------------------------------------------------
func Spawn(gs GenServer, args ...interface{}) (*Pid, error) {

	prefix := ""
	name := ""

	pid, err := SpawnPrefixName(gs, prefix, name, args...)

	return pid, err
}

func SpawnPrefixName(
	gs GenServer,
	prefix, name string,
	args ...interface{}) (*Pid, error) {

	options := gs.Options()

	chanSize, ok := options["chan-size"].(int)
	if !ok {
		chanSize = 100
	}

	pid, err := env.makePid(prefix, name)
	if err != nil {
		return nil, err
	}
	pid.inChan = make(chan interface{}, chanSize)
	pid.stopChan = make(chan *stopReq)

	initCh := make(chan *genInitReply)

	go GenServerLoop(gs, prefix, name, initCh, pid, args...)
	initReply := <-initCh

	if initReply.action == GenInitOk {
		return pid, nil
	}

	return nil, errors.New(initReply.reason)
}

// Register associates the name with pid
func Register(name string, pid *Pid) error {
	replyChan := make(chan bool)
	r := regNameReq{name: name, pid: pid, replyTo: replyChan}
	env.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%s' already registered", name)
}

func RegisterPrefix(prefix, name string, pid *Pid) error {
	replyChan := make(chan bool)
	r := regNameReq{prefix: prefix, name: name, pid: pid, replyTo: replyChan}
	env.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%s/%s' already registered", name, prefix)
}

// Unregister removes the registered name
func Unregister(name string) {
	replyChan := make(chan bool)
	r := unregNameReq{name: name, replyTo: replyChan}
	env.registry.unregNameChan <- r
	<-replyChan
}

func UnregisterPrefix(prefix, name string) {
	if prefix == "" && name == "" {
		return
	}

	replyChan := make(chan bool)
	r := unregNameReq{prefix: prefix, name: name, replyTo: replyChan}
	env.registry.unregNameChan <- r
	<-replyChan
}

// Whereis returns pid of registered process
func Whereis(name string) *Pid {
	pid := WhereisPrefix("", name)

	return pid
}

func WhereisPrefix(prefix, name string) *Pid {
	replyChan := make(chan *Pid)
	r := whereNameReq{prefix: prefix, name: name, replyTo: replyChan}
	env.registry.whereNameChan <- r
	pid := <-replyChan

	return pid
}

// Returns all pids with same prefix
func Whereare(prefix string) regMap {
	replyChan := make(chan regMap)
	r := wherePrefixReq{prefix: prefix, replyTo: replyChan}
	env.registry.wherePrefixChan <- r
	regs := <-replyChan

	return regs
}

// ---------------------------------------------------------------------------
func (n *act) run() {
	go n.registrator()
}

func (n *act) registrator() {
	for {
		select {
		case req := <-n.registry.makePidChan:
			n.serial += 1

			var newPid Pid
			newPid.id = n.serial

			var resp makePidResp
			resp.pid = &newPid

			// register name with prefix if not empty
			if req.name != "" {
				if _, ok := n.registered[req.prefix]; ok {
					// map with prefix exists
					if _, ok := n.registered[req.prefix][req.name]; ok {
						// name already registered
						resp.err = fmt.Errorf("name '%s/%s' already registered",
							req.prefix, req.name)
					} else {
						n.registered[req.prefix][req.name] = resp.pid
					}

				} else { // no maps with prefix
					n.registered[req.prefix] = make(regMap)
					n.registered[req.prefix][req.name] = resp.pid
				}
			}

			req.replyTo <- resp

		case req := <-n.registry.regNameChan:
			if _, ok := n.registered[req.prefix]; !ok {
				n.registered[req.prefix] = make(regMap)
			}
			if _, ok := n.registered[req.prefix][req.name]; ok {
				req.replyTo <- false // name registered
			} else {
				n.registered[req.prefix][req.name] = req.pid
				req.replyTo <- true
			}

		case req := <-n.registry.unregNameChan:
			if _, ok := n.registered[req.prefix]; ok {
				delete(n.registered[req.prefix], req.name)
			}
			req.replyTo <- true

		case req := <-n.registry.whereNameChan:
			req.replyTo <- n.registered[req.prefix][req.name]

		case req := <-n.registry.wherePrefixChan:
			var rpids regMap
			if pids, ok := n.registered[req.prefix]; ok {
				rpids = make(regMap)
				for k, v := range pids {
					rpids[k] = v
				}
			}
			req.replyTo <- rpids
		}
	}
}

func (n *act) makePid(p, a string) (*Pid, error) {
	replyChan := make(chan makePidResp)
	n.registry.makePidChan <- makePidReq{prefix: p, name: a, replyTo: replyChan}
	resp := <-replyChan

	return resp.pid, resp.err
}

func (pid *Pid) closePidChannels() {
	if pid != nil {
		close(pid.inChan)
		close(pid.stopChan)
	}
}

func (pid *Pid) Id() uint64 {
	if pid == nil {
		return 0
	}

	return pid.id
}
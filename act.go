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

type Opts struct {
	Prefix                   string
	Name                     interface{}
	Chan_size                uint32
	Return_pid_if_registered bool
}

type makePidResp struct {
	pid *Pid
	err error
}

type makePidReq struct {
	opts    *Opts
	replyTo chan<- makePidResp
}

type regNameReq struct {
	prefix  string
	name    interface{}
	pid     *Pid
	replyTo chan<- bool
}

type unregNameReq struct {
	prefix  string
	name    interface{}
	replyTo chan<- bool
}

type whereNameReq struct {
	prefix  string
	name    interface{}
	replyTo chan<- *Pid
}

type regMap map[interface{}]*Pid
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
func Spawn(gs GenServer, args ...interface{}) (pid *Pid, err error) {

	pid, err = SpawnOpts(gs, &Opts{}, args...)

	return
}

func SpawnPrefixName(
	gs GenServer,
	prefix string,
	name interface{},
	args ...interface{}) (pid *Pid, err error) {

	opts := &Opts{
		Prefix: prefix,
		Name:   name,
	}
	pid, err = SpawnOpts(gs, opts, args...)

	return
}

func SpawnOpts(gs GenServer, opts *Opts, args ...interface{}) (*Pid, error) {

	if opts.Chan_size == 0 {
		opts.Chan_size = 100
	}

	pid, err := env.makePid(opts)
	if err != nil {
		return nil, err
	}
	pid.inChan = make(chan interface{}, opts.Chan_size)
	pid.stopChan = make(chan *stopReq)

	initChan := make(chan Term)

	go GenServerLoop(gs, opts.Prefix, opts.Name, initChan, pid, args...)
	result := <-initChan

	switch result := result.(type) {
	case *GsInitOk:
	case *GsInitStop:
		return nil, errors.New(result.Reason)
	case error:
		return nil, result
	}

	return pid, nil
}

// Register associates the name with pid
func Register(name interface{}, pid *Pid) error {
	replyChan := make(chan bool)
	r := regNameReq{name: name, pid: pid, replyTo: replyChan}
	env.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%v' already registered", name)
}

func RegisterPrefix(prefix string, name interface{}, pid *Pid) error {
	replyChan := make(chan bool)
	r := regNameReq{prefix: prefix, name: name, pid: pid, replyTo: replyChan}
	env.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%s/%v' already registered", prefix, name)
}

// Unregister removes the registered name
func Unregister(name interface{}) {
	replyChan := make(chan bool)
	r := unregNameReq{name: name, replyTo: replyChan}
	env.registry.unregNameChan <- r
	<-replyChan
}

func UnregisterPrefix(prefix string, name interface{}) {
	if prefix == "" && name == nil {
		return
	}

	replyChan := make(chan bool)
	r := unregNameReq{prefix: prefix, name: name, replyTo: replyChan}
	env.registry.unregNameChan <- r
	<-replyChan
}

// Whereis returns pid of registered process
func Whereis(name interface{}) *Pid {
	pid := WhereisPrefix("", name)

	return pid
}

func WhereisPrefix(prefix string, name interface{}) *Pid {
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
			// n.serial += 1

			var newPid Pid
			newPid.id = n.serial + 1

			var resp makePidResp
			resp.pid = &newPid

			// register name with prefix if not empty
			if req.opts.Name != nil {
				if _, ok := n.registered[req.opts.Prefix]; ok {
					// map with prefix exists
					if pid, ok := n.registered[req.opts.Prefix][req.opts.Name]; ok {
						if req.opts.Return_pid_if_registered == true {
							resp.pid = pid
						} else {
							// name already registered
							resp.err =
								fmt.Errorf("name '%s/%v' already registered",
									req.opts.Prefix, req.opts.Name)
						}
					} else {
						n.registered[req.opts.Prefix][req.opts.Name] = resp.pid
					}

				} else { // no maps with prefix
					n.registered[req.opts.Prefix] = make(regMap)
					n.registered[req.opts.Prefix][req.opts.Name] = resp.pid
				}
			}

			if resp.err == nil {
				n.serial += 1
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

func (n *act) makePid(opts *Opts) (*Pid, error) {
	replyChan := make(chan makePidResp)
	n.registry.makePidChan <- makePidReq{opts: opts, replyTo: replyChan}
	resp := <-replyChan

	return resp.pid, resp.err
}

func (pid *Pid) Id() uint64 {
	if pid == nil {
		return 0
	}

	return pid.id
}

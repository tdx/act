package act

import (
	"errors"
	"flag"
	"fmt"
	"log"
)

//
// Term is a type for any values
//
type Term interface{}

//
// Pid incapsulates actor identificator and
// channels to communicate to actor process
//
type Pid struct {
	id       uint64
	inChan   chan interface{}
	stopChan chan *stopReq
}

//
// Opts - options for spawn actor process
//
type Opts struct {
	Prefix                string
	Name                  interface{}
	ChanSize              uint32
	ReturnPidIfRegistered bool
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
	prefix string
	name   interface{}
}

type whereNameReq struct {
	prefix  string
	name    interface{}
	replyTo chan<- *Pid
}

//
// RegMap map of registered processes with same prefix
//
type RegMap map[interface{}]*Pid
type wherePrefixReq struct {
	prefix  string
	replyTo chan<- RegMap
}

type registryChan struct {
	makePidChan     chan makePidReq
	regNameChan     chan regNameReq
	unregNameChan   chan unregNameReq
	whereNameChan   chan whereNameReq
	wherePrefixChan chan wherePrefixReq
}

type Act struct {
	serial     uint64
	registry   *registryChan
	registered map[string]RegMap
}

// ---------------------------------------------------------------------------
var nTrace bool
var env *Act // default env

func init() {
	flag.BoolVar(&nTrace, "trace", false, "trace actors")

	env = NewEnv()
}

func NewEnv() *Act {

	registry := &registryChan{
		makePidChan:     make(chan makePidReq),
		regNameChan:     make(chan regNameReq),
		unregNameChan:   make(chan unregNameReq),
		whereNameChan:   make(chan whereNameReq),
		wherePrefixChan: make(chan wherePrefixReq),
	}

	a := &Act{
		registry:   registry,
		registered: make(map[string]RegMap),
	}

	// without prefix
	a.registered[""] = make(RegMap)

	a.run()

	return a
}

func nLog(f string, a ...interface{}) {
	if nTrace {
		log.Printf(f, a...)
	}
}

//
// Spawn spawns a new GenServer process
//
func Spawn(gs GenServer, args ...interface{}) (*Pid, error) {

	pid, err := env.Spawn(gs, args...)

	return pid, err
}

func (a *Act) Spawn(gs GenServer, args ...interface{}) (pid *Pid, err error) {

	pid, err = a.SpawnOpts(gs, &Opts{}, args...)

	return
}

//
// SpawnPrefixName spawns a new GenServer process with given prefix and name
//
func SpawnPrefixName(
	gs GenServer,
	prefix string,
	name interface{},
	args ...interface{}) (*Pid, error) {

	return env.SpawnPrefixName(gs, prefix, name, args...)
}

func (a *Act) SpawnPrefixName(
	gs GenServer,
	prefix string,
	name interface{},
	args ...interface{}) (pid *Pid, err error) {

	opts := &Opts{
		Prefix: prefix,
		Name:   name,
	}
	pid, err = a.SpawnOpts(gs, opts, args...)

	return
}

//
// SpawnOpts spawns a new GenServer process with given opts
//
func SpawnOpts(gs GenServer, opts *Opts, args ...interface{}) (*Pid, error) {
	return env.SpawnOpts(gs, opts, args...)
}

func (a *Act) SpawnOpts(
	gs GenServer,
	opts *Opts,
	args ...interface{},
) (*Pid, error) {

	if opts.ChanSize == 0 {
		opts.ChanSize = 100
	}

	pid, err := a.makePid(opts)
	if err != nil {
		return nil, err
	}

	initChan := make(chan Term)

	go a.GenServerLoop(gs, opts.Prefix, opts.Name, initChan, pid, args...)
	result := <-initChan

	switch result := result.(type) {
	case gsInitOk:
	case *GsInitStop:
		return nil, errors.New(result.Reason)
	case error:
		return nil, result
	}

	return pid, nil
}

//
// Register associates the name with pid
//
func Register(name interface{}, pid *Pid) error {
	return env.Register(name, pid)
}

func (a *Act) Register(name interface{}, pid *Pid) error {
	replyChan := make(chan bool, 1)
	r := regNameReq{name: name, pid: pid, replyTo: replyChan}
	a.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%v' already registered", name)
}

//
// RegisterPrefix associates the prefix + name with pid
//
func RegisterPrefix(prefix string, name interface{}, pid *Pid) error {
	return env.RegisterPrefix(prefix, name, pid)
}

func (a *Act) RegisterPrefix(prefix string, name interface{}, pid *Pid) error {
	replyChan := make(chan bool, 1)
	r := regNameReq{prefix: prefix, name: name, pid: pid, replyTo: replyChan}
	a.registry.regNameChan <- r
	reply := <-replyChan

	if reply == true {
		return nil
	}

	return fmt.Errorf("name '%s/%v' already registered", prefix, name)
}

//
// Unregister removes the registered name
//
func Unregister(name interface{}) {
	env.Unregister(name)
}

func (a *Act) Unregister(name interface{}) {
	r := unregNameReq{name: name}
	a.registry.unregNameChan <- r
}

//
// UnregisterPrefix removes the registered name
//
func UnregisterPrefix(prefix string, name interface{}) {
	env.UnregisterPrefix(prefix, name)
}

func (a *Act) UnregisterPrefix(prefix string, name interface{}) {
	if prefix == "" && name == nil {
		return
	}

	r := unregNameReq{prefix: prefix, name: name}
	a.registry.unregNameChan <- r
}

//
// Whereis returns pid of registered process
//
func Whereis(name interface{}) *Pid {
	return env.Whereis(name)
}

func (a *Act) Whereis(name interface{}) *Pid {
	pid := a.WhereisPrefix("", name)

	return pid
}

//
// WhereisPrefix returns pid of registered process
//
func WhereisPrefix(prefix string, name interface{}) *Pid {
	return env.WhereisPrefix(prefix, name)
}

func (a *Act) WhereisPrefix(prefix string, name interface{}) *Pid {
	replyChan := make(chan *Pid, 1)
	r := whereNameReq{prefix: prefix, name: name, replyTo: replyChan}
	a.registry.whereNameChan <- r
	pid := <-replyChan

	return pid
}

//
// Whereare returns all pids with same prefix
//
func Whereare(prefix string) RegMap {
	return env.Whereare(prefix)
}

func (a *Act) Whereare(prefix string) RegMap {
	replyChan := make(chan RegMap, 1)
	r := wherePrefixReq{prefix: prefix, replyTo: replyChan}
	a.registry.wherePrefixChan <- r
	regs := <-replyChan

	return regs
}

// ---------------------------------------------------------------------------
func (a *Act) run() {
	go a.registrator()
}

func (a *Act) registrator() {
	for {

		//
		// first handle new registation
		//
		select {
		case req := <-a.registry.makePidChan:
			a.regNewPid(&req)
			continue
		default:
		}

		select {
		case req := <-a.registry.makePidChan:
			a.regNewPid(&req)
		case req := <-a.registry.regNameChan:
			if _, ok := a.registered[req.prefix]; !ok {
				a.registered[req.prefix] = make(RegMap)
			}
			if _, ok := a.registered[req.prefix][req.name]; ok {
				req.replyTo <- false // name registered
			} else {
				a.registered[req.prefix][req.name] = req.pid
				req.replyTo <- true
			}

		case req := <-a.registry.unregNameChan:
			if _, ok := a.registered[req.prefix]; ok {
				delete(a.registered[req.prefix], req.name)
			}

		case req := <-a.registry.whereNameChan:
			req.replyTo <- a.registered[req.prefix][req.name]

		case req := <-a.registry.wherePrefixChan:
			var rpids RegMap
			if pids, ok := a.registered[req.prefix]; ok {
				rpids = make(RegMap)
				for k, v := range pids {
					rpids[k] = v
				}
			}
			req.replyTo <- rpids
		}
	}
}

func (a *Act) regNewPid(req *makePidReq) {

	newPidCreated := true

	var resp makePidResp
	resp.pid = &Pid{id: a.serial + 1}

	// register name with prefix if not empty
	if req.opts.Name != nil {
		if _, ok := a.registered[req.opts.Prefix]; ok {
			// map with prefix exists
			if pid, ok := a.registered[req.opts.Prefix][req.opts.Name]; ok {

				newPidCreated = false

				if req.opts.ReturnPidIfRegistered == true {
					resp.pid = pid
				} else {
					// name already registered
					resp.pid = nil
					resp.err =
						fmt.Errorf("name '%s/%v' already registered",
							req.opts.Prefix, req.opts.Name)
				}
			} else {
				a.registered[req.opts.Prefix][req.opts.Name] = resp.pid
			}

		} else { // no maps with prefix
			a.registered[req.opts.Prefix] = make(RegMap)
			a.registered[req.opts.Prefix][req.opts.Name] = resp.pid
		}
	}

	if newPidCreated {
		resp.pid.inChan = make(chan interface{}, req.opts.ChanSize)
		resp.pid.stopChan = make(chan *stopReq)
		a.serial++
	}

	req.replyTo <- resp
}

func (a *Act) makePid(opts *Opts) (*Pid, error) {
	replyChan := make(chan makePidResp, 1)
	a.registry.makePidChan <- makePidReq{opts: opts, replyTo: replyChan}
	resp := <-replyChan

	return resp.pid, resp.err
}

//
// Id returns process identificator
//
func (pid *Pid) Id() uint64 {
	if pid == nil {
		return 0
	}

	return pid.id
}

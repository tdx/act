package act

import (
	"fmt"
	"log"
)

//
// GenServerImpl is the default implementation of GenServer interface
//
type GenServerImpl struct {
	GenServer
	self   *Pid // Pid of process
	prefix string
	name   interface{} // Registered name of process
}

//
// Init initializes process state using arbitrary arguments
//
func (gs *GenServerImpl) Init(args ...interface{}) Term {

	log.Printf("GenServerImpl:Init : pid: #%d, reg name: %s/%s, args: %#v",
		gs.Self().Id(), gs.Prefix(), gs.NameStr(), args)

	return GsInitOk
}

//
// HandleCast handles incoming messages from `pid.Cast(data)`
//
func (gs *GenServerImpl) HandleCast(req Term) Term {

	log.Printf("GenServerImpl:HandleCast : %#v", req)

	return GsCastNoReply
}

//
// HandleCall handles incoming messages from `pid.Call(data, from)`
//
func (gs *GenServerImpl) HandleCall(req Term, from From) Term {

	log.Printf("GenServerImpl:HandleCall : %#v, From: %#v", req, from)

	return &GsCallReply{"ok"}
}

//
// Terminate called when process died
//
func (gs *GenServerImpl) Terminate(reason string) {
	log.Printf("GenServerImpl:Terminate : %#v", reason)
}

func (gs *GenServerImpl) setPid(pid *Pid) {
	gs.self = pid
}

func (gs *GenServerImpl) setPrefix(prefix string) {
	gs.prefix = prefix
}

func (gs *GenServerImpl) setName(name interface{}) {
	gs.name = name
}

func (gs *GenServerImpl) Self() *Pid {
	return gs.self
}

func (gs *GenServerImpl) Prefix() string {
	return gs.prefix
}

func (gs *GenServerImpl) Name() interface{} {
	return gs.name
}

func (gs *GenServerImpl) NameStr() string {
	switch name := gs.name.(type) {
	case string:
		return name
	default:
		if name == nil {
			return ""
		} else {
			return fmt.Sprintf("%v", name)
		}
	}
}

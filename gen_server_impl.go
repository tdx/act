package act

import (
	"fmt"
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

	return GsInitOk
}

//
// HandleCast handles incoming messages from `pid.Cast(data)`
//
func (gs *GenServerImpl) HandleCast(req Term) Term {

	return GsCastNoReply
}

//
// HandleCall handles incoming messages from `pid.Call(data)`
//
func (gs *GenServerImpl) HandleCall(req Term, from From) Term {

	return GsCallReplyOk
}

//
// Terminate called when process died
//
func (gs *GenServerImpl) Terminate(reason string) {
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

//
// Self returns pid of the process
//
func (gs *GenServerImpl) Self() *Pid {
	return gs.self
}

//
// Id returns id of the process
//
func (gs *GenServerImpl) Id() uint64 {
	return gs.self.Id()
}

//
// Prefix returns prefix of the process
//
func (gs *GenServerImpl) Prefix() string {
	return gs.prefix
}

//
// Name returns name of the process
//
func (gs *GenServerImpl) Name() interface{} {
	return gs.name
}

//
// NameStr return name of the process as a string
//
func (gs *GenServerImpl) NameStr() string {
	switch name := gs.name.(type) {
	case string:
		return name
	default:
		if name == nil {
			return ""
		}
		return fmt.Sprintf("%v", name)
	}
}

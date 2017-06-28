package act

import (
	"log"
)

//
// GenServerImpl is the default implementation of GenServer interface
//
type GenServerImpl struct {
	GenServer
	self         *Pid   // Pid of process
	prefix, name string // Registered name of process
}

//
// Init initializes process state using arbitrary arguments
//
func (gs *GenServerImpl) Init(
	args ...interface{}) (

	action GenInitAction,
	stopReason string) {

	log.Printf("GenServerImpl:Init : pid: #%d, reg name: %s/%s, args: %#v",
		gs.Self().Id(), gs.Prefix(), gs.Name(), args)

	return GenInitOk, ""
}

//
// HandleCast handles incoming messages from `pid.Cast(data)`
//
func (gs *GenServerImpl) HandleCast(
	req Term) (

	action GenCastAction,
	stopReason string) {

	log.Printf("GenServerImpl:HandleCast : %#v", req)

	return GenCastNoreply, ""
}

//
// HandleCall handles incoming messages from `pid.Call(data, from)`
//
func (gs *GenServerImpl) HandleCall(
	req Term,
	from From) (

	action GenCallAction,
	reply Term,
	stopReason string) {

	log.Printf("GenServerImpl:HandleCall : %#v, From: %#v", req, from)

	return GenCallReply, "ok", ""
}

//
// Terminate called when process died
//
func (gs *GenServerImpl) Terminate(reason string) {
	log.Printf("GenServerImpl:Terminate : %#v", reason)
}

//
// Options returns map of default process-related options
//
func (gs *GenServerImpl) Options() map[string]interface{} {
	return map[string]interface{}{
		"chan-size": 100, // size of channel for messages
	}
}

func (gs *GenServerImpl) setPid(pid *Pid) {
	gs.self = pid
}

func (gs *GenServerImpl) setPrefix(prefix string) {
	gs.prefix = prefix
}

func (gs *GenServerImpl) setName(name string) {
	gs.name = name
}

func (gs *GenServerImpl) Self() *Pid {
	return gs.self
}

func (gs *GenServerImpl) Prefix() string {
	return gs.prefix
}

func (gs *GenServerImpl) Name() string {
	return gs.name
}

# act - Go actors library

[![][go-report-svg]][go-report-url] [![][travis-svg]][travis-url] [![][coveralls-svg]][coveralls-url]

## How to use

### Define the actor

Each actor is a separate process that implements the `GenServer` interface and can contain a state.

```go

import (
  "github.com/tdx/act"
  "log"
)

type gs struct {
	GenServerImpl
	// some state variables
}

//
// Init initializes process state using arbitrary arguments
//
func (s *gs) Init(args ...interface{}) Term {
	return act.GsIntiOk
}

//
// HandleCall handles incoming synchronous messages from `pid.Call(data)`
//
func (s *gs) HandleCall(req act.Term, from From) Term {
	log.Printf("HandleCall : %#v, From: %#v", req, from)

	return act.GsCallReplyOk
}

//
// HandleCast handles incoming asynchronous messages from `pid.Cast(data)`
//
func (s *gs) HandleCast(req act.Term) Term {
	log.Printf("HandleCast : %#v", req)

	return act.GsCastNoReply
}

//
// Terminate called when process died
//
func (s *gs) Terminate(reason string) {
	log.Printf("Terminate : %s", reason)
}

```

### Run the actor

```go

func Start() (*act.Pid, error) {
	gs := new(gs)
	pid, err := act.Spawn(gs)
	return pid, err
}
```

### Interact with the actor

There are two ways to send a message to the actor.
Synchronous `Call` and asynchronous `Cast`.

```go
	pid, err := Start()
	if err != nil {
		return
	}

	err = pid.Cast(...)

	result, err := pid.Call(...)
```

### Stop the actor

The actor can be stopped from outside

```go
	err := pid.Stop()
```

or in one of the callbacks.

```go
func (s *gs) HandleCast(req act.Term) Term {
	// if something goes wrong
	return &act.GsCastStop{"reason to stop"}
}

func (s *gs) HandleCall(req act.Term, from From) Term {
	// if something goes wrong
	return &act.GsCallStop{"reason to stop", someValueToReturnToCaller}
}

```

Before the actor process stopped `Terminate` callback is called.

## Process registry

Process registry stores pid associations with a given name.
To accociate a pid with a name you can use `SpawnOpts`

```go
	gs := new(gs)
	opts := &act.Opts{
		Name: "some name",  // Name is interface{}, so can be complicated
	}
	pid, err := act.SpawnOpts(gs, opts)
```

or register pid in one of the callbacks.

```go
func (s *gs) Init(args ...interface{}) Term {

	act.Register(someUsefulData, s.Self())

	return act.GsIntiOk
}
```

You can get a pid by name from any part of the application and use this pid to interact with actor.

```go
	pid := act.Whereis("some name")
	if pid != nil {
		// use pid
	}
```

When registering a process name, you can place it in a group.
Later you can get a list of processes in the group.

```go
	gs1 := new(gs)
	opts1 := &act.Opts{
		Prefix: "group1",
		Name: "name1",
	}
	pid1, err := act.SpawnOpts(gs1, opts1)

	gs2 := new(gs)
	opts2 := &act.Opts{
		Prefix: "group1",
		Name: "name2",
	}
	pid2, err := act.SpawnOpts(gs2, opts2)

	//
	for name, pid := range act.Whereare("group1") {
	}
```

## Timer

A timer is used to send a message to the actor after an arbitrary period of time.
Time is always measured in **milliseconds**.
Timer messages fall into the `HandleCast` handler.

```go
func SendDelayedNotify(pid *act.Pid) {
	pid.SendAfter("Hello", 100)
}

//
func (s *gs) HandleCast(req act.Term) act.Term {
	switch req := req.(type) {
	case string:
		if req == "Hello" {
			//
		}
	}
}
```

### Actor timeouts

Actor can set the inactivity timeout. After the timeout occurs, the process will receive an `act.GsTimeout` message.

```go
func (s *gs) Init(args ...interface{}) act.Term {
	// some initialization
	return &act.GsInitTimeout{30000}
}

func (s *gs) HandleCast(req act.Term) act.Term {
	switch req := req.(type) {
	//...
	case act.GsTimeout:
		return &act.GsCastStop{"session expired"}
	//...
	}
}
```

The inactivity timer can be set in any message handler.
`HandleCast` returns `&act.GsCastNoReplyTmeout{}`.
`HandleCall` returns `&act.GsCallReplyTimeout{}` or `&act.GsCallNoReplyTimeout{}`.


[go-report-url]: https://goreportcard.com/report/github.com/tdx/act
[go-report-svg]: https://goreportcard.com/badge/github.com/tdx/act

[travis-url]: https://travis-ci.org/tdx/act
[travis-svg]: https://travis-ci.org/tdx/act.svg?branch=master

[coveralls-url]: https://coveralls.io/github/tdx/act?branch=master
[coveralls-svg]: https://coveralls.io/repos/github/tdx/act/badge.svg?branch=master

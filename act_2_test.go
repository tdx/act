package act

import (
	"fmt"
	"testing"
)

//
// gen_server
//
type gs2 struct {
	GenServerImpl
	i int
}

//
// callbacks
//
func (s *gs2) Init(args ...interface{}) Term {
	s.i = 10

	return GsInitOk
}

func (s *gs2) HandleCall(req Term, from From) Term {
	switch req := req.(type) {
	case *reqGetI:
		req.i = s.i
	case *reqSetI:
		s.i = req.i
	default:
		return fmt.Errorf("unexpected call: %#v\n", req)
	}

	return GsCallReplyOk
}

//
// messages
//
type reqGetI struct {
	i int
}

type reqSetI struct {
	i int
}

//
// api
//
func getI(pid *Pid) (int, error) {
	r := &reqGetI{}
	_, err := pid.Call(r)
	if err != nil {
		return 0, err
	}

	return r.i, nil
}

func setI(pid *Pid, i int) error {
	r := &reqSetI{i: i}
	_, err := pid.Call(r)
	return err
}

func TestRegistered(t *testing.T) {

	gs := new(gs2)

	opts := &Opts{
		Name: "ttt.name",
		ReturnPidIfRegistered: true}

	pid, err := SpawnOpts(gs, opts)
	if err != nil {
		t.Fatal(err)
	}
	if opts.ReturnedRegisteredPid == true {
		t.Fatalf("retured registered pid!")
	}

	i := 20
	if err := setI(pid, i); err != nil {
		t.Fatal(err)
	}

	i, err = getI(pid)
	if err != nil {
		t.Fatal(err)
	}
	if i != 20 {
		t.Fatalf("expected i 20, got %d\n", i)
	}

	opts2 := &Opts{
		Name: "ttt.name",
		ReturnPidIfRegistered: true}

	pid2, err := SpawnOpts(gs, opts2)
	if err != nil {
		t.Fatal(err)
	}
	if opts2.ReturnedRegisteredPid != true {
		t.Fatalf("pid must be registered!")
	}
	if pid.Id() != pid2.Id() {
		t.Fatalf("expired pid id %d, got %d\n", pid.Id(), pid2.Id())
	}

	//
	// check gs2.i has not changed
	//
	i2, err := getI(pid2)
	if err != nil {
		t.Fatal(err)
	}
	if i2 != i {
		t.Fatalf("expected i %d, got %d\n", i, i2)
	}
}

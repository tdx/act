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

func TestSpawnOrLocate(t *testing.T) {

	gs := new(gs2)

	opts := &Opts{Name: "ttt.name"}

	pid, newPid, err := SpawnOrLocate(gs, opts)
	if err != nil {
		t.Fatal(err)
	}
	if newPid != true {
		t.Fatalf("returned registered pid, expected new process!")
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
		t.Fatalf("expected i=20, got %d\n", i)
	}

	opts2 := &Opts{Name: "ttt.name"}

	pid2, newPid2, err := SpawnOrLocate(gs, opts2)
	if err != nil {
		t.Fatal(err)
	}
	if newPid2 == true {
		t.Fatalf("pid must be registered!")
	}
	if pid.Id() != pid2.Id() {
		t.Fatalf("expected pid id=%d, got %d\n", pid.Id(), pid2.Id())
	}

	//
	// check gs2.i has not changed
	//
	i2, err := getI(pid2)
	if err != nil {
		t.Fatal(err)
	}
	if i2 != i {
		t.Fatalf("expected i=%d, got %d\n", i, i2)
	}
}

func TestSpawnOrLocate2(t *testing.T) {

	gs := new(gs2)

	opts := &Opts{}

	_, _, err := SpawnOrLocate(gs, opts)
	if err == nil {
		t.Fatalf("name for regisration is empty - must fail")
	}
}

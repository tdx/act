package act

import (
	"testing"
)

func TestPidId(t *testing.T) {
	var pid *Pid

	if pid.Id() != 0 {
		t.Errorf("nil.Pid() %d != 0", pid.Id())
	}
}

//
// Registrator
//
func TestRegister(t *testing.T) {
	start_server(t)

	prefix := "gsGroup"
	name := "test_name"

	err := Register(name, pid)
	if err != nil {
		t.Error(err)
	}

	pid1 := Whereis(name)
	if pid1 == nil {
		t.Errorf("process '%s' not registered", name)
	}

	if pid.Id() != pid1.Id() {
		t.Error("process with same name have different pids")
	}

	Unregister(name)
	pid1 = Whereis(name)
	if pid1 != nil {
		t.Errorf("process '%s' still registered", name)
	}

	err = RegisterPrefix(prefix, name, pid)
	if err != nil {
		t.Error(err)
	}

	pid2 := WhereisPrefix(prefix, name)
	if pid2 == nil {
		t.Errorf("process '%s/%s' not registered", prefix, name)
	}

	if pid.Id() != pid2.Id() {
		t.Error("process with same name have different pids")
	}

	UnregisterPrefix(prefix, name)
	pid2 = WhereisPrefix(prefix, name)
	if pid2 != nil {
		t.Errorf("process '%s/%s' still registered", prefix, name)
	}

	//
	err = Register(name, pid)
	if err != nil {
		t.Error(err)
	}

	err = Register(name, pid)
	if err == nil {
		t.Error("can not register two pids with same name")
	}

	Unregister(name)
	pid1 = Whereis(name)
	if pid1 != nil {
		t.Errorf("process '%s' still registered", name)
	}

	//
	err = RegisterPrefix(prefix, name, pid)
	if err != nil {
		t.Error(err)
	}

	err = RegisterPrefix(prefix, "name2", pid)
	if err != nil {
		t.Error(err)
	}

	pids := Whereare(prefix)
	if len(pids) != 2 {
		t.Errorf("Whereare(%s) %d != 2", prefix, len(pids))
	}
}

func TestRegisterInSpawn(t *testing.T) {
	prefix := "gsGroup"
	name := "proc1"

	_, err := start_prefix(prefix, name)
	if err != nil {
		t.Error(err)
	}

	pid1 := Whereis(name)
	if pid1 != nil {
		t.Errorf("name '%s' must not be registered", name)
	}

	pid1 = WhereisPrefix(prefix, name)
	if pid1 == nil {
		t.Errorf("name '%s' must be registered", name)
	}

	Unregister(name)
	pid1 = WhereisPrefix(prefix, name)
	if pid1 == nil {
		t.Errorf("name '%s' must be registered", name)
	}

	err = RegisterPrefix(prefix, name, pid1)
	if err == nil {
		t.Error("can not register two pids with same prefix and name")
	}

	_, err = start_prefix(prefix, name)
	if err == nil {
		t.Error("can not spqsn two process with same prefix and name")
	}

	_, err = start_prefix("newGroup", name)
	if err != nil {
		t.Error(err)
	}
}

func TestRegisterInOpts(t *testing.T) {
	prefix := "gsGroup"
	name := "proc2"

	opts := &Opts{
		Prefix: prefix,
		Name:   name,
	}

	s := new(gsi)
	_, err := SpawnOpts(s, opts)
	if err != nil {
		t.Fatal(err)
	}

	pid := WhereisPrefix(prefix, name)
	if pid == nil {
		t.Fatal("pid must be registered")
	}

	pid.Stop()
}

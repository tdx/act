package act

import (
	"testing"
)

type gsi struct {
	GenServerImpl
}

//
// gsi has default implementation of GenServer interface
//
func TestGenServerImpl(t *testing.T) {
	pid, err := Spawn(new(gsi))

	if err != nil {
		t.Fatal(err)
	}

	_, err = pid.Call(1)
	if err != nil {
		t.Error(err)
	}

	err = pid.Cast(1)
	if err != nil {
		t.Error(err)
	}

	err = pid.Stop()
	if err != nil {
		t.Error(err)
	}
}

func TestGenServerImplReg(t *testing.T) {

	s := new(gsi)

	pid, err := SpawnPrefixName(s, "test", [2]byte{1, 3})
	if err != nil {
		t.Fatal(err)
	}

	_, err = pid.Call(1)
	if err != nil {
		t.Error(err)
	}

	err = pid.Cast(1)
	if err != nil {
		t.Error(err)
	}

	pid1 := WhereisPrefix("test", s.Name())
	if pid1 == nil {
		t.Error("server must be registered")
	}

	if pid.Id() != pid1.Id() {
		t.Error("wrong pid id: want %v - got %v", pid.Id(), pid1.Id())
	}

	err = pid.Stop()
	if err != nil {
		t.Error(err)
	}
}

// name is string
func TestGenServerImplReg2(t *testing.T) {

	s := new(gsi)

	pid, err := SpawnPrefixName(s, "test", "name")
	if err != nil {
		t.Fatal(err)
	}

	_, err = pid.Call(1)
	if err != nil {
		t.Error(err)
	}

	err = pid.Cast(1)
	if err != nil {
		t.Error(err)
	}

	pid1 := WhereisPrefix("test", s.Name())
	if pid1 == nil {
		t.Error("server must be registered")
	}

	if pid.Id() != pid1.Id() {
		t.Error("wrong pid id: want %v - got %v", pid.Id(), pid1.Id())
	}

	err = pid.Stop()
	if err != nil {
		t.Error(err)
	}

}

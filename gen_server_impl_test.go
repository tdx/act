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
	s := new(gsi)
	pid, err := Spawn(s)

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

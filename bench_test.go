package act

import (
	"fmt"
	"testing"
)

// ----------------------------------------------------------------------------
// Benchmarks
// ----------------------------------------------------------------------------
func BenchmarkStartServer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		run_server()
	}
}

func BenchmarkStartRegisteredServer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		start_prefix("gsGroup", fmt.Sprintf("proc_%d", i))
	}
}

func BenchmarkCallPid(b *testing.B) {
	pid, err := run_server()
	if err == nil {
		req := &reqInc{}
		for i := 0; i < b.N; i++ {
			pid.Call(req)
		}
	}
}

func BenchmarkCallName(b *testing.B) {
	prefix := "gsGroup"
	name := "test_proc_123"
	opts := &Opts{
		Prefix: prefix,
		Name:   name,
		Return_pid_if_registered: true}

	start_server_opts(b, opts)

	var pid *Pid
	req := &reqInc{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid = WhereisPrefix(prefix, name)
		if pid == nil {
			b.Fatal("pid == nil")
		}
		pid.Call(req)
	}
}

func BenchmarkCastPid(b *testing.B) {
	pid, err := run_server()
	if err == nil {
		for i := 0; i < b.N; i++ {
			pid.Cast(cmdTest)
		}
	}
}

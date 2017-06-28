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

func BenchmarkCall(b *testing.B) {
	pid, err := run_server()
	if err == nil {
		req := &reqInc{}
		for i := 0; i < b.N; i++ {
			pid.Call(req)
		}
	}
}

func BenchmarkCast(b *testing.B) {
	pid, err := run_server()
	if err == nil {
		for i := 0; i < b.N; i++ {
			pid.Cast(cmdTest)
		}
	}
}

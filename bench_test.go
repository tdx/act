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
		runServer()
	}
}

func BenchmarkStartRegisteredServer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		startPrefix("gsGroup", fmt.Sprintf("proc_%d", i))
	}
}

func BenchmarkCallPid(b *testing.B) {
	pid, err := runServer()
	if err == nil {

		req := &reqInc{}
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pid.Call(req)
			}
		})
	}
}

func BenchmarkCallName(b *testing.B) {

	prefix := "gsGroup"
	name := "test_proc_123"
	opts := &Opts{
		Prefix: prefix,
		Name:   name,
		ReturnPidIfRegistered: true}

	startServerOpts(b, opts)

	var pid *Pid
	req := &reqInc{}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pid = WhereisPrefix(prefix, name)
			if pid == nil {
				b.Fatal("pid == nil")
			}
			pid.Call(req)
		}
	})
}

func BenchmarkCastPid(b *testing.B) {
	pid, err := runServer()
	if err == nil {

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pid.Cast(cmdTest)
			}
		})
	}
}

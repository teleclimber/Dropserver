package leaktest

// borrowed from https://github.com/anacrolix/missinggo/blob/master/leaktest/goleaktest.go

import (
	"runtime"
	"testing"
	"time"

	"github.com/bradfitz/iter"
)

// GoroutineLeakCheck compares the number of goroutines at he beginning og a test to the end.
// Put GoroutineLeakCheck(t)() at the top of your test. Make sure the
// goroutine count is steady before your test begins.
func GoroutineLeakCheck(t testing.TB) func() {
	numStart := runtime.NumGoroutine()
	return func() {
		var numNow int
		wait := time.Millisecond
		//started := time.Now()
		for range iter.N(10) { // 1 second
			numNow = runtime.NumGoroutine()
			if numNow <= numStart {
				break
			}
			//t.Errorf("%d excess goroutines after %s", numNow-numStart, time.Since(started))
			time.Sleep(wait)
			wait *= 2
		}

		if numNow > numStart {
			t.Errorf("More goroutines thn when we started: %d to %d", numStart, numNow)
		}
		// I'd print stacks, or treat this as fatal, but I think
		// runtime.NumGoroutine is including system routines for which we are
		// not provided the stacks, and are spawned unpredictably.
		//t.Logf("have %d goroutines, started with %d", numNow, numStart)
		// select {}
	}
}

package oom

import (
	"net/http"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/pressly/chi"
	"golang.org/x/net/context"
)

var terminating int32

// Selfdestruct is a middleware that terminates current process if system
// memory use exceeds the threshold set.
//
// Most common use case is automatic restart of leaky *cough*GCO*cough service
// all you need is this middleware and something that will start the process
// again after it's been terminated (docker --restart=always for example )
//
// OOMSelfdestruct accepts one parameter - a float64 defining a fraction of the
// system memory that can be used before process self-terminates using system
// signal (SIG_TERM).
// As long as the app handles shutdown gracefully on SIG_TERM it should just work
//
// Right now it relies on memory usage data provided by /proc/meminfo so it is
// Linux specific. For convenience (dev envs) it will do nothing on other archs
func Selfdestruct(limit float64) func(chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if MemoryUsage() > limit {
				go selfdestruct(ctx, SignalSelfdestructGroup)
			}

			next.ServeHTTPC(ctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

// SelfdestructFn is a version of Selfdestruct that accepts a custom
// termination function ( func(context.Context) bool ) that handles the process
// shutdown.
// Use it if your app requires extra steps to gracefully shut down.
func SelfdestructFn(limit float64, fn func(ctx context.Context) bool) func(chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if MemoryUsage() > limit {
				go selfdestruct(ctx, fn)
			}

			next.ServeHTTPC(ctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

func selfdestruct(ctx context.Context, fn func(ctx context.Context) bool) {
	// if it's already terminating do nothing
	if !atomic.CompareAndSwapInt32(&terminating, 0, 1) {
		return
	}
	if !fn(ctx) {
		// selfdestruct failed, give future requests a chance to try again
		atomic.SwapInt32(&terminating, 0)
	}
}

// SignalSelfdestructGroup sends SIGTERM to whole process group
func SignalSelfdestructGroup(ctx context.Context) bool {
	gpid, err := syscall.Getpgid(os.Getpid())
	if err == nil {
		if err = syscall.Kill(gpid, syscall.SIGTERM); err == nil {
			return true
		}
	}
	return false
}

// SignalSelfdestructProcess sends SIGTERM to current process
func SignalSelfdestructProcess(ctx context.Context) bool {
	proc, err := os.FindProcess(os.Getpid())
	if err == nil {
		if err = proc.Signal(syscall.SIGTERM); err == nil {
			return true
		}
	}
	return false
}

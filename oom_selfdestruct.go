package selfdestruct

import (
	"net/http"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/pressly/chi"
	"golang.org/x/net/context"
)

var terminating int32

// OOMSelfdestruct is a middleware that terminates current process if system
// memory use exceeds the threshold set.
//
// Most common use case is automatic restart of leaky *cough*GCO*cough service
// all you need is this middleware and something that will start the process
// again after it's been terminated (docker --restart=always for example )
//
// OOMSelfdestruct accepts two parameters, second one is optional:
// - a float64 defining a fraction of the system memory that can be used before
// process self-terminates
// - optional shutdown function taking context.Context as parameter and
// returining bool - false indicates failure
// If shutdown function is not provided it uses system signals to initialize
// selfdestruct, so any graceful shutdown you have in place should just work
//
// Right now it relies on memory usage data provided by /proc/meminfo so it is
// Linux specific. For convenience (dev envs) it will do nothing on other archs
func OOMSelfdestruct(limit float64, fn ...func(ctx context.Context) bool) func(chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if MemoryUsage() > limit {
				go selfdestruct(ctx, fn...)
			}

			next.ServeHTTPC(ctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

func selfdestruct(ctx context.Context, fn ...func(ctx context.Context) bool) {
	// if it's already terminating do nothing
	if !atomic.CompareAndSwapInt32(&terminating, 0, 1) {
		return
	}
	if len(fn) > 0 {
		if !fn[0](ctx) {
			// selfdestruct failed, give future requests a chance to try again
			atomic.CompareAndSwapInt32(&terminating, 1, 0)
		}
		return
	}

	proc, err := os.FindProcess(os.Getpid())
	if err == nil {
		if err = proc.Signal(syscall.SIGTERM); err == nil {
			return
		}
	}
	// selfdestruct failed, give future requests a chance to try again
	atomic.CompareAndSwapInt32(&terminating, 1, 0)
}

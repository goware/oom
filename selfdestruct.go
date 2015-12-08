package selfdestruct

import (
	"net/http"
	"runtime"
	"syscall"

	"github.com/pressly/chi"
	"golang.org/x/net/context"
)

// OOMKiller is a middleware that cancels incoming requests and returns a 503 HTTP
// code if system memory use exceeds the threshold set.
// order).
//
// This middleware should be inserted fairly early in the middleware stack to
// ensure that request is cancelled early
//
// OOMKiller accepts a single parameter - a float64 defining a fraction of the
// memory process can use before it starts cancelling requests
//
// Right now it relies on total memory allocated by current process - it would
// have been better to use total available system memory, but reading /proc/meminfo
// is not portable

// TODO: second parameter should be a shutdown function.
// ie. when we register the middleware, we can pass an app.Shutdown() function
// to this hook to support graceful shutdowns.
// func MemoryLimit(limit float64, shutdownFn func()) func(chi.Handler) chi.Handler {

func MemoryLimit(limit float64) func(chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			si := &syscall.Sysinfo_t{} // Btw, I get undefined syscall.Sysinfo_t on darwin64 / go1.5.2
			me := &runtime.MemStats{}
			runtime.ReadMemStats(me)
			syscall.Sysinfo(si)

			if float64(me.Alloc)/float64(si.Totalram) > limit {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTPC(ctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

// TODO: only check the memory every X seconds. If we've already checked
// in that period, then just continue with the next handler.

// TODO: check that the memory limit has been reached Y times in a row
// before calling the exit function

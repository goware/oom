package oom

import (
	"net/http"

	"github.com/pressly/chi"
	"golang.org/x/net/context"
)

// Protect is a middleware that cancels incoming requests and returns a 503
// HTTP code if system memory use exceeds the threshold set.
//
// This middleware should be inserted fairly early in the middleware stack to
// ensure that request is cancelled early
//
// Protect accepts a single parameter - a float64 defining a fraction of the
// system memory process can use before it starts cancelling requests
//
// Right now it relies on memory usage data provided by /proc/meminfo so it is
// Linux specific. For convenience (dev envs) it will do nothing on other archs
func Protect(limit float64) func(chi.Handler) chi.Handler {
	return func(next chi.Handler) chi.Handler {
		fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			if MemoryUsage() > limit {
				http.Error(w, http.StatusText(503), 503)
				return
			}

			next.ServeHTTPC(ctx, w, r)
		}
		return chi.HandlerFunc(fn)
	}
}

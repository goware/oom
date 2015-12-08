oom
============
[![GoDoc](https://godoc.org/github.com/goware/oom?status.svg)](https://godoc.org/github.com/goware/oom) [![Go Report Card](http://goreportcard.com/badge/goware/oom)](http://goreportcard.com/report/goware/oom) [![License:MIT](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://github.com/goware/oom/blob/master/LICENSE.md)

`oom` is a small collection of middlewares for [chi](https://github.com/pressly/chi) that help you handle your system running out of memory

In theory memory use should be more or less stable or at least predictable and you should be scaling our by adding new servers.

In practice you have:
* leaky GCO libs
* spiky load or memory use
* thundering herd issues
* not enough time to track memory leak
* all of the above

### Stop processing new requests until GC catches up
Put `oom.Protect(t float64)` early in your middleware chain to terminate request processing when total system memory use exceeds `t` and return HTTP 503 response

Example
```go
func main() {
  r := chi.NewRouter()

  // Start returning 503 if memory use exceeds 90% of total system memory
  r.Use(oom.Protect(0.9))
  r.Get("/", GetResource)

  // This endpoint uses a lot of memory, use lower threshold
  r.Post("/", oom.Protect(0.7), CreateResource)

  http.ListenAndServe(":80", r)
}
```

### Restart process
Put `oom.Selfdestruct(t float64)` in your middleware chain to terminate process (not just goroutine!) when total system memory use exceeds `t`. Termination will be handled by sending SIGTERM to the process letting you gracefully stop it

Example
```go
func main() {
  r := chi.NewRouter()

  // This is one leaky http handler that will make process eventually consume
  // all the memory available and crash.
  // Gracefully terminate it when system memory use exceeds 95% and have docker
  // --restart=always start it again
  r.Use(oom.Selfdestruct(0.95))
  r.Get("/", GetResource)
  r.Post("/", CreateResource)

  graceful.AddSignal(syscall.SIGTERM)
  graceful.Timeout(10 * time.Second) // Wait timeout for handlers to finish.
  graceful.ListenAndServe(":80", r)
  graceful.Wait()
}
```

### Call a custom function to handle restart/cleanup for you
Put `oom.SelfdestructFn(t float64, fn func(context.Context) bool)` in your middleware chain to terminate process (not just goroutine!) when total system memory use exceeds `t` by calling your own function that will handle the termination, or do something completely different.

Example
```go
func main() {
  r := chi.NewRouter()

  // This is one leaky http handler that will make process eventually consume
  // all the memory available and crash.
  // Handle that with a custom function
  r.Use(oom.SelfdestructFn(0.95, customKillFunction))
  r.Get("/", GetResource)
  r.Post("/", CreateResource)

  http.ListenAndServe(":80", r)
}
```

### Change memory use sampling frequency
Use `oom.SetUpdateInteval(i time.Duration)` to only update current memory use at most every `i`. By default it's updated on every request.

Example
```go
func main() {
  r := chi.NewRouter()

  // This app receives a very high volume of requests that have a tiny impact on
  // memory use. Sampling system memory use once a second is enough
  oom.SetUpdateInterval(time.Second)

  // Start returning 503 if memory use exceeds 90% of total system memory
  r.Use(oom.Protect(0.9))
  r.Get("/", GetResource)

  // This endpoint uses a lot of memory, use lower threshold
  r.Post("/", oom.Protect(0.7), CreateResource)

  http.ListenAndServe(":80", r)
}
```

### OS compatibility
oom middlewares are only able to get memory usage data on Linux. Kernels 3.14 and newer offer better accuracy.
Pull requests adding other OS support are very welcome!

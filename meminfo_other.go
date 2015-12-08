// +build !linux

package oom

// MemoryUsage does nothing if you're not on Linux
// Implementations for other OSes/archs are always welcome
func MemoryUsage() float64 {
	return 0.0
}

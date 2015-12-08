// +build linux

package selfdestruct

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var meminfo = &MemInfo{}

func MemoryUsage() float64 {
	meminfo.Update()

	total := meminfo.Total()
	if total == 0 { // don't divide by 0
		return 0.0
	}
	return float64(meminfo.Used()) / float64(total)
}

func SetUpdateInteval(i time.Duration) {
	if i.Nanoseconds() == 0 {
		return
	}
	meminfo.mutex.Lock()
	meminfo.updateInterval = &i
	meminfo.mutex.Unlock()
}

// Following is copied/adapted github.com/guillermo/go.procmeminfo

type MemInfo struct {
	mutex          sync.RWMutex
	lastUpdate     time.Time // in milliseconds
	updateInterval *time.Duration

	Values map[string]uint64
}

func (m *MemInfo) Update() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.updateInterval != nil {
		now := time.Now()
		if now.Before(m.lastUpdate.Add(*m.updateInterval)) {
			return nil // no need to update yet
		}
		m.lastUpdate = now
	}

	if m.Values == nil {
		m.Values = make(map[string]uint64)
	}

	var err error

	path := filepath.Join("/proc/meminfo")
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		n := strings.Index(text, ":")
		if n == -1 {
			continue
		}

		key := text[:n] // metric
		data := strings.Split(strings.Trim(text[(n+1):], " "), " ")
		if len(data) == 1 {
			value, err := strconv.ParseUint(data[0], 10, 64)
			if err != nil {
				continue
			}
			m.Values[key] = value
		} else if len(data) == 2 {
			if data[1] == "kB" {
				value, err := strconv.ParseUint(data[0], 10, 64)
				if err != nil {
					continue
				}
				m.Values[key] = value * 1024
			}
		}

	}
	return nil
}

// Total() returns total RAM of system
func (m *MemInfo) Total() uint64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	total, _ := m.Values["MemTotal"]
	return total
}

// Available() returns the amount of still RAM available
// It uses new process mem available estimation - factors in reclaiming file
// buffers and cache.
// On Linux kernels 3.14+ it tens to err on the side of caution
// On Linux kernels older than 3.14 it's a tiny bit too optimistic, unless you
// are using next to no file buffers
func (m *MemInfo) Available() uint64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if av, ok := m.Values["MemAvailable"]; ok {
		return av
	}
	// This will happen only on kernels < 3.14, will overestimate a tiny bit
	fr, _ := m.Values["MemFree"]
	buf, _ := m.Values["Buffers"]
	ca, _ := m.Values["Cached"]
	return fr + buf + ca
}

// Used() returns the amount of non-reclaimable memory used
func (m *MemInfo) Used() uint64 {
	return m.Total() - m.Available()
}

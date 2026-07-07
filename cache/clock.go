// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package cache provides caching functionality
// Author: Admilson B. F. Cossa

package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// CachedClock provides a fast, low-cost timestamp read for the hot path.
// It is updated by a background goroutine at a configurable interval (default 10ms for High-performance performance).
// Hot path reads are atomic loads of an int64 unix-nanoseconds value.
type CachedClock struct {
	_       [64]byte // Padding before
	nsec    atomic.Int64
	_       [64 - 8]byte // Padding after (64 - sizeof(int64))
	stop    chan struct{}
	stopped atomic.Bool // Guard against double-close
}

// NewCachedClock creates a CachedClock whose cached timestamp is refreshed by a
// background goroutine every updateInterval. A non-positive interval defaults to 10ms.
func NewCachedClock(updateInterval time.Duration) *CachedClock {
	cc := &CachedClock{stop: make(chan struct{})}
	if updateInterval <= 0 {
		updateInterval = 10 * time.Millisecond // High-performance: 10ms cache for production
	}
	cc.nsec.Store(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cc.nsec.Store(time.Now().UnixNano())
			case <-cc.stop:
				return
			}
		}
	}()
	return cc
}

// GetNsec returns a pointer to the atomic cached nanoseconds value.
func (cc *CachedClock) GetNsec() *atomic.Int64 {
	return &cc.nsec
}

// GetNsecValue returns the cached nanoseconds value directly (for hot path optimization)
func (cc *CachedClock) GetNsecValue() int64 {
	return cc.nsec.Load()
}

// Now returns the cached time as a time.Time.
func (cc *CachedClock) Now() time.Time {
	n := cc.nsec.Load()
	return time.Unix(0, n)
}

// Close stops the background refresh goroutine. It is safe to call multiple times.
func (cc *CachedClock) Close() {
	if cc.stopped.CompareAndSwap(false, true) {
		close(cc.stop)
	}
}

// ----------------------------- Cached Clock --------------------------------

// Global cached clock singleton to eliminate allocations
var (
	globalCachedClock     *CachedClock
	globalCachedClockOnce sync.Once
)

// GetGlobalCachedClock returns the singleton cached clock instance.
func GetGlobalCachedClock() *CachedClock {
	globalCachedClockOnce.Do(func() {
		globalCachedClock = NewCachedClock(10 * time.Millisecond)
	})
	return globalCachedClock
}

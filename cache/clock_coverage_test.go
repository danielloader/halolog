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
// Package cache tests exercise the CachedClock hot-path timestamp cache.
// @author Admilson B. F. Cossa

package cache

import (
	"sync"
	"testing"
	"time"
)

// TestNewCachedClock_DefaultInterval verifies that a non-positive interval falls
// back to the documented 10ms default without panicking, and that the clock is
// immediately usable (seeded on construction).
func TestNewCachedClock_DefaultInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
	}{
		{name: "zero_interval_defaults", interval: 0},
		{name: "negative_interval_defaults", interval: -5 * time.Millisecond},
		{name: "explicit_small_interval", interval: 1 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cc := NewCachedClock(tc.interval)
			defer cc.Close()

			// The clock must be seeded on construction (not zero).
			if got := cc.GetNsecValue(); got == 0 {
				t.Fatalf("expected clock to be seeded on construction, got 0")
			}

			// Now() must return a plausible, recent time.
			now := cc.Now()
			delta := time.Since(now)
			if delta < -time.Second || delta > time.Minute {
				t.Fatalf("Now() returned implausible time %v (delta from wall clock: %v)", now, delta)
			}
		})
	}
}

// TestCachedClock_GetNsecPointer verifies GetNsec returns a live pointer to the
// same atomic that GetNsecValue reads, so a background tick is observable through it.
func TestCachedClock_GetNsecPointer(t *testing.T) {
	cc := NewCachedClock(0)
	defer cc.Close()

	p := cc.GetNsec()
	if p == nil {
		t.Fatal("GetNsec returned nil pointer")
	}

	// The pointer must reflect the same value as GetNsecValue.
	if p.Load() != cc.GetNsecValue() {
		t.Fatalf("GetNsec pointer (%d) disagrees with GetNsecValue (%d)", p.Load(), cc.GetNsecValue())
	}
}

// TestCachedClock_BackgroundRefresh verifies the background goroutine actually
// advances the cached timestamp over time.
func TestCachedClock_BackgroundRefresh(t *testing.T) {
	// Use a fast 1ms tick so the test is quick but deterministic.
	cc := NewCachedClock(1 * time.Millisecond)
	defer cc.Close()

	first := cc.GetNsecValue()

	// Wait long enough for several ticks to fire.
	deadline := time.Now().Add(500 * time.Millisecond)
	var second int64
	for time.Now().Before(deadline) {
		second = cc.GetNsecValue()
		if second > first {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	if second <= first {
		t.Fatalf("background refresh did not advance cached time: first=%d second=%d", first, second)
	}
}

// TestCachedClock_Now verifies Now() converts the cached nanoseconds back into a
// consistent time.Time whose UnixNano matches the cached value.
func TestCachedClock_Now(t *testing.T) {
	cc := NewCachedClock(10 * time.Millisecond)
	defer cc.Close()

	nsec := cc.GetNsecValue()
	now := cc.Now()

	if now.UnixNano() != nsec {
		t.Fatalf("Now().UnixNano()=%d does not match cached nsec=%d", now.UnixNano(), nsec)
	}
}

// TestCachedClock_CloseIdempotent verifies Close can be called multiple times
// safely (the stopped guard must prevent a double-close panic).
func TestCachedClock_CloseIdempotent(t *testing.T) {
	cc := NewCachedClock(5 * time.Millisecond)

	// Multiple sequential closes must not panic.
	cc.Close()
	cc.Close()
	cc.Close()
}

// TestCachedClock_CloseConcurrent verifies concurrent Close calls are safe (the
// CompareAndSwap guard must serialize the channel close).
func TestCachedClock_CloseConcurrent(t *testing.T) {
	cc := NewCachedClock(5 * time.Millisecond)

	var wg sync.WaitGroup
	const closers = 16
	wg.Add(closers)
	for i := 0; i < closers; i++ {
		go func() {
			defer wg.Done()
			cc.Close()
		}()
	}
	wg.Wait()
}

// TestCachedClock_StopsRefreshAfterClose verifies the cached value no longer
// advances once the background goroutine has been stopped.
func TestCachedClock_StopsRefreshAfterClose(t *testing.T) {
	cc := NewCachedClock(1 * time.Millisecond)

	// Let it tick at least once.
	time.Sleep(20 * time.Millisecond)
	cc.Close()

	// Give the goroutine time to observe the stop signal and exit.
	time.Sleep(20 * time.Millisecond)
	valueAfterClose := cc.GetNsecValue()

	// Wait a while; the value must not change since the refresher is stopped.
	time.Sleep(50 * time.Millisecond)
	if cc.GetNsecValue() != valueAfterClose {
		t.Fatalf("cached value advanced after Close: before=%d after=%d",
			valueAfterClose, cc.GetNsecValue())
	}
}

// TestGetGlobalCachedClock_Singleton verifies the global cached clock is a stable
// singleton (sync.Once guarantees the same instance is returned).
func TestGetGlobalCachedClock_Singleton(t *testing.T) {
	first := GetGlobalCachedClock()
	second := GetGlobalCachedClock()

	if first == nil {
		t.Fatal("GetGlobalCachedClock returned nil")
	}
	if first != second {
		t.Fatal("GetGlobalCachedClock did not return the same singleton instance")
	}

	// The global clock must be seeded and usable.
	if first.GetNsecValue() == 0 {
		t.Fatal("global cached clock is not seeded")
	}
}

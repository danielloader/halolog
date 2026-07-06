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
// Package types provides core type definitions
// Author: Admilson B. F. Cossa

package types

import (
	"sync/atomic"
	"time"
)

// Global cached clock for entry functions
var globalEntryClock = &entryCachedClock{
	cachedTime: atomic.Value{},
	lastUpdate: atomic.Int64{},
}

func init() {
	now := time.Now()
	globalEntryClock.cachedTime.Store(now)
	globalEntryClock.lastUpdate.Store(now.UnixNano())
}

// entryCachedClock provides cached time for entry functions
type entryCachedClock struct {
	cachedTime atomic.Value // stores time.Time
	lastUpdate atomic.Int64 // stores UnixNano
}

// now returns cached time, updating if older than 1ms
func (cc *entryCachedClock) now() time.Time {
	lastNano := cc.lastUpdate.Load()
	now := time.Now()

	// Update if older than 1ms (1000x reduction in time.Now() calls)
	if now.Sub(time.Unix(0, lastNano)) > time.Millisecond {
		cc.cachedTime.Store(now)
		cc.lastUpdate.Store(now.UnixNano())
		return now
	}

	return cc.cachedTime.Load().(time.Time)
}

// UnixTimestampNow returns current Unix timestamp as int64
func UnixTimestampNow() int64 {
	return globalEntryClock.now().Unix()
}

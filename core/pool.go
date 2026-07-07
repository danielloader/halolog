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

// Package core provides the high-performance HaloLog logger implementation.
// Author: Admilson B. F. Cossa

package core

import (
	"sync"

	"github.com/go-gen-ecosystem/halolog/types"
)

// perPState holds per-processor state for lock-free logging.
// Pre-allocated LogEntry avoids heap allocation in the hot path.
type perPState struct {
	entry    types.LogEntry
	fieldBuf [64]types.TypedFieldData // Buffer for entry fields
	_        [64 - 8]byte             // Padding to cache line
}

// globalPerPPool is the per-P state pool
var globalPerPPool = &perPPool{
	pool: sync.Pool{
		New: func() interface{} {
			state := &perPState{}
			state.entry.StaticFields = state.fieldBuf[:]
			return state
		},
	},
}

type perPPool struct {
	pool sync.Pool
}

// get retrieves a per-P state from sync.Pool
//
//go:inline
func (p *perPPool) get() *perPState {
	s := p.pool.Get().(*perPState)
	s.entry.StaticFieldCount = 0
	// Ensure fields slice is reset to buffer capacity ???
	// s.entry.StaticFields is assumed to be backed by fieldBuf or re-sliced correctly.
	// Since we don't clobber the capacity in usage (we used [:n]),
	// we just need to ensuring len is correct?
	// The usage sets len/cap when writing.
	return s
}

// put returns the state to the pool
//
//go:inline
func (p *perPPool) put(state *perPState) {
	state.entry.Reset() // Safe reset
	p.pool.Put(state)
}

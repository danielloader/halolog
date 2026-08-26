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

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/types"
)

// perPState holds per-processor state for lock-free logging.
// Pre-allocated LogEntry avoids heap allocation in the hot path.
type perPState struct {
	entry    types.LogEntry
	fieldBuf [64]types.TypedFieldData // Buffer for entry fields
	_        [64 - 8]byte             // Padding to cache line

	// Direct-append fast path (see fluent_typed.go dispatch). directJSON is
	// selected per line at acquisition: the adapter's current direct encoder is
	// asserted to the CONCRETE JSON formatter so every per-field call below is
	// statically dispatched (and inlinable) — the same specialization precedent
	// as Logger.discardAdapter. A non-JSON DirectFieldEncoder simply keeps the
	// capture path (still correct). Both slices keep their heap backing across
	// reuses so the path stays zero-allocation after warmup.
	directJSON   *jsonfmt.Formatter
	directFields []byte // encoded `,"k":v` members for the current line
	lineBuf      []byte // scratch for assembling the full line

	// Level-first Line API state (see line.go): the owning logger and the
	// line's level, fixed when the line is opened.
	owner     *Logger
	lineLevel types.LogLevel

	// epoch is a use-generation counter: bumped every time the state returns
	// to the pool. Builders capture it at acquisition and terminals compare it
	// before dispatching, so a builder reused after its terminal (a documented
	// misuse) becomes a harmless no-op instead of double-releasing the state —
	// which would hand one pooled state to two live builders and silently
	// attribute one call's fields to another's line.
	epoch uint32

	directArr [1024]byte
	lineArr   [1280]byte
}

// globalPerPPool is the per-P state pool
var globalPerPPool = &perPPool{
	pool: sync.Pool{
		New: func() interface{} {
			state := &perPState{}
			state.entry.StaticFields = state.fieldBuf[:]
			state.directFields = state.directArr[:0]
			state.lineBuf = state.lineArr[:0]
			return state
		},
	},
}

type perPPool struct {
	pool sync.Pool
}

// get retrieves a per-P state from sync.Pool
func (p *perPPool) get() *perPState {
	s := p.pool.Get().(*perPState)
	s.entry.StaticFieldCount = 0
	s.directJSON = nil
	s.directFields = s.directFields[:0]
	// Restore the static-fields slice to its full backing buffer. A prior
	// dispatch may have resliced it to [:count]; without this, the next borrower
	// sees a shortened slice and silently drops fields once the write index
	// reaches that stale length. Reslicing the fixed fieldBuf array allocates
	// nothing, so the zero-allocation hot path is preserved.
	s.entry.StaticFields = s.fieldBuf[:]
	return s
}

// maxRetainedLineBytes caps how much direct-path buffer growth a pooled state
// may keep between uses. One pathological line (e.g. a multi-megabyte field)
// would otherwise pin its grown buffer on the pooled state indefinitely — an
// unbounded memory-retention vector. Oversized buffers are reset to their
// fixed backing arrays instead.
const maxRetainedLineBytes = 64 << 10

// put returns the state to the pool
func (p *perPPool) put(state *perPState) {
	state.entry.Reset() // Safe reset
	state.owner = nil   // do not pin a Logger via the pool
	state.epoch++       // invalidate any builder still holding this state
	if cap(state.lineBuf) > maxRetainedLineBytes {
		state.lineBuf = state.lineArr[:0]
	}
	if cap(state.directFields) > maxRetainedLineBytes {
		state.directFields = state.directArr[:0]
	}
	p.pool.Put(state)
}

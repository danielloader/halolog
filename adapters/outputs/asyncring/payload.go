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

// @author Admilson B. F. Cossa

package asyncring

import "github.com/go-gen-ecosystem/halolog/types"

// slotInlineFields is the number of fields a ring slot can hold without
// allocating. Records with more fields fall back to a heap slice on copy (rare).
// Chosen to cover the overwhelming majority of log lines while keeping each slot
// (and therefore the whole ring) small.
const slotInlineFields = 8

// entryPayload is a ring slot's owned copy of a log record. The producer copies
// a live (pooled) LogEntry into it so the source entry can be recycled
// immediately — this is what makes the async path safe, unlike enqueuing a
// pointer to a pooled entry.
//
// Field data is shallow-copied: string keys/values and the boxing-free typed
// values are immutable and safe to read later on the writer goroutine. An
// interface-wrapped MUTABLE pointer value is copied by reference and must not be
// logged through the async ring if the caller mutates it afterwards (documented
// on the adapter).
type entryPayload struct {
	level     types.LogLevel
	tsUnix    int64
	line      int
	message   string
	component string
	file      string

	fieldBuf [slotInlineFields]types.TypedFieldData
	// fields references fieldBuf on the common path, or a heap slice when a record
	// carries more than slotInlineFields fields.
	fields []types.TypedFieldData
}

// copyFrom fills p from a live log entry. It flattens the entry's static and
// dynamic fields into fields. It allocates only when the total field count
// exceeds slotInlineFields.
func (p *entryPayload) copyFrom(e *types.LogEntry) {
	p.level = e.Level
	p.tsUnix = e.TimestampUnix
	p.line = e.Line
	p.message = e.Message
	p.component = e.Component
	p.file = e.File

	n := e.StaticFieldCount
	if n > len(e.StaticFields) {
		n = len(e.StaticFields)
	}
	total := n + len(e.Fields)

	var dst []types.TypedFieldData
	if total <= len(p.fieldBuf) {
		dst = p.fieldBuf[:0]
	} else {
		dst = make([]types.TypedFieldData, 0, total)
	}
	dst = append(dst, e.StaticFields[:n]...)
	dst = append(dst, e.Fields...)
	p.fields = dst
}

// into projects the payload onto a scratch LogEntry the formatter can serialize.
// The scratch entry aliases p's field storage; it is only valid until the slot
// is freed, which the single writer goroutine guarantees by serializing before
// returning from dequeue.
func (p *entryPayload) into(e *types.LogEntry) {
	e.Level = p.level
	e.TimestampUnix = p.tsUnix // formatters read TimestampUnix; Timestamp is unused
	e.Line = p.line
	e.Message = p.message
	e.Component = p.component
	e.File = p.file
	e.StaticFields = p.fields
	e.StaticFieldCount = len(p.fields)
	e.Fields = nil
}

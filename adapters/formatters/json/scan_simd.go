//go:build goexperiment.simd

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

// Package json — PREP BRANCH ONLY: portable-SIMD escape scanning built on
// Go's experimental `simd` package (GOEXPERIMENT=simd, Go 1.27+). This file
// is compiled only under the experiment and is NOT wired into the shipped
// appendJSONString: it exists so the moment the simd package graduates,
// HaloLog flips a build tag with the scanner already written, oracle-tested,
// and benchmarked. The shipped default remains the SWAR scan.
//
// Mechanics per vector block (≥16 bytes, hardware width where available):
//   - needs-escape = (b ≤ 0x1F) OR (b == '"') OR (b == '\\'),
//     with unsigned ≤ built from Min+Equal since the portable layer has no
//     Less yet: min(b, 0x1F) == b  ⟺  b ≤ 0x1F.
//   - any-lane test: mask → lane bytes → reinterpret as uint64 words →
//     StorePart to a stack array → OR the ≤8 words. No reduction primitive
//     exists in the experiment yet; this is the cheapest portable substitute.
//   - a dirty block bails to the exact per-byte escape path from the block
//     start (identical semantics to the SWAR scanner); sub-vector tails use
//     the scalar table loop directly, because LoadUint8sPart ZERO-FILLS the
//     remainder and 0x00 is a control byte — a zero-filled tail would
//     always read as dirty.
//
// Author: Admilson B. F. Cossa

package json

import (
	"simd"
	"unsafe"
)

// escapeThreshold and the two escape bytes, broadcast once per scan.
const (
	simdCtrlMax   = 0x1F
	simdQuote     = '"'
	simdBackslash = '\\'
)

// maxVectorBytes bounds the stack scratch for the any-lane test: the simd
// package guarantees a fixed in-process vector length of at least 128 bits;
// current hardware tops out at 512 bits (64 bytes → 8 uint64 words).
const maxVectorWords = 8

// appendJSONStringSIMD is the portable-SIMD counterpart of appendJSONString:
// byte-identical output (see the oracle test), vector-width clean-scan.
func appendJSONStringSIMD(dst []byte, s string) []byte {
	// Read-only zero-copy view; the vector loads never write through it.
	b := unsafe.Slice(unsafe.StringData(s), len(s))

	ctrl := simd.BroadcastUint8s(simdCtrlMax)
	quote := simd.BroadcastUint8s(simdQuote)
	bslash := simd.BroadcastUint8s(simdBackslash)

	w := ctrl.Len() // fixed for the process: ≥16 bytes
	i, n := 0, len(b)
	var words [maxVectorWords]uint64

	for ; i+w <= n; i += w {
		v := simd.LoadUint8s(b[i : i+w])
		// b ≤ 0x1F  ⟺  min(b, 0x1F) == b   (portable unsigned ≤)
		dirtyMask := v.Min(ctrl).Equal(v).
			Or(v.Equal(quote)).
			Or(v.Equal(bslash))

		// Any-lane test: mask lanes → bytes (0x00/0xFF) → uint64 words.
		lanes := dirtyMask.ToInt8s().ToBits().ReshapeToUint64s()
		stored := lanes.StorePart(words[:])
		var any uint64
		for k := 0; k < stored; k++ {
			any |= words[k]
		}
		if any != 0 {
			// Exact escape path re-scans from the block start, emitting
			// clean bytes verbatim — same bail contract as the SWAR scan.
			return appendJSONEscaped(dst, s, i)
		}
	}
	for ; i < n; i++ {
		if !jsonNoEscape[s[i]] {
			return appendJSONEscaped(dst, s, i)
		}
	}
	return append(dst, s...)
}

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

// Package json — equivalence and performance tests for the SWAR
// word-at-a-time escape scan in appendJSONString.
// @author Admilson B. F. Cossa

package json

import (
	"bytes"
	"strings"
	"testing"
)

// referenceAppendJSONString is the pre-SWAR per-byte implementation, kept as
// the oracle: the SWAR scan must be byte-identical to it for every input.
func referenceAppendJSONString(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if !jsonNoEscape[s[i]] {
			return appendJSONEscaped(dst, s, i)
		}
	}
	return append(dst, s...)
}

// TestSWAR_EquivalenceExhaustiveBytes places every possible byte value at
// every position of an 8-byte window (word path) and a 3-byte window (tail
// path), so each SWAR lane and the sub-word loop see all 256 byte values.
func TestSWAR_EquivalenceExhaustiveBytes(t *testing.T) {
	for _, width := range []int{3, 8, 11} {
		for pos := 0; pos < width; pos++ {
			for b := 0; b < 256; b++ {
				buf := bytes.Repeat([]byte{'a'}, width)
				buf[pos] = byte(b)
				s := string(buf)
				got := appendJSONString(nil, s)
				want := referenceAppendJSONString(nil, s)
				if !bytes.Equal(got, want) {
					t.Fatalf("width=%d pos=%d byte=0x%02x: got %q want %q", width, pos, b, got, want)
				}
			}
		}
	}
}

// TestSWAR_EquivalenceStructured covers boundary lengths, escape bytes on
// word boundaries, UTF-8 (high-bit bytes must never trip the hasless term),
// and long mixed payloads.
func TestSWAR_EquivalenceStructured(t *testing.T) {
	cases := make([]string, 0, 18)
	cases = append(cases,
		"",
		"a", "ab", "abcdefg", "abcdefgh", "abcdefghi",
		strings.Repeat("x", 64),
		strings.Repeat("x", 65),
		"clean-prefix-clean-prefix\"then-quote",
		"seven..\\",  // escape byte exactly at index 7 (last lane of first word)
		"eight...\"", // escape byte at index 8 (first lane of second word)
		"\x00\x01\x02\x03\x04\x05\x06\x07",
		"tab\tnewline\ncr\rquote\"backslash\\",
		"héllo wörld 世界 🎉 — emoji and multibyte",
		"\x7f\x80\x81\xfe\xff high bytes pass through",
		strings.Repeat("the quick brown fox jumps over the lazy dog ", 20),
		strings.Repeat("q\"", 33),
	)
	// A pseudo-random soup over all byte values, deterministic seed.
	soup := make([]byte, 0, 4096)
	state := uint64(0x9E3779B97F4A7C15)
	for i := 0; i < 4096; i++ {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		soup = append(soup, byte(state))
	}
	cases = append(cases, string(soup))

	for i, s := range cases {
		got := appendJSONString(nil, s)
		want := referenceAppendJSONString(nil, s)
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d (%q…): SWAR output diverges from reference", i, truncate(s, 24))
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Benchmarks: the clean-scan throughput is where SWAR pays.

func BenchmarkAppendJSONString_Clean20(b *testing.B) {
	s := "user login processed"
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = appendJSONString(dst[:0], s)
	}
	_ = dst
}

func BenchmarkAppendJSONString_Clean80(b *testing.B) {
	s := strings.Repeat("request handled successfully for tenant ", 2)
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = appendJSONString(dst[:0], s)
	}
	_ = dst
}

func BenchmarkAppendJSONString_EscapeAt60(b *testing.B) {
	s := strings.Repeat("x", 60) + "\"tail"
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = appendJSONString(dst[:0], s)
	}
	_ = dst
}

func BenchmarkAppendJSONString_Reference_Clean80(b *testing.B) {
	s := strings.Repeat("request handled successfully for tenant ", 2)
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = referenceAppendJSONString(dst[:0], s)
	}
	_ = dst
}

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

// Package json — oracle and benchmarks for the experimental SIMD scanner
// (prep branch; compiled only under GOEXPERIMENT=simd).
// @author Admilson B. F. Cossa

package json

import (
	"bytes"
	"strings"
	"testing"
)

// TestSIMD_EquivalenceExhaustiveBytes mirrors the SWAR oracle: every byte
// value at every position across widths spanning sub-vector tails, exact
// vector multiples, and straddles.
func TestSIMD_EquivalenceExhaustiveBytes(t *testing.T) {
	for _, width := range []int{3, 15, 16, 17, 31, 32, 33, 64, 65} {
		for pos := 0; pos < width; pos++ {
			for b := 0; b < 256; b++ {
				buf := bytes.Repeat([]byte{'a'}, width)
				buf[pos] = byte(b)
				s := string(buf)
				got := appendJSONStringSIMD(nil, s)
				want := referenceAppendJSONString(nil, s)
				if !bytes.Equal(got, want) {
					t.Fatalf("width=%d pos=%d byte=0x%02x: got %q want %q", width, pos, b, got, want)
				}
			}
		}
	}
}

// TestSIMD_EquivalenceStructured reuses the hostile structured cases.
func TestSIMD_EquivalenceStructured(t *testing.T) {
	cases := []string{
		"",
		"a", "abcdefghijklmno", "abcdefghijklmnop", "abcdefghijklmnopq",
		strings.Repeat("x", 256),
		"clean-prefix-clean-prefix\"then-quote",
		"tab\tnewline\ncr\rquote\"backslash\\",
		"héllo wörld 世界 🎉 — multibyte stays clean",
		"\x7f\x80\x81\xfe\xff high bytes pass through",
		strings.Repeat("the quick brown fox jumps over the lazy dog ", 40),
		strings.Repeat("q\"", 65),
	}
	for i, s := range cases {
		got := appendJSONStringSIMD(nil, s)
		want := referenceAppendJSONString(nil, s)
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d: SIMD output diverges from reference", i)
		}
	}
}

func FuzzAppendJSONStringSIMD(f *testing.F) {
	f.Add("plain")
	f.Add("quote\" ctrl\x01 slash\\ high\xff")
	f.Fuzz(func(t *testing.T, s string) {
		if !bytes.Equal(appendJSONStringSIMD(nil, s), referenceAppendJSONString(nil, s)) {
			t.Fatalf("SIMD diverges for %q", s)
		}
	})
}

// Benchmarks: SIMD vs the shipped SWAR scan on clean payloads of increasing
// length — the graduation decision data.
func benchScanner(b *testing.B, fn func([]byte, string) []byte, s string) {
	dst := make([]byte, 0, len(s)+64)
	b.SetBytes(int64(len(s)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = fn(dst[:0], s)
	}
	_ = dst
}

func BenchmarkScanSIMD_Clean80(b *testing.B) {
	benchScanner(b, appendJSONStringSIMD, strings.Repeat("request handled for tenant acme okay ", 3)[:80])
}

func BenchmarkScanSWAR_Clean80(b *testing.B) {
	benchScanner(b, appendJSONString, strings.Repeat("request handled for tenant acme okay ", 3)[:80])
}

func BenchmarkScanSIMD_Clean1K(b *testing.B) {
	benchScanner(b, appendJSONStringSIMD, strings.Repeat("sixteen byte txt ", 64)[:1024])
}

func BenchmarkScanSWAR_Clean1K(b *testing.B) {
	benchScanner(b, appendJSONString, strings.Repeat("sixteen byte txt ", 64)[:1024])
}

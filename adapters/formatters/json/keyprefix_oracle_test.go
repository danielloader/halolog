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

// Package json — byte-identity oracle for the fused plain-key emit in
// appendKeyPrefix, mirroring the SWAR scanner's oracle discipline.
// @author Admilson B. F. Cossa

package json

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// referenceAppendKeyPrefix is the pre-fusion implementation (scan via
// appendJSONString, then bulk copy), kept as the oracle: the fused
// single-pass emit must be byte-identical for every key.
func referenceAppendKeyPrefix(dst []byte, kd *types.FieldKey, key string) []byte {
	if kd != nil && len(kd.JSONFragment) > 0 {
		return append(dst, kd.JSONFragment...)
	}
	dst = append(dst, ',', '"')
	dst = referenceAppendJSONString(dst, key)
	return append(dst, '"', ':')
}

// TestKeyPrefix_EquivalenceExhaustiveBytes places every byte value at every
// position of short, word-sized, and straddling keys, plus a non-empty dst
// prefix so the bail path's truncation-to-mark is exercised.
func TestKeyPrefix_EquivalenceExhaustiveBytes(t *testing.T) {
	prefix := []byte(`{"seed":1`)
	for _, width := range []int{1, 2, 3, 8, 11} {
		for pos := 0; pos < width; pos++ {
			for b := 0; b < 256; b++ {
				buf := bytes.Repeat([]byte{'k'}, width)
				buf[pos] = byte(b)
				key := string(buf)
				got := appendKeyPrefix(append([]byte(nil), prefix...), nil, key)
				want := referenceAppendKeyPrefix(append([]byte(nil), prefix...), nil, key)
				if !bytes.Equal(got, want) {
					t.Fatalf("width=%d pos=%d byte=0x%02x: got %q want %q", width, pos, b, got, want)
				}
			}
		}
	}
}

// TestKeyPrefix_EquivalenceStructured covers empty keys, long keys, escape
// bytes at both ends, keyed-fragment passthrough, and unicode keys.
func TestKeyPrefix_EquivalenceStructured(t *testing.T) {
	keys := []string{
		"", "k", "user_id", "exactly8b", strings.Repeat("long_key_", 8),
		"\"leading", "trailing\"", "mid\"dle", "\\", "\n", "\x00",
		"tab\tkey", "ünïcodé_кей_键",
		strings.Repeat("\"", 17),
	}
	for i, key := range keys {
		got := appendKeyPrefix(nil, nil, key)
		want := referenceAppendKeyPrefix(nil, nil, key)
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d key=%q: fused diverges from reference", i, key)
		}
	}
	// Pre-declared fragment path must be a pure passthrough on both.
	kd := &types.FieldKey{Name: "n", JSONFragment: KeyFragment("we\"ird")}
	got := appendKeyPrefix(nil, kd, "ignored")
	want := referenceAppendKeyPrefix(nil, kd, "ignored")
	if !bytes.Equal(got, want) {
		t.Fatalf("keyed fragment path diverges: %q vs %q", got, want)
	}
}

// FuzzAppendKeyPrefix cross-checks the fused emit against the reference on
// arbitrary key bytes and arbitrary existing buffer contents.
func FuzzAppendKeyPrefix(f *testing.F) {
	f.Add("", "user_id")
	f.Add(`{"time":"x"`, "k")
	f.Add("prefix", "we\"ird\\key\n")
	f.Add("", "\x00\x1f\x7f\x80\xff")
	f.Fuzz(func(t *testing.T, prior, key string) {
		got := appendKeyPrefix([]byte(prior), nil, key)
		want := referenceAppendKeyPrefix([]byte(prior), nil, key)
		if !bytes.Equal(got, want) {
			t.Fatalf("fused diverges for prior=%q key=%q", prior, key)
		}
	})
}

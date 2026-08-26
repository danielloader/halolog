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

// Package json — native fuzz targets. Run continuously with
//
//	go test ./adapters/formatters/json -fuzz FuzzAppendJSONString -fuzztime 30s
//
// On plain `go test` runs each target executes its seed corpus, so these are
// also always-on regression tests.
// @author Admilson B. F. Cossa

package json

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// FuzzAppendJSONString cross-checks the SWAR scanner against the retained
// per-byte reference implementation on arbitrary byte strings: the two must
// be byte-identical for every input.
func FuzzAppendJSONString(f *testing.F) {
	seeds := []string{
		"",
		"plain ascii message",
		"quote\" backslash\\ newline\n tab\t nul\x00",
		"héllo 世界 🎉",
		"\x7f\x80\xfe\xff",
		"seven..\\eight...\"nine",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := appendJSONString(nil, s)
		want := referenceAppendJSONString(nil, s)
		if !bytes.Equal(got, want) {
			t.Fatalf("SWAR diverges from reference for %q", s)
		}
	})
}

// FuzzFormatValidJSON asserts the formatter emits parseable JSON for
// arbitrary message/key/value bytes — hostile input may look ugly, but it
// must never corrupt the log stream.
func FuzzFormatValidJSON(f *testing.F) {
	f.Add("msg", "key", "value", int64(42), 3.14)
	f.Add("", "", "", int64(-1), 0.0)
	f.Add("q\"uote", "back\\slash", "ctrl\x01", int64(1<<62), -1e300)
	f.Fuzz(func(t *testing.T, msg, key, sval string, ival int64, fval float64) {
		fm := NewJsonFormatter()
		entry := &types.LogEntry{
			TimestampUnix: 1_756_200_000_000_000_000,
			Level:         types.InfoLevel,
			Message:       msg,
			StaticFields: []types.TypedFieldData{
				{Key: key, Val: types.StringValue(sval)},
				{Key: key, Val: types.Int64Value(ival)},
				{Key: key, Val: types.Float64Value(fval)},
			},
			StaticFieldCount: 3,
		}
		out := fm.Format(entry, nil)
		if !json.Valid(out) {
			t.Fatalf("formatter produced invalid JSON for msg=%q key=%q: %s", msg, key, out)
		}
	})
}

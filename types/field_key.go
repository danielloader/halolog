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

// Package types provides core type definitions.
// @author Admilson B. F. Cossa

package types

// FieldKey is a pre-declared, immutable field key. Declaring a key once (e.g. as
// a package-level var) lets the logging hot path skip per-call key escaping: a
// formatter can emit the key with a single copy of a pre-rendered fragment
// instead of scanning and escaping the key string on every log call.
//
// FieldKey is intentionally a plain data holder with no behaviour that would
// couple this leaf package to a specific formatter. Format-specific fragments
// are computed by the owning formatter package and stored here at construction
// time; the constructors live in higher-level packages (e.g. the root package's
// Key helper) that are allowed to import both types and the formatter.
//
// A FieldKey must be treated as immutable after construction — its fragments are
// read concurrently on the logging hot path without synchronization.
type FieldKey struct {
	// Name is the raw, unescaped key. Formatters that do not have a pre-rendered
	// fragment (e.g. plain-text/console) use this directly.
	Name string

	// JSONFragment holds the pre-escaped JSON object member prefix for this key,
	// i.e. the bytes `,"<escaped-name>":` including the leading comma, the quotes
	// around the key, and the trailing colon. It is nil when no JSON fragment was
	// precomputed, in which case a JSON formatter falls back to escaping Name.
	JSONFragment []byte
}

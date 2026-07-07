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

// AdapterManager manages output adapters.
//
//nolint:interfacebloat // Cohesive lifecycle+dispatch contract for the adapter set; members belong together.
type AdapterManager interface {
	Add(adapter Adapter)
	Remove(name string) bool
	Clear()
	WriteAll(entry *LogEntry)
	Get(name string) Adapter
	CloseAll() error
	Snapshot() []Adapter
	Clone() AdapterManager
}

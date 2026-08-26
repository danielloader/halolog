//go:build !unix && !windows

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
// Package adapters provides output adapters
// Author: Admilson B. F. Cossa

package file

// processExistsPlatform on platforms with no process-liveness probe
// (js/wasm, plan9, wasip1) reports every PID as alive. That is the safe
// direction: a lock is then never judged stale, so it is never stolen from a
// live process; the cost is that a crashed process's lock must be removed
// manually on these platforms.
func processExistsPlatform(_ int) bool {
	return true
}

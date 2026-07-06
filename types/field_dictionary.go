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

// FieldDictionary provides optimized field key management interface
// This interface allows types/ to depend only on the contract, not the implementation
type FieldDictionary interface {
	// Register registers a new field key and returns its ID
	Register(key string) int

	// Get retrieves field key by ID
	Get(id int) (string, bool)

	// GetID retrieves field ID by key
	GetID(key string) (int, bool)

	// GetBitmask retrieves the bitmask for a field key
	GetBitmask(key string) (uint64, bool)

	// Count returns the total number of registered fields
	Count() int

	// RegisterBulk registers multiple field keys atomically
	RegisterBulk(keys []string) []int

	// GetStats returns performance statistics
	GetStats() FieldDictionaryStats

	// GetOrRegisterFieldID gets existing field ID or registers new field
	GetOrRegisterFieldID(key string) int

	// GetFieldByID retrieves field key by ID (used in SetByID)
	GetFieldByID(id int) string
}

// FieldDictionaryStats contains performance statistics for field dictionary
type FieldDictionaryStats struct {
	TotalFields  int    // Total number of registered fields
	FastPathHits uint64 // Number of fast path hits (existing fields)
	SlowPathHits uint64 // Number of slow path hits (new fields)
}

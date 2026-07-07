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

import (
	"strconv"
)

// NewFieldBuffer creates a new field buffer
func NewFieldBuffer() *FieldBuffer {
	return &FieldBuffer{
		keys:   make([]string, 0, 8),
		values: make([]interface{}, 0, 8),
		keyMap: make(map[string]int, 8),
	}
}

// Add adds a key-value pair to the buffer
func (fb *FieldBuffer) Add(key string, value interface{}) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = formatValue(value)
		return
	}

	// Handle nil initialization
	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]interface{}, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	// Add new field (append handles growth)
	idx := len(fb.keys)
	fb.keys = append(fb.keys, key)
	fb.values = append(fb.values, formatValue(value))
	fb.keyMap[key] = idx
}

// Get retrieves a value by key
func (fb *FieldBuffer) Get(key string) (any, bool) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		return fb.values[idx], true
	}
	return "", false
}

// Keys returns all keys
func (fb *FieldBuffer) Keys() []string {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	keys := make([]string, len(fb.keys))
	copy(keys, fb.keys)
	return keys
}

// Values returns all values
func (fb *FieldBuffer) Values() []any {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	values := make([]any, len(fb.values))
	copy(values, fb.values)
	return values
}

// ToMap converts to a regular map
func (fb *FieldBuffer) ToMap() map[string]interface{} {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	m := make(map[string]interface{}, len(fb.keys))
	for i := range fb.keys {
		m[fb.keys[i]] = fb.values[i]
	}
	return m
}

// FromMap populates from a regular map
func (fb *FieldBuffer) FromMap(m map[string]interface{}) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	fb.keys = fb.keys[:0]
	fb.values = fb.values[:0]
	// Rebuild the key->index map so Get works after FromMap. It was previously
	// left stale, so every Get returned (nil, false) even though Size/ToMap
	// (which read keys/values directly) were correct.
	if fb.keyMap == nil {
		fb.keyMap = make(map[string]int, len(m))
	} else {
		for k := range fb.keyMap {
			delete(fb.keyMap, k)
		}
	}

	for k, v := range m {
		fb.keyMap[k] = len(fb.keys)
		fb.keys = append(fb.keys, k)
		fb.values = append(fb.values, formatValue(v))
	}
}

// Clear clears the buffer
func (fb *FieldBuffer) Clear() {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	fb.keys = fb.keys[:0]
	fb.values = fb.values[:0]
	// Clear the hash map
	for k := range fb.keyMap {
		delete(fb.keyMap, k)
	}
}

// Size returns the number of entries
func (fb *FieldBuffer) Size() int {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return len(fb.keys)
}

// Reset resets the buffer
func (fb *FieldBuffer) Reset() {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	fb.keys = fb.keys[:0]
	fb.values = fb.values[:0]
	// Clear the hash map
	for k := range fb.keyMap {
		delete(fb.keyMap, k)
	}
}

// Len returns the length
func (fb *FieldBuffer) Len() int {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return len(fb.keys)
}

// Key returns key at index
func (fb *FieldBuffer) Key(i int) string {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.keys[i]
}

// Value returns value at index
func (fb *FieldBuffer) Value(i int) any {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	return fb.values[i]
}

// Set adds or updates a field
func (fb *FieldBuffer) Set(key string, value any) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	// Initialize slices if needed
	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = value
		return
	}

	// Add new field if capacity allows
	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, value)
		fb.keyMap[key] = idx
	}
}

// SetInt adds an integer field
func (fb *FieldBuffer) SetInt(key string, value int64) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	// Initialize slices if needed
	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	// Use pre-allocated buffer in the struct
	buf := fb.intBuf[:]
	n := len(buf)

	// Handle special case of minimum int64
	if value == -9223372036854775808 {
		n = len(buf) - 20
		copy(buf[n:], "-9223372036854775808")
	} else {
		// Convert int64 to string backwards
		negative := value < 0
		if negative {
			value = -value
		}

		if value == 0 {
			n--
			buf[n] = '0'
		} else {
			for value > 0 {
				n--
				buf[n] = byte('0' + value%10)
				value /= 10
			}
		}

		if negative {
			n--
			buf[n] = '-'
		}
	}

	str := string(buf[n:])

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = str
		return
	}

	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, str)
		fb.keyMap[key] = idx
	}
}

// SetBool adds a boolean field
func (fb *FieldBuffer) SetBool(key string, value bool) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	str := "false"
	if value {
		str = "true"
	}

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = str
		return
	}

	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, str)
		fb.keyMap[key] = idx
	}
}

// SetFloat adds a float field
func (fb *FieldBuffer) SetFloat(key string, value float64) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	str := strconv.FormatFloat(value, 'f', -1, 64)

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = str
		return
	}

	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, str)
		fb.keyMap[key] = idx
	}
}

// SetString adds a string field
func (fb *FieldBuffer) SetString(key string, value string) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = value
		return
	}

	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, value)
		fb.keyMap[key] = idx
	}
}

// SetNil adds a nil field
func (fb *FieldBuffer) SetNil(key string) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if fb.keys == nil {
		fb.keys = make([]string, 0, 8)
		fb.values = make([]any, 0, 8)
		fb.keyMap = make(map[string]int, 8)
	}

	// O(1) lookup using hash map
	if idx, exists := fb.keyMap[key]; exists {
		fb.values[idx] = "null"
		return
	}

	if len(fb.keys) < cap(fb.keys) {
		idx := len(fb.keys)
		fb.keys = append(fb.keys, key)
		fb.values = append(fb.values, "null")
		fb.keyMap[key] = idx
	}
}

// SetBulk adds multiple fields
func (fb *FieldBuffer) SetBulk(fields []TypedField) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	fb.keys = fb.keys[:0]
	fb.values = fb.values[:0]

	needed := len(fields)
	if needed == 0 {
		return
	}

	if cap(fb.keys) >= needed {
		fb.keys = fb.keys[:needed]
		fb.values = fb.values[:needed]
		for i := 0; i < len(fields); i++ {
			fb.keys[i] = fields[i].GetKey()
			fb.values[i] = fields[i].GetValue()
		}
		return
	}

	newCap := needed * 2
	if newCap < 8 {
		newCap = 8
	}

	fb.keys = make([]string, needed, newCap)
	fb.values = make([]any, needed, newCap)

	for i := 0; i < len(fields); i++ {
		fb.keys[i] = fields[i].GetKey()
		fb.values[i] = fields[i].GetValue()
	}
}

// ForEach iterates over all fields with locking
func (fb *FieldBuffer) ForEach(fn func(key string, value any)) {
	if fb == nil || fn == nil {
		return
	}

	fb.mu.RLock()
	defer fb.mu.RUnlock()

	for i := 0; i < len(fb.keys); i++ {
		fn(fb.keys[i], fb.values[i])
	}
}

// ForEachUnsafe iterates without locking (use with caution)
func (fb *FieldBuffer) ForEachUnsafe(fn func(key string, value any)) {
	if fb == nil || fn == nil {
		return
	}
	for i := 0; i < len(fb.keys); i++ {
		fn(fb.keys[i], fb.values[i])
	}
}

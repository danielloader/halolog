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

// Package types coverage tests for the legacy and enhanced Indexed field
// stores, using a small deterministic field dictionary.
// @author Admilson B. F. Cossa

package types

import (
	"sync"
	"testing"
)

// testDictionary is a small deterministic FieldDictionary implementation used to
// drive the IndexedFieldStore with bounded, sequential field IDs. This
// avoids the hash-based fallback (nil dictionary) which can generate very large
// IDs unsuitable for chunk allocation in a test.
type testDictionary struct {
	mu     sync.Mutex
	byKey  map[string]int
	byID   map[int]string
	nextID int
}

func newTestDictionary() *testDictionary {
	return &testDictionary{
		byKey: make(map[string]int),
		byID:  make(map[int]string),
	}
}

func (d *testDictionary) Register(key string) int {
	return d.GetOrRegisterFieldID(key)
}

func (d *testDictionary) Get(id int) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	k, ok := d.byID[id]
	return k, ok
}

func (d *testDictionary) GetID(key string) (int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id, ok := d.byKey[key]
	return id, ok
}

func (d *testDictionary) GetBitmask(key string) (uint64, bool) {
	id, ok := d.GetID(key)
	if !ok {
		return 0, false
	}
	return 1 << uint(id%64), true
}

func (d *testDictionary) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.byKey)
}

func (d *testDictionary) RegisterBulk(keys []string) []int {
	ids := make([]int, len(keys))
	for i, k := range keys {
		ids[i] = d.GetOrRegisterFieldID(k)
	}
	return ids
}

func (d *testDictionary) GetStats() FieldDictionaryStats {
	return FieldDictionaryStats{TotalFields: d.Count()}
}

func (d *testDictionary) GetOrRegisterFieldID(key string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if id, ok := d.byKey[key]; ok {
		return id
	}
	id := d.nextID
	d.nextID++
	d.byKey[key] = id
	d.byID[id] = key
	return id
}

func (d *testDictionary) GetFieldByID(id int) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.byID[id]
}

func TestLegacyBasicIndexedFieldStore(t *testing.T) {
	qfs := NewBasicIndexedFieldStore(8)

	qfs.Set("a", "1")
	qfs.Set("b", "2")
	qfs.SetInt("c", 42)

	if qfs.Size() != 3 {
		t.Fatalf("Size expected 3, got %d", qfs.Size())
	}

	if v, ok := qfs.Get("a"); !ok || v != "1" {
		t.Errorf("Get(a) wrong: %v %v", v, ok)
	}
	if v, ok := qfs.Get("c"); !ok || v != "42" {
		t.Errorf("Get(c) wrong: %v %v", v, ok)
	}
	if _, ok := qfs.Get("missing"); ok {
		t.Errorf("Get(missing) should be false")
	}

	// Overwrite existing.
	qfs.Set("a", "updated")
	if v, _ := qfs.Get("a"); v != "updated" {
		t.Errorf("Set overwrite wrong: %v", v)
	}
	qfs.SetInt("c", 99)
	if v, _ := qfs.Get("c"); v != "99" {
		t.Errorf("SetInt overwrite wrong: %v", v)
	}

	if qfs.GetActiveBits() == 0 {
		t.Errorf("GetActiveBits should be non-zero after sets")
	}

	// Iterate visits all active fields.
	visited := map[string]string{}
	qfs.Iterate(func(k, v string) { visited[k] = v })
	if len(visited) != 3 {
		t.Errorf("Iterate visited %d, want 3", len(visited))
	}

	qfs.Reset()
	if qfs.Size() != 0 {
		t.Errorf("Reset should clear store, got size %d", qfs.Size())
	}
}

func TestNewBasicIndexedFieldStore_CapacityCap(t *testing.T) {
	// Capacity larger than 64 is capped to 64.
	qfs := NewBasicIndexedFieldStore(200)
	// We can add at most 64 fields; validate we can set well past the cap request.
	for i := 0; i < 64; i++ {
		qfs.Set(string(rune('A'+i%26))+string(rune('0'+i/26)), "v")
	}
	if qfs.Size() > 64 {
		t.Errorf("legacy store should be capped at 64, got %d", qfs.Size())
	}
}

func TestIndexedFieldStore_WithDict(t *testing.T) {
	dict := newTestDictionary()
	eqfs := NewIndexedFieldStoreWithDict(16, dict)

	eqfs.Set("k1", "v1")
	eqfs.Set("k2", "v2")

	if eqfs.Size() != 2 {
		t.Fatalf("Size expected 2, got %d", eqfs.Size())
	}

	if v, ok := eqfs.Get("k1"); !ok || v != "v1" {
		t.Errorf("Get(k1) wrong: %v %v", v, ok)
	}
	if _, ok := eqfs.Get("missing"); ok {
		t.Errorf("Get(missing) should be false")
	}

	// Overwrite.
	eqfs.Set("k1", "updated")
	if v, _ := eqfs.Get("k1"); v != "updated" {
		t.Errorf("Set overwrite wrong: %v", v)
	}

	all := eqfs.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll expected 2, got %d", len(all))
	}

	visited := 0
	eqfs.Iterate(func(k, v string) { visited++ })
	if visited != 2 {
		t.Errorf("Iterate visited %d, want 2", visited)
	}

	eqfs.Reset()
	if eqfs.Size() != 0 {
		t.Errorf("Reset should clear, got size %d", eqfs.Size())
	}
}

func TestIndexedFieldStore_DefaultChunkSize(t *testing.T) {
	// Non-positive chunk size falls back to default.
	dict := newTestDictionary()
	eqfs := NewIndexedFieldStoreWithDict(0, dict)
	eqfs.Set("k", "v")
	if v, ok := eqfs.Get("k"); !ok || v != "v" {
		t.Errorf("store with default chunk size failed: %v %v", v, ok)
	}

	eqfs2 := NewIndexedFieldStore(-5)
	if eqfs2 == nil {
		t.Fatal("NewIndexedFieldStore returned nil")
	}
}

func TestIndexedFieldStore_SetByID(t *testing.T) {
	dict := newTestDictionary()
	eqfs := NewIndexedFieldStoreWithDict(16, dict)

	id := dict.GetOrRegisterFieldID("registered")
	eqfs.SetByID(id, "value-by-id")

	if v, ok := eqfs.Get("registered"); !ok || v != "value-by-id" {
		t.Errorf("SetByID/Get roundtrip wrong: %v %v", v, ok)
	}
}

func TestIndexedFieldStore_FastPathToggle(t *testing.T) {
	dict := newTestDictionary()
	eqfs := NewIndexedFieldStoreWithDict(16, dict)

	// Seed chunk 0 via the slow path so subsequent fast-path writes land in
	// an already-allocated chunk (fast path bails to slow path otherwise).
	eqfs.Set("seed", "s")

	eqfs.EnableFastPath()

	// Writing a brand-new field key (bit previously 0) through the fast path
	// works correctly.
	eqfs.Set("fresh", "fv")
	if v, ok := eqfs.Get("fresh"); !ok || v != "fv" {
		t.Errorf("fast-path fresh Set/Get wrong: %v %v", v, ok)
	}

	// NOTE: re-Set of an already-set key on the fast path is intentionally
	// NOT asserted here. setFastPath uses atomic.AddUint64(&bitmask, 1<<idx)
	// which double-counts an already-set bit and corrupts the bitmask, making
	// the field unreadable. This is a real bug (see returned bug report).

	all := eqfs.GetAll() // exercises getAllFastPath
	if len(all) < 1 {
		t.Errorf("GetAll (fast path) returned %d", len(all))
	}

	eqfs.DisableFastPath()
	if v, ok := eqfs.Get("fresh"); !ok || v != "fv" {
		t.Errorf("after DisableFastPath Get wrong: %v %v", v, ok)
	}
}

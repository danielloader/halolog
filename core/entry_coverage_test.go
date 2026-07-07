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

// Package core coverage tests for the stack-allocated Entry type.
// @author Admilson B. F. Cossa

package core

import (
	"errors"
	"testing"

	"github.com/go-gen-ecosystem/halolog/types"
)

// TestEntry_TypedAddHelpers covers AddString/AddInt/AddInt64/AddFloat64/AddBool/AddError.
func TestEntry_TypedAddHelpers(t *testing.T) {
	var e Entry
	e.AddString("s", "hello")
	e.AddInt("i", 3)
	e.AddInt64("i64", 7)
	e.AddFloat64("f", 1.5)
	e.AddBool("b", false)
	e.AddError(errors.New("err msg"))

	if e.FieldCount() != 6 {
		t.Fatalf("FieldCount = %d, want 6", e.FieldCount())
	}

	fields := e.Fields()
	byKey := map[string]interface{}{}
	for _, f := range fields {
		byKey[f.Key] = f.Value
	}
	if byKey["s"] != "hello" {
		t.Errorf("s = %v", byKey["s"])
	}
	if byKey["i"] != 3 {
		t.Errorf("i = %v", byKey["i"])
	}
	if byKey["i64"] != int64(7) {
		t.Errorf("i64 = %v", byKey["i64"])
	}
	if byKey["f"] != 1.5 {
		t.Errorf("f = %v", byKey["f"])
	}
	if byKey["b"] != false {
		t.Errorf("b = %v", byKey["b"])
	}
	if byKey["error"] != "err msg" {
		t.Errorf("error = %v", byKey["error"])
	}
}

// TestEntry_AddErrorNil ensures AddError(nil) is a no-op.
func TestEntry_AddErrorNil(t *testing.T) {
	var e Entry
	e.AddError(nil)
	if e.FieldCount() != 0 {
		t.Errorf("FieldCount = %d, want 0 after AddError(nil)", e.FieldCount())
	}
}

// TestEntry_FieldsEmpty verifies Fields() returns nil when there are no fields.
func TestEntry_FieldsEmpty(t *testing.T) {
	var e Entry
	if got := e.Fields(); got != nil {
		t.Errorf("Fields() on empty entry = %v, want nil", got)
	}
}

// TestEntry_FieldsWithDynamic exercises the dynamic-overflow branch of Fields()
// (more than 16 fields) and confirms both static + dynamic fields are returned.
func TestEntry_FieldsWithDynamic(t *testing.T) {
	var e Entry
	for i := 0; i < 20; i++ {
		e.AddInt("k", i)
	}
	if e.fieldCount != 16 {
		t.Fatalf("static fieldCount = %d, want 16", e.fieldCount)
	}
	if len(e.dynamicFields) != 4 {
		t.Fatalf("dynamicFields = %d, want 4", len(e.dynamicFields))
	}

	all := e.Fields()
	if len(all) != 20 {
		t.Errorf("Fields() len = %d, want 20", len(all))
	}
	// Static values are 0..15, dynamic are 16..19.
	if all[0].Value != 0 || all[15].Value != 15 {
		t.Errorf("static boundary values wrong: [0]=%v [15]=%v", all[0].Value, all[15].Value)
	}
	if all[16].Value != 16 || all[19].Value != 19 {
		t.Errorf("dynamic boundary values wrong: [16]=%v [19]=%v", all[16].Value, all[19].Value)
	}
}

// TestEntry_ForEachFieldWithDynamic iterates over both static and dynamic
// fields and covers early termination inside the dynamic range.
func TestEntry_ForEachFieldWithDynamic(t *testing.T) {
	var e Entry
	for i := 0; i < 18; i++ { // 16 static + 2 dynamic
		e.AddInt("k", i)
	}

	seen := 0
	e.ForEachField(func(key string, value interface{}) bool {
		seen++
		return true
	})
	if seen != 18 {
		t.Errorf("full iteration saw %d, want 18", seen)
	}

	// Stop partway through the dynamic range (after 17 visits).
	seen = 0
	e.ForEachField(func(key string, value interface{}) bool {
		seen++
		return seen < 17
	})
	if seen != 17 {
		t.Errorf("early-stop iteration saw %d, want 17", seen)
	}
}

// TestEntry_ResetClearsDynamic ensures Reset zeroes counters and truncates the
// dynamic slice while preserving its capacity.
func TestEntry_ResetClearsDynamic(t *testing.T) {
	var e Entry
	for i := 0; i < 20; i++ {
		e.AddInt("k", i)
	}
	dynCap := cap(e.dynamicFields)

	e.Reset()

	if e.fieldCount != 0 {
		t.Errorf("fieldCount = %d after reset, want 0", e.fieldCount)
	}
	if len(e.dynamicFields) != 0 {
		t.Errorf("dynamicFields len = %d after reset, want 0", len(e.dynamicFields))
	}
	if cap(e.dynamicFields) != dynCap {
		t.Errorf("dynamicFields cap = %d after reset, want preserved %d", cap(e.dynamicFields), dynCap)
	}
	// Static field values are nilled out.
	if e.staticFields[0].Value != nil {
		t.Errorf("static field value not cleared: %v", e.staticFields[0].Value)
	}
}

// TestEntry_ToLogEntryDynamicFields covers ToLogEntry's dynamic-fields copy
// branch and the small-target-buffer allocation branch.
func TestEntry_ToLogEntryDynamicFields(t *testing.T) {
	var e Entry
	e.Timestamp = 42
	e.Level = types.ErrorLevel
	e.Component = "svc"
	e.Message = "boom"
	for i := 0; i < 18; i++ { // triggers dynamicFields (2 overflow)
		e.AddInt("k", i)
	}

	var target types.LogEntry // zero StaticFields cap -> forces allocation branch
	e.ToLogEntry(&target)

	if target.Level != types.ErrorLevel || target.Message != "boom" {
		t.Errorf("meta mismatch: level=%v msg=%q", target.Level, target.Message)
	}
	if target.TimestampUnix != 42 || target.Component != "svc" {
		t.Errorf("meta mismatch: ts=%d comp=%q", target.TimestampUnix, target.Component)
	}
	if target.StaticFieldCount != 16 {
		t.Errorf("StaticFieldCount = %d, want 16", target.StaticFieldCount)
	}
	if len(target.StaticFields) != 16 {
		t.Errorf("len(StaticFields) = %d, want 16", len(target.StaticFields))
	}
	if len(target.Fields) != 2 {
		t.Errorf("len(Fields) = %d, want 2 (dynamic copied)", len(target.Fields))
	}
}

// TestEntry_ToLogEntryReusesTargetBuffer verifies the fast branch where the
// target already has enough StaticFields capacity (no re-allocation).
func TestEntry_ToLogEntryReusesTargetBuffer(t *testing.T) {
	var e Entry
	e.AddString("a", "1")
	e.AddString("b", "2")

	target := types.LogEntry{
		StaticFields: make([]types.TypedFieldData, 0, 8),
	}
	e.ToLogEntry(&target)

	if target.StaticFieldCount != 2 {
		t.Errorf("StaticFieldCount = %d, want 2", target.StaticFieldCount)
	}
	if cap(target.StaticFields) < 2 {
		t.Errorf("cap shrank unexpectedly: %d", cap(target.StaticFields))
	}
	if target.StaticFields[0].Key != "a" || target.StaticFields[1].Key != "b" {
		t.Errorf("field keys wrong: %q %q", target.StaticFields[0].Key, target.StaticFields[1].Key)
	}
}

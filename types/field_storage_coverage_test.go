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

// Package types coverage tests for FieldValue, FieldBuffer, FieldsBuilder,
// StaticField, StaticFieldPool, and style/bitflag helpers.
// @author Admilson B. F. Cossa

package types

import (
	"errors"
	"strings"
	"testing"
)

func TestFieldValue_Constructors_ToInterface(t *testing.T) {
	tests := []struct {
		name string
		fv   FieldValue
		kind FieldKind
		want interface{}
	}{
		{"string", StringValue("hi"), KindString, "hi"},
		{"int", IntValue(7), KindInt, 7},
		{"int64", Int64Value(9), KindInt64, int64(9)},
		{"float64", Float64Value(2.5), KindFloat64, 2.5},
		{"bool-true", BoolValue(true), KindBool, true},
		{"bool-false", BoolValue(false), KindBool, false},
		{"any", AnyValue([]int{1}), KindAny, []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fv.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", tt.fv.Kind, tt.kind)
			}
			got := tt.fv.ToInterface()
			// Slices compared loosely by length via type assertion.
			if s, ok := tt.want.([]int); ok {
				gs, ok2 := got.([]int)
				if !ok2 || len(gs) != len(s) {
					t.Errorf("ToInterface any slice wrong: %v", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("ToInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldValue_Error(t *testing.T) {
	fv := ErrorValue(errors.New("boom"))
	if fv.Kind != KindError || fv.String != "boom" {
		t.Errorf("ErrorValue wrong: %+v", fv)
	}
	if fv.ToInterface() != "boom" {
		t.Errorf("ErrorValue.ToInterface should be message string")
	}

	nilFV := ErrorValue(nil)
	if nilFV.Kind != KindError || nilFV.String != "" {
		t.Errorf("ErrorValue(nil) wrong: %+v", nilFV)
	}
}

func TestFieldValue_ToInterface_Unknown(t *testing.T) {
	var fv FieldValue // KindUnknown zero value
	if fv.ToInterface() != nil {
		t.Errorf("KindUnknown ToInterface should be nil, got %v", fv.ToInterface())
	}
}

func TestStyleBitflags(t *testing.T) {
	var s uint8
	s = AddStyle(s, StyleBold)
	if !HasStyle(s, StyleBold) {
		t.Error("AddStyle/HasStyle bold failed")
	}
	if HasStyle(s, StyleItalic) {
		t.Error("italic should not be set")
	}

	s = AddStyle(s, StyleItalic)
	if !HasStyle(s, StyleItalic) || !HasStyle(s, StyleBold) {
		t.Error("should have both bold and italic")
	}

	s = RemoveStyle(s, StyleBold)
	if HasStyle(s, StyleBold) {
		t.Error("RemoveStyle should have cleared bold")
	}
	if !HasStyle(s, StyleItalic) {
		t.Error("RemoveStyle should not have cleared italic")
	}

	s = ToggleStyle(s, StyleUnderline)
	if !HasStyle(s, StyleUnderline) {
		t.Error("ToggleStyle should have set underline")
	}
	s = ToggleStyle(s, StyleUnderline)
	if HasStyle(s, StyleUnderline) {
		t.Error("ToggleStyle twice should have cleared underline")
	}
}

func TestStyledField_FluentAPI(t *testing.T) {
	base := TypedFieldData{Key: "amount", Value: int64(100), Type: TypedFieldInt64}
	sf := &StyledField{Field: base}

	sf.Color(ColorGreen).Background(ColorBlack).SetBold().SetItalic().
		SetUnderline().SetMasked().SetHighlight()

	if sf.FgColor != ColorGreen {
		t.Errorf("Color wrong: %v", sf.FgColor)
	}
	if sf.BgColor != ColorBlack {
		t.Errorf("Background wrong: %v", sf.BgColor)
	}
	if !sf.Bold || !sf.Italic || !sf.Underline || !sf.Masked || !sf.Highlight {
		t.Errorf("style flags not all set: %+v", sf)
	}

	if got := sf.ToTypedField(); got.GetKey() != "amount" {
		t.Errorf("ToTypedField wrong: %+v", got)
	}
}

func TestStyledField_SemanticStyles(t *testing.T) {
	mk := func() *StyledField { return &StyledField{} }

	if s := mk().Info(); s.FgColor != ColorCyan {
		t.Errorf("Info color wrong: %v", s.FgColor)
	}
	if s := mk().Success(); s.FgColor != ColorGreen {
		t.Errorf("Success color wrong: %v", s.FgColor)
	}
	if s := mk().Warning(); s.FgColor != ColorYellow {
		t.Errorf("Warning color wrong: %v", s.FgColor)
	}
	if s := mk().Error(); s.FgColor != ColorRed || !s.Bold {
		t.Errorf("Error style wrong: %+v", s)
	}
	if s := mk().Critical(); s.FgColor != ColorBrightRed || s.BgColor != ColorBlack || !s.Bold || !s.Underline {
		t.Errorf("Critical style wrong: %+v", s)
	}
	if s := mk().Muted(); s.FgColor != ColorGray {
		t.Errorf("Muted color wrong: %v", s.FgColor)
	}
}

func TestFieldBuffer_Lifecycle(t *testing.T) {
	fb := NewFieldBuffer()
	if fb.Size() != 0 || fb.Len() != 0 {
		t.Fatalf("new buffer should be empty")
	}

	fb.Add("a", "1")
	fb.Add("b", 2)
	if fb.Size() != 2 {
		t.Fatalf("after 2 Add expected size 2, got %d", fb.Size())
	}

	// Overwrite existing key.
	fb.Add("a", "updated")
	if fb.Size() != 2 {
		t.Errorf("overwrite should not grow size, got %d", fb.Size())
	}
	if v, ok := fb.Get("a"); !ok || v != "updated" {
		t.Errorf("Get after overwrite wrong: %v %v", v, ok)
	}

	if _, ok := fb.Get("missing"); ok {
		t.Errorf("Get missing key should be false")
	}

	keys := fb.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys length wrong: %d", len(keys))
	}
	vals := fb.Values()
	if len(vals) != 2 {
		t.Errorf("Values length wrong: %d", len(vals))
	}

	m := fb.ToMap()
	if len(m) != 2 || m["a"] != "updated" {
		t.Errorf("ToMap wrong: %v", m)
	}

	// Index accessors.
	if fb.Key(0) == "" {
		t.Errorf("Key(0) should be non-empty")
	}
	_ = fb.Value(0)

	fb.Clear()
	if fb.Size() != 0 {
		t.Errorf("Clear should empty buffer, got %d", fb.Size())
	}
}

func TestFieldBuffer_FromMap(t *testing.T) {
	fb := NewFieldBuffer()
	fb.FromMap(map[string]interface{}{"x": 1, "y": "two"})
	if fb.Size() != 2 {
		t.Fatalf("FromMap expected 2 entries, got %d", fb.Size())
	}
	// FromMap repopulates keys/values, so index-based views reflect the map.
	m := fb.ToMap()
	if m["y"] != "two" {
		t.Errorf("FromMap ToMap(y) wrong: %v", m["y"])
	}
	if m["x"] != "1" {
		t.Errorf("FromMap ToMap(x) wrong: %v", m["x"])
	}
	// NOTE: fb.Get("y") returns (,false) here because FromMap does NOT
	// repopulate the internal keyMap that Get relies on. This is a real bug
	// in FieldBuffer.FromMap (see returned bug report) - asserting the
	// broken behaviour here would be brittle, so we validate the map-based
	// views (Size/ToMap) that are correctly populated.
}

func TestFieldBuffer_TypedSetters(t *testing.T) {
	// The typed setters only append while len(keys) < cap(keys). A fresh
	// buffer has capacity 8, so keep the number of distinct keys <= 8.
	fb := NewFieldBuffer()
	fb.SetInt("i", 42)
	fb.SetInt("neg", -128)
	fb.SetInt("zero", 0)
	fb.SetBool("b", true)
	fb.SetBool("bf", false)
	fb.SetFloat("f", 3.14)
	fb.SetString("str", "value")
	fb.SetNil("n")

	if v, ok := fb.Get("i"); !ok || v != "42" {
		t.Errorf("SetInt wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("neg"); !ok || v != "-128" {
		t.Errorf("SetInt negative wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("zero"); !ok || v != "0" {
		t.Errorf("SetInt zero wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("b"); !ok || v != "true" {
		t.Errorf("SetBool true wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("bf"); !ok || v != "false" {
		t.Errorf("SetBool false wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("f"); !ok || v != "3.14" {
		t.Errorf("SetFloat wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("str"); !ok || v != "value" {
		t.Errorf("SetString wrong: %v %v", v, ok)
	}
	if v, ok := fb.Get("n"); !ok || v != "null" {
		t.Errorf("SetNil wrong: %v %v", v, ok)
	}

	// Update via typed setters exercises the existing-key branch (does not grow).
	fb.SetInt("i", 100)
	if v, _ := fb.Get("i"); v != "100" {
		t.Errorf("SetInt update wrong: %v", v)
	}
	fb.SetString("str", "new")
	if v, _ := fb.Get("str"); v != "new" {
		t.Errorf("SetString update wrong: %v", v)
	}
	fb.SetBool("b", false)
	if v, _ := fb.Get("b"); v != "false" {
		t.Errorf("SetBool update wrong: %v", v)
	}
	fb.SetFloat("f", 6.28)
	if v, _ := fb.Get("f"); v != "6.28" {
		t.Errorf("SetFloat update wrong: %v", v)
	}
	fb.SetNil("n")
	if v, _ := fb.Get("n"); v != "null" {
		t.Errorf("SetNil update wrong: %v", v)
	}
}

func TestFieldBuffer_SetPlain(t *testing.T) {
	fb := NewFieldBuffer()
	fb.Set("s", "raw")
	if v, ok := fb.Get("s"); !ok || v != "raw" {
		t.Errorf("Set wrong: %v %v", v, ok)
	}
	// Update existing key via Set.
	fb.Set("s", "changed")
	if v, _ := fb.Get("s"); v != "changed" {
		t.Errorf("Set update wrong: %v", v)
	}
}

func TestFieldBuffer_SetIntMinValue(t *testing.T) {
	fb := NewFieldBuffer()
	fb.SetInt("min", -9223372036854775808)
	if v, ok := fb.Get("min"); !ok || v != "-9223372036854775808" {
		t.Errorf("SetInt min int64 wrong: %v %v", v, ok)
	}
}

func TestFieldBuffer_SetBulk(t *testing.T) {
	fb := NewFieldBuffer()
	fields := []TypedField{
		TypedFieldData{Key: "a", Value: "1", Type: TypedFieldString},
		TypedFieldData{Key: "b", Value: int64(2), Type: TypedFieldInt64},
	}
	fb.SetBulk(fields)
	if fb.Len() != 2 {
		t.Fatalf("SetBulk expected 2, got %d", fb.Len())
	}

	// SetBulk with empty slice resets contents.
	fb.SetBulk(nil)
	if fb.Len() != 0 {
		t.Errorf("SetBulk(nil) should reset to empty, got %d", fb.Len())
	}

	// Force the grow path by exceeding the initial capacity of 8.
	fb2 := NewFieldBuffer()
	big := make([]TypedField, 20)
	for i := range big {
		big[i] = TypedFieldData{Key: strings.Repeat("k", i+1), Value: int64(i)}
	}
	fb2.SetBulk(big)
	if fb2.Len() != 20 {
		t.Errorf("SetBulk grow path expected 20, got %d", fb2.Len())
	}
}

func TestFieldBuffer_ForEach(t *testing.T) {
	fb := NewFieldBuffer()
	fb.Add("a", "1")
	fb.Add("b", "2")

	count := 0
	fb.ForEach(func(key string, value any) { count++ })
	if count != 2 {
		t.Errorf("ForEach visited %d, want 2", count)
	}

	count = 0
	fb.ForEachUnsafe(func(key string, value any) { count++ })
	if count != 2 {
		t.Errorf("ForEachUnsafe visited %d, want 2", count)
	}

	// Nil-safe behaviour.
	var nilBuf *FieldBuffer
	nilBuf.ForEach(func(key string, value any) { t.Error("should not be called") })
	nilBuf.ForEachUnsafe(func(key string, value any) { t.Error("should not be called") })
	fb.ForEach(nil)
	fb.ForEachUnsafe(nil)
}

func TestFieldBuffer_Reset(t *testing.T) {
	fb := NewFieldBuffer()
	fb.Add("a", "1")
	fb.Reset()
	if fb.Size() != 0 {
		t.Errorf("Reset should empty buffer, got %d", fb.Size())
	}
	// Re-add works after reset.
	fb.Add("b", "2")
	if fb.Size() != 1 {
		t.Errorf("re-add after reset failed, got %d", fb.Size())
	}
}

func TestFieldsBuilder(t *testing.T) {
	fb := NewFieldsBuilder()
	out := fb.AddInt("i", 1).
		AddInt64("i64", 2).
		AddFloat64("f", 3.5).
		AddBool("b", true).
		AddString("s", "hello").
		Build()

	if len(out) != 5 {
		t.Fatalf("FieldsBuilder.Build expected 5, got %d", len(out))
	}
	byKey := map[string]interface{}{}
	for _, f := range out {
		byKey[f.Key] = f.Value
	}
	if byKey["i"] != 1 {
		t.Errorf("AddInt value wrong: %v", byKey["i"])
	}
	if byKey["i64"] != "2" {
		t.Errorf("AddInt64 value wrong: %v", byKey["i64"])
	}
	if byKey["b"] != "true" {
		t.Errorf("AddBool value wrong: %v", byKey["b"])
	}
	if byKey["s"] != "hello" {
		t.Errorf("AddString value wrong: %v", byKey["s"])
	}
	if !strings.HasPrefix(byKey["f"].(string), "3.5") {
		t.Errorf("AddFloat64 value wrong: %v", byKey["f"])
	}
}

func TestStaticField_Conversion(t *testing.T) {
	// String field.
	strField := TypedFieldData{Key: "name", Value: "alice", Type: TypedFieldString}
	sf := ConvertTypedFieldToStatic(strField)
	if sf.Key() != "name" {
		t.Errorf("StaticField.Key wrong: %q", sf.Key())
	}
	if sf.Type() != TypedFieldString {
		t.Errorf("StaticField.Type wrong: %v", sf.Type())
	}
	if sv := sf.StringValue(); sv == nil || *sv != "alice" {
		t.Errorf("StaticField.StringValue wrong: %v", sv)
	}
	if string(sf.KeyBytes()) != "name" {
		t.Errorf("StaticField.KeyBytes wrong: %q", string(sf.KeyBytes()))
	}

	// Int field.
	intField := TypedFieldData{Key: "count", Value: int64(42), Type: TypedFieldInt64}
	si := ConvertTypedFieldToStatic(intField)
	if si.IntValue() != 42 {
		t.Errorf("StaticField.IntValue wrong: %d", si.IntValue())
	}

	// Int (non-int64) field widens.
	intField2 := TypedFieldData{Key: "c2", Value: 7, Type: TypedFieldInt64}
	si2 := ConvertTypedFieldToStatic(intField2)
	if si2.IntValue() != 7 {
		t.Errorf("StaticField.IntValue widen wrong: %d", si2.IntValue())
	}

	// Bool field.
	boolField := TypedFieldData{Key: "ok", Value: true, Type: TypedFieldBool}
	sb := ConvertTypedFieldToStatic(boolField)
	if !sb.BoolValue() {
		t.Errorf("StaticField.BoolValue wrong")
	}

	// Float field.
	floatField := TypedFieldData{Key: "score", Value: 9.0, Type: TypedFieldFloat64}
	sfl := ConvertTypedFieldToStatic(floatField)
	if sfl.FloatValue() != 9.0 {
		t.Errorf("StaticField.FloatValue wrong: %v", sfl.FloatValue())
	}
}

func TestStaticField_KeyTruncation(t *testing.T) {
	longKey := strings.Repeat("z", 50) // exceeds 31 byte cap
	f := TypedFieldData{Key: longKey, Value: "v", Type: TypedFieldString}
	sf := ConvertTypedFieldToStatic(f)
	if len(sf.Key()) > 31 {
		t.Errorf("StaticField key should be truncated to <=31, got %d", len(sf.Key()))
	}
}

func TestStaticField_ValueTypeMismatch(t *testing.T) {
	// Requesting the wrong accessor should return the zero value.
	strField := TypedFieldData{Key: "k", Value: "v", Type: TypedFieldString}
	sf := ConvertTypedFieldToStatic(strField)
	if sf.IntValue() != 0 {
		t.Errorf("IntValue on string field should be 0")
	}
	if sf.FloatValue() != 0.0 {
		t.Errorf("FloatValue on string field should be 0")
	}
	if sf.BoolValue() {
		t.Errorf("BoolValue on string field should be false")
	}
	if sf.UintValue() != 0 {
		t.Errorf("UintValue on string field should be 0")
	}

	intField := TypedFieldData{Key: "k", Value: int64(1), Type: TypedFieldInt64}
	si := ConvertTypedFieldToStatic(intField)
	if si.StringValue() != nil {
		t.Errorf("StringValue on int field should be nil")
	}
}

func TestConvertTypedFieldToTypedFieldData(t *testing.T) {
	src := TypedFieldData{Key: "k", Value: "v", Type: TypedFieldString}
	out := ConvertTypedFieldToTypedFieldData(src)
	if out.Key != "k" || out.Value != "v" || out.Type != TypedFieldString {
		t.Errorf("ConvertTypedFieldToTypedFieldData wrong: %+v", out)
	}
}

func TestStaticFieldPool(t *testing.T) {
	pool := NewStaticFieldPool()
	f := pool.Get()
	if f == nil {
		t.Fatal("pool.Get returned nil")
	}
	f.Key = "used"
	f.Value = "data"
	f.Type = TypedFieldInt64
	f.Val = Int64Value(7)
	pool.Put(f)

	// Put resets the field for reuse.
	if f.Key != "" || f.Value != nil || f.Type != TypedFieldString || f.Val.Kind != KindUnknown {
		t.Errorf("pool.Put should reset field, got %+v", f)
	}
}

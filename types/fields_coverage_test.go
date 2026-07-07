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

// Package types coverage tests for field constructors, type inference,
// value formatting, and typed-field accessors.
// @author Admilson B. F. Cossa

package types

import (
	"errors"
	"testing"
	"time"
)

func TestFields_KeyValuePairs(t *testing.T) {
	fields := Fields(
		"name", "john",
		"age", 42,
		"active", true,
		"score", 98.5,
	)
	if len(fields) != 4 {
		t.Fatalf("Fields expected 4, got %d", len(fields))
	}
	if fields[0].GetKey() != "name" || fields[0].GetType() != TypedFieldString {
		t.Errorf("Fields[0] wrong: %+v", fields[0])
	}
	if fields[1].GetType() != TypedFieldInt64 {
		t.Errorf("Fields[1] should be int64, got %v", fields[1].GetType())
	}
	if fields[2].GetType() != TypedFieldBool {
		t.Errorf("Fields[2] should be bool, got %v", fields[2].GetType())
	}
	if fields[3].GetType() != TypedFieldFloat64 {
		t.Errorf("Fields[3] should be float64, got %v", fields[3].GetType())
	}
}

func TestFields_OddPairs(t *testing.T) {
	if got := Fields("only_key"); got != nil {
		t.Errorf("Fields with odd args should return nil, got %v", got)
	}
}

func TestFields_NonStringKeySkipped(t *testing.T) {
	fields := Fields(123, "value", "good", "ok")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field (non-string key skipped), got %d", len(fields))
	}
	if fields[0].GetKey() != "good" {
		t.Errorf("expected surviving field key 'good', got %q", fields[0].GetKey())
	}
}

func TestFieldMap(t *testing.T) {
	m := map[string]interface{}{"a": 1, "b": "two"}
	fields := FieldMap(m)
	if len(fields) != 2 {
		t.Fatalf("FieldMap expected 2, got %d", len(fields))
	}
	seen := map[string]bool{}
	for _, f := range fields {
		seen[f.GetKey()] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Errorf("FieldMap missing keys: %v", seen)
	}
}

func TestFieldMap_Empty(t *testing.T) {
	if got := FieldMap(map[string]interface{}{}); got != nil {
		t.Errorf("FieldMap of empty map should be nil, got %v", got)
	}
}

func TestExplicitFieldBuilders(t *testing.T) {
	if f := String("k", "v"); f.Type != TypedFieldString || f.Value != "v" {
		t.Errorf("String builder wrong: %+v", f)
	}
	if f := Int("k", 7); f.Type != TypedFieldInt64 || f.Value != int64(7) {
		t.Errorf("Int builder wrong: %+v", f)
	}
	if f := Uint("k", 8); f.Type != TypedFieldUint64 || f.Value != uint64(8) {
		t.Errorf("Uint builder wrong: %+v", f)
	}
	if f := Float("k", 1.25); f.Type != TypedFieldFloat64 || f.Value != 1.25 {
		t.Errorf("Float builder wrong: %+v", f)
	}
	if f := Bool("k", true); f.Type != TypedFieldBool || f.Value != true {
		t.Errorf("Bool builder wrong: %+v", f)
	}
	now := time.Now()
	if f := Time("k", now); f.Type != TypedFieldTime {
		t.Errorf("Time builder wrong: %+v", f)
	}
	if f := Duration("k", time.Second); f.Type != TypedFieldDuration {
		t.Errorf("Duration builder wrong: %+v", f)
	}
	if f := Bytes("k", []byte("abc")); f.Type != TypedFieldBytes {
		t.Errorf("Bytes builder wrong: %+v", f)
	}
	if f := Nil("k"); f.Type != TypedFieldNil || f.Value != nil {
		t.Errorf("Nil builder wrong: %+v", f)
	}
}

func TestErrorField(t *testing.T) {
	f := Error("err", errors.New("bad"))
	if f.Type != TypedFieldError || f.Value != "bad" {
		t.Errorf("Error builder wrong: %+v", f)
	}
	fNil := Error("err", nil)
	if fNil.Type != TypedFieldNil || fNil.Value != nil {
		t.Errorf("Error(nil) should be TypedFieldNil, got %+v", fNil)
	}
}

func TestFieldShortcuts(t *testing.T) {
	if f := UserID(12345); f.Key != "user_id" {
		t.Errorf("UserID key wrong: %q", f.Key)
	}
	if f := RequestID("req-1"); f.Key != "request_id" || f.Value != "req-1" {
		t.Errorf("RequestID wrong: %+v", f)
	}
	if f := TraceID("trace-1"); f.Key != "trace_id" {
		t.Errorf("TraceID key wrong: %q", f.Key)
	}
	if f := SpanID("span-1"); f.Key != "span_id" {
		t.Errorf("SpanID key wrong: %q", f.Key)
	}
	if f := Err(errors.New("e")); f.Key != "error" || f.Type != TypedFieldError {
		t.Errorf("Err wrong: %+v", f)
	}
	if f := Msg("hello"); f.Key != "msg" || f.Value != "hello" {
		t.Errorf("Msg wrong: %+v", f)
	}
	if f := Method("GET"); f.Key != "method" || f.Value != "GET" {
		t.Errorf("Method wrong: %+v", f)
	}
	if f := Path("/x"); f.Key != "path" || f.Value != "/x" {
		t.Errorf("Path wrong: %+v", f)
	}
	if f := Status(200); f.Key != "status" || f.Value != int64(200) {
		t.Errorf("Status wrong: %+v", f)
	}
	if f := DurationMs(150); f.Key != "duration_ms" || f.Value != int64(150) {
		t.Errorf("DurationMs wrong: %+v", f)
	}
}

func TestSF_StyledFieldFactory(t *testing.T) {
	sf := SF("status", "ok")
	if sf == nil {
		t.Fatal("SF returned nil")
	}
	if sf.Field.GetKey() != "status" {
		t.Errorf("SF field key wrong: %q", sf.Field.GetKey())
	}
	styled := sf.Success()
	if styled.FgColor != ColorGreen {
		t.Errorf("SF().Success() should set green, got %v", styled.FgColor)
	}
}

func TestAutoInferType_AllKinds(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  TypedFieldType
	}{
		{"string", "s", TypedFieldString},
		{"int", int(1), TypedFieldInt64},
		{"int8", int8(1), TypedFieldInt64},
		{"int16", int16(1), TypedFieldInt64},
		{"int32", int32(1), TypedFieldInt64},
		{"int64", int64(1), TypedFieldInt64},
		{"uint", uint(1), TypedFieldUint64},
		{"uint8", uint8(1), TypedFieldUint64},
		{"uint16", uint16(1), TypedFieldUint64},
		{"uint32", uint32(1), TypedFieldUint64},
		{"uint64", uint64(1), TypedFieldUint64},
		{"float32", float32(1), TypedFieldFloat64},
		{"float64", float64(1), TypedFieldFloat64},
		{"bool", true, TypedFieldBool},
		{"time", time.Now(), TypedFieldTime},
		{"duration", time.Second, TypedFieldDuration},
		{"bytes", []byte("x"), TypedFieldBytes},
		{"nil", nil, TypedFieldNil},
		{"struct-fallback", struct{ X int }{1}, TypedFieldString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := AutoInferType("k", tt.value)
			if f == nil {
				t.Fatalf("AutoInferType returned nil for %s", tt.name)
			}
			if f.Type != tt.want {
				t.Errorf("AutoInferType(%s) type = %v, want %v", tt.name, f.Type, tt.want)
			}
		})
	}
}

func TestAutoInferType_ErrorValue(t *testing.T) {
	f := AutoInferType("e", errors.New("bad"))
	if f.Type != TypedFieldError || f.Value != "bad" {
		t.Errorf("AutoInferType(error) wrong: %+v", f)
	}

	var nilErr error
	f2 := AutoInferType("e", nilErr)
	if f2.Type != TypedFieldNil {
		t.Errorf("AutoInferType(nil error) should be Nil, got %v", f2.Type)
	}
}

func TestAutoInferType_IntWidening(t *testing.T) {
	f := AutoInferType("k", int8(-5))
	if v, ok := f.Value.(int64); !ok || v != -5 {
		t.Errorf("AutoInferType(int8) should widen to int64(-5), got %v (%T)", f.Value, f.Value)
	}
}

func TestFormatAny(t *testing.T) {
	tests := []struct {
		value interface{}
		want  string
	}{
		{"str", "str"},
		{int(-3), "-3"},
		{int64(9), "9"},
		{int32(9), "9"},
		{int16(9), "9"},
		{int8(9), "9"},
		{uint(7), "7"},
		{uint64(7), "7"},
		{uint32(7), "7"},
		{uint16(7), "7"},
		{uint8(7), "7"},
		{float64(1.5), "1.5"},
		{float32(2.5), "2.5"},
		{true, "true"},
		{false, "false"},
		{nil, "null"},
		{errors.New("oops"), "oops"},
		{time.Duration(0), "0s"}, // fmt.Stringer path
	}
	for _, tt := range tests {
		if got := FormatAny(tt.value); got != tt.want {
			t.Errorf("FormatAny(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestFormatAny_Fallback(t *testing.T) {
	type opaque struct{ A, B int }
	got := FormatAny(opaque{1, 2})
	if got == "" {
		t.Error("FormatAny fallback should produce non-empty string")
	}
}

func TestTypedFieldData_Accessors(t *testing.T) {
	str := TypedFieldData{Key: "k", Value: "hello", Type: TypedFieldString}
	if str.GetKey() != "k" {
		t.Errorf("GetKey wrong")
	}
	if str.GetValue() != "hello" {
		t.Errorf("GetValue wrong")
	}
	if str.GetType() != TypedFieldString {
		t.Errorf("GetType wrong")
	}
	if str.IsOptimized() {
		t.Errorf("IsOptimized should default false")
	}
	if sp := str.StringValue(); sp == nil || *sp != "hello" {
		t.Errorf("StringValue wrong: %v", sp)
	}

	i := TypedFieldData{Value: int64(99)}
	if i.IntValue() != 99 {
		t.Errorf("IntValue(int64) wrong: %d", i.IntValue())
	}
	i2 := TypedFieldData{Value: int(7)}
	if i2.IntValue() != 7 {
		t.Errorf("IntValue(int) wrong: %d", i2.IntValue())
	}
	notInt := TypedFieldData{Value: "notint"}
	if notInt.IntValue() != 0 {
		t.Errorf("IntValue on non-int should be 0")
	}

	u := TypedFieldData{Value: uint64(5)}
	if u.UintValue() != 5 {
		t.Errorf("UintValue(uint64) wrong")
	}
	u2 := TypedFieldData{Value: uint(6)}
	if u2.UintValue() != 6 {
		t.Errorf("UintValue(uint) wrong")
	}
	notUint := TypedFieldData{Value: "x"}
	if notUint.UintValue() != 0 {
		t.Errorf("UintValue non-uint should be 0")
	}

	fl := TypedFieldData{Value: float64(3.5)}
	if fl.FloatValue() != 3.5 {
		t.Errorf("FloatValue(float64) wrong")
	}
	fl2 := TypedFieldData{Value: float32(2.5)}
	if fl2.FloatValue() != 2.5 {
		t.Errorf("FloatValue(float32) wrong")
	}
	notFloat := TypedFieldData{Value: "x"}
	if notFloat.FloatValue() != 0.0 {
		t.Errorf("FloatValue non-float should be 0")
	}

	b := TypedFieldData{Value: true}
	if !b.BoolValue() {
		t.Errorf("BoolValue(true) wrong")
	}
	notBool := TypedFieldData{Value: "x"}
	if notBool.BoolValue() {
		t.Errorf("BoolValue non-bool should be false")
	}
}

func TestTypedFieldData_StringValueVariants(t *testing.T) {
	fromErr := TypedFieldData{Value: errors.New("boom")}
	if sp := fromErr.StringValue(); sp == nil || *sp != "boom" {
		t.Errorf("StringValue(error) wrong: %v", sp)
	}
	fromBytes := TypedFieldData{Value: []byte("abc")}
	if sp := fromBytes.StringValue(); sp == nil || *sp != "abc" {
		t.Errorf("StringValue([]byte) wrong: %v", sp)
	}
	fromInt := TypedFieldData{Value: 5}
	if sp := fromInt.StringValue(); sp != nil {
		t.Errorf("StringValue(int) should be nil, got %v", *sp)
	}
}

func TestTypedFieldType_String(t *testing.T) {
	tests := []struct {
		typ  TypedFieldType
		want string
	}{
		{TypedFieldString, "string"},
		{TypedFieldInt64, "int64"},
		{TypedFieldUint64, "uint64"},
		{TypedFieldFloat64, "float64"},
		{TypedFieldBool, "bool"},
		{TypedFieldTime, "time"},
		{TypedFieldDuration, "duration"},
		{TypedFieldError, "error"},
		{TypedFieldBytes, "bytes"},
		{TypedFieldNil, "nil"},
		{TypedFieldAny, "any"},
		{TypedFieldType(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("TypedFieldType(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

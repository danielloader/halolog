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
//
// Package utils behavioural coverage tests.
//
// @author Admilson B. F. Cossa

package utils

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// String builder pool
// -----------------------------------------------------------------------------

func TestAcquireReleaseStringBuilder(t *testing.T) {
	sb := AcquireStringBuilder()
	if sb == nil {
		t.Fatal("AcquireStringBuilder returned nil")
	}
	sb.WriteString("hello")
	if got := sb.String(); got != "hello" {
		t.Fatalf("builder content = %q, want %q", got, "hello")
	}
	ReleaseStringBuilder(sb)

	// Reacquire: pooled builder must be reset to empty.
	sb2 := AcquireStringBuilder()
	if got := sb2.String(); got != "" {
		t.Fatalf("reacquired builder not reset: got %q", got)
	}
	ReleaseStringBuilder(sb2)

	// Releasing nil must be safe (no panic).
	ReleaseStringBuilder(nil)
}

// -----------------------------------------------------------------------------
// FormatString (minimal %s / %d formatter)
// -----------------------------------------------------------------------------

func TestFormatString(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []interface{}
		want   string
	}{
		{"no verbs", "plain text", nil, "plain text"},
		{"one string", "hello %s", []interface{}{"world"}, "hello world"},
		{"one int", "n=%d", []interface{}{42}, "n=42"},
		{"int64", "n=%d", []interface{}{int64(64)}, "n=64"},
		{"uint", "n=%d", []interface{}{uint(7)}, "n=7"},
		{"mixed", "%s=%d", []interface{}{"code", 200}, "code=200"},
		{"trailing percent", "100%", nil, "100%"},
		{"missing arg for %s", "x=%s", nil, "x="},
		{"missing arg for %d", "x=%d", nil, "x="},
		{"non-string arg for %s ignored", "v=%s", []interface{}{123}, "v="},
		{"default fallthrough int32", "n=%d", []interface{}{int32(9)}, "n=9"},
		{"unknown verb copied", "50%q done", []interface{}{"x"}, "50%q done"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatString(tc.format, tc.args...); got != tc.want {
				t.Fatalf("FormatString(%q, %v) = %q, want %q", tc.format, tc.args, got, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// FormatInt / AppendInt (cache boundaries)
// -----------------------------------------------------------------------------

func TestFormatInt(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{200, "200"},
		{404, "404"},
		{599, "599"},   // last cached
		{600, "600"},   // first uncached
		{1000, "1000"}, // uncached
		{-1, "-1"},     // negative -> uncached path
	}
	for _, tc := range tests {
		if got := FormatInt(tc.in); got != tc.want {
			t.Fatalf("FormatInt(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAppendInt(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{599, "599"},
		{600, "600"},
		{-5, "-5"},
	}
	for _, tc := range tests {
		buf := AppendInt([]byte("v="), tc.in)
		if got := string(buf); got != "v="+tc.want {
			t.Fatalf("AppendInt(v=, %d) = %q, want %q", tc.in, got, "v="+tc.want)
		}
	}
}

// -----------------------------------------------------------------------------
// Prefix / suffix helpers
// -----------------------------------------------------------------------------

func TestFormatIntWithPrefix(t *testing.T) {
	if got := FormatIntWithPrefix("msg", 42); got != "msg42" {
		t.Fatalf("cached: got %q", got)
	}
	if got := FormatIntWithPrefix("user:", 12345); got != "user:12345" {
		t.Fatalf("uncached: got %q", got)
	}
}

func TestFormatStringWithSuffix(t *testing.T) {
	if got := FormatStringWithSuffix("file", ".log"); got != "file.log" {
		t.Fatalf("got %q", got)
	}
	if got := FormatStringWithSuffix("", ""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestAppendStringWithSufix(t *testing.T) {
	buf := AppendStringWithSufix([]byte("["), "file", ".log")
	if got := string(buf); got != "[file.log" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendIntWithPrefix(t *testing.T) {
	buf := AppendIntWithPrefix(make([]byte, 0, 32), "goroutines=", 42)
	if got := string(buf); got != "goroutines=42" {
		t.Fatalf("got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Key/value helpers
// -----------------------------------------------------------------------------

func TestFormatKeyValueAny(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		want  string
	}{
		{"status", 200, "status=200"},
		{"user", "john", "user=john"},
		{"active", true, "active=true"},
		{"inactive", false, "inactive=false"},
	}
	for _, tc := range tests {
		if got := FormatKeyValueAny(tc.key, tc.value); got != tc.want {
			t.Fatalf("FormatKeyValueAny(%q,%v) = %q, want %q", tc.key, tc.value, got, tc.want)
		}
	}
}

func TestFormatKeyValueAnySep(t *testing.T) {
	tests := []struct {
		key, sep string
		value    interface{}
		want     string
	}{
		{"status", "=", 200, "status=200"},
		{"user", ": ", "john", "user: john"},
		{"key", " -> ", "value", "key -> value"},
		{"id", ":", 12345, "id:12345"},
	}
	for _, tc := range tests {
		if got := FormatKeyValueAnySep(tc.key, tc.sep, tc.value); got != tc.want {
			t.Fatalf("FormatKeyValueAnySep(%q,%q,%v) = %q, want %q", tc.key, tc.sep, tc.value, got, tc.want)
		}
	}
}

func TestAppendKeyValueAny(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		want  string
	}{
		{"method", "GET", "method=GET"},
		{"status", 200, "status=200"},
		{"cached64", int64(100), "cached64=100"}, // int64 < 600 cache path
		{"big64", int64(9000), "big64=9000"},     // int64 >= 600 path
		{"active", true, "active=true"},
		{"inactive", false, "inactive=false"},
		{"ratio", float64(1.5), "ratio=1.5"}, // float64 path
	}
	for _, tc := range tests {
		buf := AppendKeyValueAny(make([]byte, 0, 64), tc.key, tc.value)
		if got := string(buf); got != tc.want {
			t.Fatalf("AppendKeyValueAny(%q,%v) = %q, want %q", tc.key, tc.value, got, tc.want)
		}
	}
}

// -----------------------------------------------------------------------------
// Two-int helpers
// -----------------------------------------------------------------------------

func TestFormatStringWithTwoIntsAndLabels(t *testing.T) {
	// Signature: (label1 string, a int, separator string, label2 string, b int)
	got := FormatStringWithTwoIntsAndLabels("goroutine ", 42, ", message ", "", 123)
	if got != "goroutine 42, message 123" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatTwoInts(t *testing.T) {
	tests := []struct {
		l1   string
		a    int
		sep  string
		l2   string
		b    int
		want string
	}{
		{"goroutine ", 42, ", message ", "", 123, "goroutine 42, message 123"},
		{"request ", 100, " retry ", "", 3, "request 100 retry 3"},
		{"user:", 123, "", ":session:", 456, "user:123:session:456"},
	}
	for _, tc := range tests {
		if got := FormatTwoInts(tc.l1, tc.a, tc.sep, tc.l2, tc.b); got != tc.want {
			t.Fatalf("FormatTwoInts = %q, want %q", got, tc.want)
		}
	}
}

func TestAppendTwoInts(t *testing.T) {
	buf := AppendTwoInts(make([]byte, 0, 64), "goroutine ", 42, ", message ", "", 123)
	if got := string(buf); got != "goroutine 42, message 123" {
		t.Fatalf("with sep: got %q", got)
	}
	buf2 := AppendTwoInts(make([]byte, 0, 64), "a", 1, "", "b", 2)
	if got := string(buf2); got != "a1b2" {
		t.Fatalf("no sep: got %q", got)
	}
}

func TestFormatTwoIntsSimple(t *testing.T) {
	if got := FormatTwoIntsSimple("Point(", 10, ',', 20, ")"); got != "Point(10,20)" {
		t.Fatalf("point: got %q", got)
	}
	if got := FormatTwoIntsSimple("Range ", 1, '-', 100, ""); got != "Range 1-100" {
		t.Fatalf("range: got %q", got)
	}
	// separator == 0 -> no separator byte written
	if got := FormatTwoIntsSimple("x", 1, 0, 2, ""); got != "x12" {
		t.Fatalf("zero sep: got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Prefix + suffix composition
// -----------------------------------------------------------------------------

func TestFormatWithPrefixAndSuffix(t *testing.T) {
	if got := FormatWithPrefixAndSuffix("REQ-", "12345", "-END"); got != "REQ-12345-END" {
		t.Fatalf("got %q", got)
	}
	if got := FormatWithPrefixAndSuffix("<", "div", ">"); got != "<div>" {
		t.Fatalf("got %q", got)
	}
	// any-typed prefix/value
	if got := FormatWithPrefixAndSuffix(1, 2, "x"); got != "12x" {
		t.Fatalf("any: got %q", got)
	}
}

func TestAppendWithPrefixAndSuffix(t *testing.T) {
	buf := AppendWithPrefixAndSuffix(make([]byte, 0, 64), `"user_id":`, "12345", ",")
	if got := string(buf); got != `"user_id":12345,` {
		t.Fatalf("got %q", got)
	}
}

// -----------------------------------------------------------------------------
// FormatAny (every type branch)
// -----------------------------------------------------------------------------

func TestFormatAny(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{"int cached", 42, "42"},
		{"int64 cached", int64(100), "100"},
		{"int64 uncached", int64(1000), "1000"},
		{"int32", int32(7), "7"},
		{"int16", int16(8), "8"},
		{"int8", int8(9), "9"},
		{"uint cached", uint(5), "5"},
		{"uint uncached", uint(1000), "1000"},
		{"uint64 cached", uint64(10), "10"},
		{"uint64 uncached", uint64(1000), "1000"},
		{"uint32 cached", uint32(11), "11"},
		{"uint32 uncached", uint32(1000), "1000"},
		{"uint16", uint16(12), "12"},
		{"uint8", uint8(13), "13"},
		{"string", "hi", "hi"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"float64", float64(3.14), "3.14"},
		{"float32", float32(2.5), "2.5"},
		{"nil", nil, "<nil>"},
		{"complex", struct{ X int }{1}, "<complex>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatAny(tc.value); got != tc.want {
				t.Fatalf("FormatAny(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestFormatTwoAny(t *testing.T) {
	if got := FormatTwoAny("user ", 5, " score ", "pts ", 99); got != "user 5 score pts 99" {
		t.Fatalf("with sep: got %q", got)
	}
	if got := FormatTwoAny("a", 1, "", "b", 2); got != "a1b2" {
		t.Fatalf("no sep: got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Error message builders
// -----------------------------------------------------------------------------

func TestFormatError(t *testing.T) {
	if got := FormatError(500, "database connection failed"); got != "Error 500: database connection failed" {
		t.Fatalf("got %q", got)
	}
	if got := FormatError(404, "user not found"); got != "Error 404: user not found" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatErrorMsg(t *testing.T) {
	if got := FormatErrorMsg("open failed", errors.New("no such file")); got != "open failed: no such file" {
		t.Fatalf("with err: got %q", got)
	}
	if got := FormatErrorMsg("all good", nil); got != "all good" {
		t.Fatalf("nil err: got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Timestamp appenders
// -----------------------------------------------------------------------------

func TestAppendTimestamp(t *testing.T) {
	// 2025-11-17T10:30:45Z
	ts := time.Date(2025, 11, 17, 10, 30, 45, 0, time.UTC)
	buf := AppendTimestamp(make([]byte, 0, 32), ts.Unix())
	if got := string(buf); got != "2025-11-17T10:30:45Z" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendTimestampNano(t *testing.T) {
	ts := time.Date(2025, 11, 17, 10, 30, 45, 123456789, time.UTC)
	buf := AppendTimestampNano(make([]byte, 0, 64), ts.UnixNano())
	if got := string(buf); got != "2025-11-17T10:30:45.123456789Z" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendTimestampNanoZeroPad(t *testing.T) {
	// Nanosecond value that requires zero padding: 5 ns -> .000000005
	ts := time.Date(2025, 1, 2, 3, 4, 5, 5, time.UTC)
	buf := AppendTimestampNano(make([]byte, 0, 64), ts.UnixNano())
	if got := string(buf); got != "2025-01-02T03:04:05.000000005Z" {
		t.Fatalf("got %q", got)
	}
}

// pad2 exercised indirectly with month>=10 and single-digit values above.
// Add an explicit day>=10 / month>=10 case plus a 3-digit branch via year handling.
func TestAppendTimestampPad2Branches(t *testing.T) {
	// December (month 12, two digits), day 25 (two digits), 23:59:59
	ts := time.Date(2025, 12, 25, 23, 59, 59, 0, time.UTC)
	buf := AppendTimestamp(make([]byte, 0, 32), ts.Unix())
	if got := string(buf); got != "2025-12-25T23:59:59Z" {
		t.Fatalf("got %q", got)
	}
}

// -----------------------------------------------------------------------------
// HTTP helpers
// -----------------------------------------------------------------------------

func TestAppendHTTPStatus(t *testing.T) {
	buf := AppendHTTPStatus(make([]byte, 0, 32), 200, "OK")
	if got := string(buf); got != "HTTP/1.1 200 OK\r\n" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendHTTPHeader(t *testing.T) {
	buf := AppendHTTPHeader(make([]byte, 0, 64), "Content-Type", "application/json")
	if got := string(buf); got != "Content-Type: application/json\r\n" {
		t.Fatalf("got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Metrics (cache hit rate)
// -----------------------------------------------------------------------------

func TestGetMetrics(t *testing.T) {
	// Drive several cached and uncached FormatInt calls, then read metrics.
	for i := 0; i < 10; i++ {
		FormatInt(i)      // cached (hit + total)
		FormatInt(10_000) // uncached (total only)
	}
	m := GetMetrics()
	if m.CacheTotal == 0 {
		t.Fatal("CacheTotal should be > 0 after calls")
	}
	if m.CacheHits > m.CacheTotal {
		t.Fatalf("hits (%d) > total (%d)", m.CacheHits, m.CacheTotal)
	}
	if m.CacheHitRate < 0 || m.CacheHitRate > 1 {
		t.Fatalf("hit rate out of [0,1]: %f", m.CacheHitRate)
	}
	// Alias must return identical shape.
	if GetStringHelperMetrics().CacheTotal == 0 {
		t.Fatal("deprecated alias returned zero total")
	}
}

// -----------------------------------------------------------------------------
// FormatWithFmt escape hatch
// -----------------------------------------------------------------------------

func TestFormatWithFmt(t *testing.T) {
	if got := FormatWithFmt("User %s id=%d", "bob", 7); got != "User bob id=7" {
		t.Fatalf("got %q", got)
	}
}

// -----------------------------------------------------------------------------
// Deprecated aliases (still exported -> must be covered)
// -----------------------------------------------------------------------------

func TestDeprecatedAliases(t *testing.T) {
	if got := FormatIntString(200); got != "200" {
		t.Fatalf("FormatIntString: %q", got)
	}
	if got := string(AppendIntString([]byte("n="), 5)); got != "n=5" {
		t.Fatalf("AppendIntString: %q", got)
	}
	if got := FormatIntAndText("g", ':', 42, "msg"); got != "g:42:msg" {
		t.Fatalf("FormatIntAndText: %q", got)
	}
	if got := string(AppendIntAndText([]byte("["), "g", ':', 42, "msg")); got != "[g:42:msg" {
		t.Fatalf("AppendIntAndText: %q", got)
	}
	if got := FormatStringWithTwoInts("Concurrent ", 1, 3); got != "Concurrent 1-3" {
		t.Fatalf("FormatStringWithTwoInts: %q", got)
	}
	if got := FormatStringWithTwoIntsAndSeparator("R", '/', 1, 2); got != "R1/2" {
		t.Fatalf("FormatStringWithTwoIntsAndSeparator: %q", got)
	}
	if got := FormatStringWithTwoValuesAndSeparator("v", ':', "a", "b"); got != "va:b" {
		t.Fatalf("FormatStringWithTwoValuesAndSeparator: %q", got)
	}
	if got := FormatStringWithPrefixAndSuffix("<", "x", ">"); got != "<x>" {
		t.Fatalf("FormatStringWithPrefixAndSuffix: %q", got)
	}
	if got := string(AppendIntStringWithPrefix([]byte{}, "n=", 5)); got != "n=5" {
		t.Fatalf("AppendIntStringWithPrefix: %q", got)
	}
	if got := string(AppendStringWithPrefixAndSuffix([]byte{}, "<", "x", ">")); got != "<x>" {
		t.Fatalf("AppendStringWithPrefixAndSuffix: %q", got)
	}
	if got := FormatErrorMessage(500, "boom"); got != "Error 500: boom" {
		t.Fatalf("FormatErrorMessage: %q", got)
	}
	if got := FormatAnyToString(42); got != "42" {
		t.Fatalf("FormatAnyToString: %q", got)
	}

	// Deprecated timestamp aliases must equal their new counterparts.
	unixSec := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC).Unix()
	unixNano := time.Date(2025, 6, 1, 12, 0, 0, 500, time.UTC).UnixNano()
	if a, b := string(AppendTimestampRFC3339(nil, unixSec)), string(AppendTimestamp(nil, unixSec)); a != b {
		t.Fatalf("AppendTimestampRFC3339 mismatch: %q vs %q", a, b)
	}
	if a, b := string(AppendTimestampRFC3339Nano(nil, unixNano)), string(AppendTimestampNano(nil, unixNano)); a != b {
		t.Fatalf("AppendTimestampRFC3339Nano mismatch: %q vs %q", a, b)
	}
	if got := string(AppendHTTPStatusLine(nil, 404, "Not Found")); got != "HTTP/1.1 404 Not Found\r\n" {
		t.Fatalf("AppendHTTPStatusLine: %q", got)
	}
}

// -----------------------------------------------------------------------------
// Cross-check: uncached FormatInt equals strconv.Itoa for a spread of values.
// -----------------------------------------------------------------------------

func TestFormatIntMatchesStrconv(t *testing.T) {
	for _, v := range []int{-1000, -1, 0, 599, 600, 12345, 1 << 20} {
		if got, want := FormatInt(v), strconv.Itoa(v); got != want {
			t.Fatalf("FormatInt(%d)=%q, strconv=%q", v, got, want)
		}
	}
}

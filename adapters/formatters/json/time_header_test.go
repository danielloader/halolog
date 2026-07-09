// @author Admilson B. F. Cossa

package json

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// headerOf renders the header for one (precision, nanos, level) combination.
func headerOf(p TimePrecision, nanos int64, level types.LogLevel) string {
	return string(headerAppenders[p](nil, nanos, level))
}

// expectedTime formats the reference timestamp the way the engine must.
func expectedTime(nanos int64, layout string) string {
	return time.Unix(0, nanos).Format(layout)
}

// TestHeaderPrecision verifies each precision strategy against the stdlib
// formatter, including zero-padded fractions and cache-hit repeatability.
func TestHeaderPrecision(t *testing.T) {
	// 42ms + 7µs + 1ns of fraction exercises zero padding at every width.
	nanos := time.Date(2026, 7, 7, 10, 30, 15, 42_007_001, time.Local).UnixNano()
	cases := []struct {
		p      TimePrecision
		layout string
	}{
		{PrecisionSecond, "2006-01-02T15:04:05Z07:00"},
		{PrecisionMilli, "2006-01-02T15:04:05.000Z07:00"},
		{PrecisionMicro, "2006-01-02T15:04:05.000000Z07:00"},
		{PrecisionNano, "2006-01-02T15:04:05.000000000Z07:00"},
	}
	for _, tc := range cases {
		want := fmt.Sprintf(`{"time":"%s","level":"INFO"`, expectedTime(nanos, tc.layout))
		for run := 0; run < 3; run++ { // first run builds caches, later runs hit them
			if got := headerOf(tc.p, nanos, types.InfoLevel); got != want {
				t.Fatalf("precision %d run %d:\n got: %s\nwant: %s", tc.p, run, got, want)
			}
		}
	}
}

// TestFusedHeader_LevelAndSecondRollover verifies the fused cache serves every
// level correctly and rebuilds when the second changes.
func TestFusedHeader_LevelAndSecondRollover(t *testing.T) {
	base := time.Date(2026, 7, 7, 11, 0, 0, 0, time.Local).UnixNano()
	for lvl := types.LogLevel(0); lvl <= 6; lvl++ {
		got := headerOf(PrecisionSecond, base, lvl)
		if !strings.HasSuffix(got, levelCache[lvl]) {
			t.Fatalf("level %d: header %q missing %q", lvl, got, levelCache[lvl])
		}
	}
	next := base + int64(time.Second)
	a := headerOf(PrecisionSecond, base, types.InfoLevel)
	b := headerOf(PrecisionSecond, next, types.InfoLevel)
	if a == b {
		t.Fatalf("second rollover not reflected: %q", a)
	}
	if headerOf(PrecisionSecond, base, types.InfoLevel) != a {
		t.Fatal("returning to a cached second must reproduce its header")
	}
}

// TestHeaderTimestamp_UTCWidth is the regression test for the fixed-width cache
// bug: the old cache-hit path appended a hardcoded 25 bytes, which corrupted
// "Z"-suffixed (20-byte) UTC timestamps on cache hits. The engine must render
// UTC timestamps identically on build and on hit.
func TestHeaderTimestamp_UTCWidth(t *testing.T) {
	nanos := time.Date(2026, 7, 7, 12, 0, 1, 0, time.Local).UnixNano()
	want := headerOf(PrecisionSecond, nanos, types.InfoLevel) // builds the cache
	got := headerOf(PrecisionSecond, nanos, types.InfoLevel)  // hits the cache
	if got != want || !strings.HasSuffix(want, `","level":"INFO"`) {
		t.Fatalf("cache hit differs from build:\n got: %q\nwant: %q", got, want)
	}
	if strings.Contains(want, "\x00") {
		t.Fatalf("header contains garbage bytes: %q", want)
	}
}

// TestPrecisionFormatterEndToEnd verifies a formatter constructed with a
// precision renders that precision through Format and stays field-correct.
func TestPrecisionFormatterEndToEnd(t *testing.T) {
	entry := goldenEntry()
	out := string(NewJsonFormatterWithPrecision(PrecisionMilli).Format(entry, nil))
	wantTime := expectedTime(entry.TimestampUnix, "2006-01-02T15:04:05.000Z07:00")
	if !strings.HasPrefix(out, `{"time":"`+wantTime+`"`) {
		t.Fatalf("milli timestamp wrong: %q (want prefix time %q)", out[:60], wantTime)
	}
	if !strings.HasSuffix(out, goldenFields) {
		t.Fatalf("fields changed under precision option: %q", out)
	}
}

// TestHeaderZeroAlloc guards the header engine: steady-state rendering at every
// precision must not allocate.
func TestHeaderZeroAlloc(t *testing.T) {
	nanos := time.Now().UnixNano()
	buf := make([]byte, 0, 128)
	for p := PrecisionSecond; p < precisionCount; p++ {
		ha := headerAppenders[p]
		ha(buf[:0], nanos, types.InfoLevel) // warm the caches
		allocs := testing.AllocsPerRun(500, func() {
			buf = ha(buf[:0], nanos, types.InfoLevel)
		})
		if allocs != 0 {
			t.Fatalf("precision %d header allocates: %v allocs/op", p, allocs)
		}
	}
}

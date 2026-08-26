# HaloLog — Comprehensive Multi-Logger Benchmark Comparison

**Measured 2026-08-26 · Go 1.27.0 · Intel Core Ultra 9 285HX · 5 runs × 1 s per scenario · benchstat medians**

This document is the canonical, reproducible record of HaloLog's standing
against the major Go structured loggers. It supersedes every earlier results
file in this directory (`benchmark_comparison.md`, `final_.md`,
`benchmark_results*.txt`, `simple_bench_result.txt`), which predate the v1.0
core and measured internal prototypes that no longer exist.

## Method — what makes this comparison fair

- Every logger emits a **full structured JSON record** — timestamp + level +
  message + fields — to `io.Discard`. No logger gets a disabled-output shortcut
  in a comparative row.
- Field values and counts are identical per scenario; each logger uses its own
  idiomatic API (zerolog `Int()`, zap `zap.Int()`, slog `slog.Int()`, phuslu
  `Int()`, HaloLog typed/keyed setters).
- The field includes **phuslu/log v1.0.128** — the fastest logger on public Go
  leaderboards — because a speed claim that never races the champion is not a
  claim.
- 5 runs × 1 s each, summarized with `benchstat` (medians). Single machine;
  expect ±10–15% run-to-run drift and treat sub-5 ns deltas as ties.
- Linux numbers come from linux/amd64 (WSL2) — the CI environment; Windows
  numbers from windows/amd64 on the same hardware. Orderings agree.

Known biases, disclosed: HaloLog serializes writes behind a mutex (lines never
interleave) while zerolog and phuslu write unlocked — a safety cost HaloLog
pays that they do not. phuslu does **not escape field keys at all** (hostile
keys produce broken JSON); HaloLog escapes plain keys per call and pre-escapes
`Key()` keys once. In the twenty-field scenario zap's field slice is built
outside the timed loop (its idiomatic accumulated-fields style), which flatters
zap slightly. `HaloLog_DisabledOutput` rows measure the no-op sink and are
**never** comparable to real-output rows.

## Results — linux/amd64 (canonical)

### Bare message

| Logger | ns/op | allocs | vs HaloLog |
|---|---:|---:|---:|
| **HaloLog** | **23.6** | 0 B, 0 | — |
| phuslu/log | 61.9 | 0 B, 0 | 2.6× slower |
| zerolog | 86.7 | 0 B, 0 | 3.7× slower |
| zap | 143.7 | 0 B, 0 | 6.1× slower |
| slog (stdlib JSON) | 234.4 | 0 B, 0 | 9.9× slower |
| logrus | 933.7 | 797 B, 21 | 40× slower |

23.6 ns/op ≈ **42 million JSON log lines per second, single goroutine**.

### One field (string)

| Logger | ns/op | allocs |
|---|---:|---:|
| **HaloLog (typed)** | **34.4** | 0 B, 0 |
| HaloLog (WithField) | 43.8 | 0 B, 0 |
| phuslu/log | 66.6 | 0 B, 0 |
| zerolog | 98.4 | 0 B, 0 |
| zap | 183.0 | 64 B, 1 |
| slog | 314.0 | 48 B, 1 |
| logrus | 1411 | 1.5 KiB, 27 |

### Ten fields (int)

| Logger | ns/op | allocs |
|---|---:|---:|
| **HaloLog (pre-declared keys)** | **79.2** | 0 B, 0 |
| **HaloLog (typed)** | **100.8** | 0 B, 0 |
| phuslu/log | 113.3 | 0 B, 0 |
| HaloLog (WithField) | 132.8 | 0 B, 0 |
| zerolog | 157.6 | 0 B, 0 |
| zap | 399.6 | 706 B, 1 |
| slog | 921.1 | 689 B, 11 |
| logrus | 3782 | 3.4 KiB, 64 |

### Twenty fields (int)

| Logger | ns/op | allocs |
|---|---:|---:|
| **HaloLog (typed)** | **155.7** | 0 B, 0 |
| phuslu/log | 175.2 | 0 B, 0 |
| zerolog | 221.5 | 0 B, 0 |
| HaloLog (WithField) | 238.4 | 0 B, 0 |
| zap (fields prebuilt) | 346.3 | 0 B, 0 |

### Disabled level (not comparable to output rows)

| | ns/op |
|---|---:|
| HaloLog, level filtered | 0.84 |

## Results — windows/amd64 (same hardware, same day, final code)

| Scenario | HaloLog | phuslu | zerolog |
|---|---:|---:|---:|
| Bare message | **27.5** | 49.7 | 88.5 |
| One field (typed) | **47.3** | 57.8 | 94.7 |
| Ten fields (typed) | 141.4 | **121.8** | 170.9 |
| Ten fields (pre-declared keys) | 127.6 | — | — |
| Twenty fields (typed) | 258.2 | **212.4** | 279.2 |

The Windows host carries heavier ambient load and ~15% more run-to-run drift;
orderings at zero/one field match Linux, while phuslu keeps the ten/twenty-
field lead there. Full raw zap/slog/logrus rows for Windows are in the same
benchstat capture (zap 158/214/964 ns for bare/one/ten).

## Verdict

- **HaloLog wins every scenario on linux/amd64**, the CI environment — at ten
  and twenty typed fields by a clear ~11% over phuslu (100.8 vs 113.3;
  155.7 vs 175.2), decisively with pre-declared keys (79.2), and by 1.9–2.6×
  at zero/one field — at 0 B/op, 0 allocs/op everywhere.
- On Windows, HaloLog wins zero- and one-field decisively; phuslu leads the
  ten/twenty-field scenarios on that (noisier) host.
- zerolog is beaten in every scenario on both platforms. zap, slog, and logrus
  are not close (3–40× slower, and all three allocate once fields appear).

## Why these numbers moved — the three optimizations (2026-08-26)

Profiling the ten-field line showed **72% of wall time in field-capture
plumbing** (`withTyped` + `addField`: building a 56-byte `FieldValue` box per
field and passing it through a two-call funnel) versus ~14% in actual byte
encoding — while phuslu appends `,"key":value` bytes immediately per field.
Three commits closed the gap and then some:

Commits are referenced by subject (stable across any history maintenance);
find each with `git log --oneline --grep`.

1. **"perf(console): format outside the write lock; lock-free formatter
   access"** — the per-line `DirectEncoder()` query became a lock-free load
   (was a mutex pair), and formatting moved outside the write lock.
2. **"perf(core,json): per-type direct append"** — typed and Line setters for
   string/int/int64/float64/bool/error write bytes in one call on the direct
   path; the `FieldValue` box is built only where the capture path needs it.
   Ten-field typed 177→105 ns; twenty-field 328→172 ns.
3. **"perf(core): message-only lines take the direct byte path"** —
   `Info(msg)` renders fused-header + closer straight into the pooled line
   buffer, no `LogEntry` touched. Bare message 49→24 ns.
4. **"perf(json): SWAR word-at-a-time escape scanning"** — the clean-string
   scan processes eight bytes per fused load with branch-free
   hasless/haszero word tricks, falling back to the exact per-byte path on
   any dirty word. Byte-identical output (exhaustive oracle tests); 80-byte
   clean scan 23.5→14.3 ns, and it widened the Linux ten/twenty-field
   margins.

Byte output is unchanged (pinned by direct-vs-capture identity tests) and every
hot path stays 0 allocs/op under 7 committed allocation guards
(`go test ./core -run TestZeroAlloc`).

What phuslu still does differently: no write mutex (interleaving possible), no
key escaping (fast but unsafe for arbitrary keys), and per-line timestamp
formatting (~a third of their bare-message cost — HaloLog's fused per-second
header cache makes the same work a single memcpy, which is where the 2.6×
bare-message margin comes from).

## Reproduce

```bash
cd benchmarks
go test -bench='BenchmarkInfo$|BenchmarkOneField$|BenchmarkTenFields$|BenchmarkTwentyFields$|BenchmarkKeyed_TenFields$' \
  -benchmem -benchtime=1s -run='^$' -count=5 . | tee results.txt
go run golang.org/x/perf/cmd/benchstat@latest results.txt
```

The scenario definitions live in `comparison_bench_test.go` (six loggers) and
`keyed_bench_test.go` (pre-declared-key variants). CI runs the allocation
guards and reports these numbers per commit via `.github/workflows/bench.yml`.

<!--
@author Admilson B. F. Cossa
-->
# Changelog

All notable changes to HaloLog are documented here. This project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed (hardening)
- **Linux/macOS build restored** — `syslog_unix.go` referenced a package-local
  formatter deleted in an earlier refactor and had never compiled since; it now
  uses `types.Formatter` with the text formatter as default. Stale
  `amd64 || arm64` build tags on the (pure-Go) JSON/text formatters and a wrong
  package clause in the file-lock fallback are gone: the module now builds on
  windows, linux, darwin, 386, riscv64, and wasm.
- **Level filtering on the classic fluent API** — `WithField(...).Info/Debug/…`
  bypassed the level filter entirely; every FieldBuilder terminal now dispatches
  through the same `dispatchLine` as the typed and Line APIs, which enforces
  level, sampling, masking, and metrics identically everywhere.
- **Sampling is real** — `Builder.Sampling`/`Config.Sampler` stored the sampler
  but never consulted it. `ShouldSample` now gates every Trace–Error line on
  all APIs (message-only, FieldBuilder, Typed, Line); Fatal/Panic are never
  sampled away.
- **Pooled-state corruption from builder reuse** — finishing a builder twice
  double-released its pooled state, later handing one state to two live
  builders (silent wrong-field attribution). A generation (epoch) counter on
  the pooled state turns any use-after-terminal into a no-op; a second
  `Line.Msg` no longer nil-panics either. Zero measured cost.
- **Data races** — `SensitiveFieldRegistry` hit/miss/check counters (bumped on
  the query path under at most an RLock) are now atomic; the HTTP and file
  adapters' runtime-swappable formatters are now behind `atomic.Pointer`, so
  `SetFormatter` can no longer tear the interface value an in-flight write is
  reading.
- **HTTP retry aliasing** — re-queued failed entries aliased a pooled slice
  that the next flush overwrites; they are copied out before re-queueing. The
  never-actually-used `http.Request` pool was removed, and the per-entry
  scratch buffer is now reused across a batch.
- **Non-finite floats** — NaN/±Inf rendered as bare `NaN`/`+Inf` (invalid
  JSON); they now render as quoted strings, matching zerolog. Matters more
  under Go 1.27, whose json/v2-backed parsers are stricter.
- **Pre-epoch timestamps** — negative unix-nanos truncated toward zero and
  rendered garbage fractional digits (`.+00`); seconds now floor-divide with a
  non-negative remainder.
- **`processExists` never worked** — it sent a nil `os.Signal`, which the os
  package rejects on every platform, so live processes were judged dead and
  their file locks stolen as "stale". Unix now probes with `signal 0` (EPERM
  counts as alive), Windows opens a process handle, and other platforms
  conservatively never break locks.
- **Rotation shutdown race** — the async compress/cleanup goroutine is now
  tracked by the adapter's WaitGroup, so `Close` waits for it instead of racing
  file teardown.
- **Fatal/Panic semantics** — `Fatal` now writes, flushes, and exits (code 1,
  overridable via `Config.ExitFunc`); `Panic` writes and panics — the
  conventional contract shared by zap/zerolog/logrus/stdlib. Previously both
  just logged.
- **Consistency sweep** — the discard fast path and metrics counting are now
  symmetric across ALL levels (previously Info had a private shortcut and
  Debug/Error/Trace/Fatal/Panic skipped metrics); pooled entries re-establish
  their `StaticFields` backing on acquire and clear used field values on
  release (no stale-PII retention); `Flush`/`Close` aggregate all adapter
  errors with `errors.Join` instead of stopping at the first; ~80 inert
  `//go:inline` pseudo-directives (not a real compiler directive) removed.

### Added
- **Edge-case regression suite** (`tests/edge`) — 19 public-API tests pinning
  the hardening pass: level filtering on every fluent API, sampler
  consultation, JSON validity under hostile values (NaN/±Inf, control chars,
  invalid UTF-8-adjacent input, 2 MiB payloads, 100-field lines), builder
  use-after-terminal misuse, Fatal/Panic semantics, pre-epoch timestamps, and
  concurrent line integrity.
- **`log/slog` bridge** (`slogbridge`) — adopt HaloLog in slog-first codebases
  with one line: `slog.SetDefault(slog.New(slogbridge.New(logger)))`. Validated
  against the standard library conformance suite (`testing/slogtest`); groups are
  encoded as dot-joined key prefixes; one documented deviation (HaloLog stamps
  its own clock time on every line, so a zero `Record.Time` is never rendered).
- **Level-first Line API** — `logger.InfoLine().Str(key, v).WithInt(...).Msg(m)`
  fixes the level when the line opens, so a disabled level costs a single check:
  no state acquisition, no field encoding, zero allocations.
- **Timestamp precision option** — `json.NewJsonFormatterWithPrecision`
  (second/milli/micro/nano). The strategy is chosen once at construction; every
  precision renders with zero allocations. Sub-second accuracy is bounded by the
  cached clock's refresh interval (~10ms by default) — documented, not hidden.
- **Direct-append fast path** — with a single raw-capable JSON adapter and no
  masking/sampling, the typed and Line builders encode fields straight to bytes
  (statically dispatched, zerolog-style); pre-declared keys emit as one memcpy.
  Output is byte-identical to the capture path (test-pinned) and every other
  configuration falls back automatically, so masking can never be bypassed.
- **Fused header cache** — at second precision the entire
  `{"time":..,"level":..` header is served per (second, level) from a lock-free
  cache: one memcpy per line.
- **Benchmark CI** — a Linux workflow gates every zero-allocation guard and
  publishes the co-measured multi-logger numbers per commit.


- **Pre-declared field keys** (`halolog.Key`, typed builder `Str/Int/Bool/…`
  methods) whose JSON escaping is computed once, so the hot path emits a key with
  a single copy and no escaping or lookup — measured ~25% faster than plain string
  keys on field-heavy lines. Plain string keys remain escaped inline (fast and
  allocation-free for the short keys typical of logging).
- **Async ring adapter** (`adapters/outputs/asyncring`) — a bounded, lock-free
  multi-producer/single-consumer ring that moves serialization and I/O off the
  caller's goroutine for low, predictable caller latency. Records are copied into
  ring-owned storage; `OnFull` selects drop (default, wait-free) or block; `Close`
  drains losslessly.

### Changed
- The JSON formatter's small-integer cache now covers 0–511 (small counts, ports,
  the HTTP status range) and the typed and boxed int/uint field paths use it.

### Fixed
- **UTC timestamp corruption on cache hits** — the old cache-hit path appended a
  hardcoded 25 bytes, corrupting 20-byte `"Z"`-suffixed UTC timestamps (it would
  have fired on any UTC machine). The header engine caches the true length.
- **Pool memory pinning** — one pathological multi-megabyte line can no longer
  pin its grown buffer on the per-P pool: retained direct-path buffers are
  capped at 64KiB.
- **`adapters/middleware` async use-after-recycle** — the channel-based async
  adapter enqueued a pointer to the caller's pooled entry, which the logger
  recycles immediately, so the background writer could serialize an overwritten
  record. Each write now copies the record into a pooled, detached entry
  (allocation-free after warmup); the new lock-free `asyncring` adapter is the
  recommended high-throughput variant.

## [1.0.0] - 2026-07-07

First public release: a zero-allocation structured logging framework for Go.

### Added
- Console/stdout output adapter (`adapters/outputs/console`) — thread-safe and
  buffer-pooled — restoring basic console logging.
- A permanent zero-allocation guard (`TestZeroAlloc_Info` /
  `TestZeroAlloc_WithFields`) that fails the suite if the hot path ever allocates.
- Reusable CI (lint + race tests + coverage gate on Linux & Windows), release
  and goreleaser configuration.

### Fixed
- **`pool.EntryPool.Close()` infinite-loop hang** — it drained a `sync.Pool`
  waiting for `Get()` to return `nil`, but a pool with a non-nil `New` func never
  returns `nil`, so `Close()` (reachable via `Shutdown()`) spun forever. The
  drain loop is removed (a `sync.Pool` needs no manual draining).
- **`EnhancedPipelineStrategy.UpdateConfiguration` self-deadlock** — it held
  `ps.mu` while calling `selectOptimalPipeline`, which re-acquired the same
  non-reentrant mutex. Selection now has an unlocked inner variant.
- **`EnhancedPipelineStrategy` `atomic.Value` panic** — the three concrete
  pipeline types were stored into one `atomic.Value`, which panics on a type
  change (triggered by reconfiguration). They are now boxed in a single wrapper.
- **Field-capture bugs** — `Entry.ToLogEntry` dropped all fields when the target
  had no backing buffer; the extracted pool returned zero-capacity static buffers
  and its field builder never wrote fields. All fixed.

### Changed
- **Reconciled the mid-refactor API.** `core` had been migrated from a registry
  (`GetLogger`) to a fluent builder (`core.New()...MustBuild()`), leaving the
  root and CLI calling removed functions. The root `halolog.GetLogger`/
  `GetLoggerWithConfig` are reimplemented as a name-keyed singleton registry over
  the real builder; the benchmark CLI, version shims, examples and sandbox are
  parked under `_parked/`.
- **Removed dead scaffolding**: the `HaloLogger` god-object (a 45-field exported
  struct never constructed) and the `LoggerProvider`/`LoggerFactory`/
  `LoggerRegistry` interfaces that only existed to return it. The real logger is
  `core.Logger`.
- Renamed the stuttering `json.JSONFormatter` / `text.TextFormatter` to
  `json.Formatter` / `text.Formatter`.

### Performance
- Hot path remains (and improved to) **0 allocations per call**: `Info`
  ~2 ns/op, a two-field `WithField` chain ~29 ns/op, both **0 B/op, 0 allocs/op**.

### Quality
- `golangci-lint` reports zero issues; build, vet, race and the full test suite
  are green.

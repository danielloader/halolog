<!--
@author Admilson B. F. Cossa
-->
# Changelog

All notable changes to HaloLog are documented here. This project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **Pre-declared field keys** (`halolog.Key`, typed builder `Str/Int/Bool/…`
  methods) whose JSON escaping is computed once, so the hot path emits a key with
  a single copy and no escaping or lookup. The plain string-key API is also
  auto-cached by the JSON formatter after first use (lock-free, 0-alloc).
- **Async ring adapter** (`adapters/outputs/asyncring`) — a bounded, lock-free
  multi-producer/single-consumer ring that moves serialization and I/O off the
  caller's goroutine for low, predictable caller latency. Records are copied into
  ring-owned storage; `OnFull` selects drop (default, wait-free) or block; `Close`
  drains losslessly.

### Changed
- The JSON formatter's small-integer cache now covers 0–511 (small counts, ports,
  the HTTP status range) and the typed and boxed int/uint field paths use it.

### Fixed
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

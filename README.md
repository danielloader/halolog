# HaloLog - Zero-Allocation Logging Framework for Go

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Performance](https://img.shields.io/badge/performance-15M%20logs%2Fsec-red.svg)](docs/PERFORMANCE.md)
[![Zero Allocation](<https://img.shields.io/badge/allocation-zero%20(0%20B%2Fop)-brightgreen.svg>)](docs/PERFORMANCE.md)

A high-performance logging framework for Go with zero-allocation design, structured logging, and enterprise-grade features.

## 🚀 Key Features

- **Zero-allocation hot path** - 15M+ logs/second throughput with 0 B/op
- **Structured logging** - Type-safe field handling with auto-inference
- **Multiple output adapters** - Console, file, HTTP, syslog, custom
- **PII masking** - Automatic sensitive data protection with regex patterns
- **Field encryption** - AES-256-GCM field-level encryption
- **Adaptive sampling** - Load-aware log sampling with configurable strategies
- **Alert integration** - Slack, PagerDuty, and webhook notifications
- **Configuration management** - YAML, JSON, environment variables
- **Thread-safe** - Lock-free design with per-P state architecture
- **Enterprise compliance** - GDPR, HIPAA, SOX, PCI-DSS ready

## 📦 Installation

```bash
go get github.com/go-gen-ecosystem/halolog
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "github.com/go-gen-ecosystem/halolog"
)

func main() {
    // Get a named logger (singleton, no cleanup needed)
    logger := halolog.GetLogger("my-app")

    // Basic logging
    logger.Info("Server started")
    logger.Debug("Debug message")
    logger.Warn("Warning message")
    logger.Error("Error occurred")

    // With fields
    logger.WithField("user_id", 12345).
        WithField("action", "login").
        Info("User logged in")

    // With error
    logger.WithError(err).
        WithField("operation", "database_query").
        Error("Database operation failed")
}
```

### Field keys: string keys vs. pre-declared keys

HaloLog offers two ways to attach fields; both are correct and both are
zero-allocation. Choose based on how hot the path is.

**String keys (ergonomic — the default).** `WithField`/`WithString` take a plain
string key, escaped inline by the JSON formatter. For the short keys typical of
logging this is already fast and allocation-free. Use it everywhere you value
convenience.

```go
logger.WithField("user_id", 12345).WithField("action", "login").Info("login")
```

**Pre-declared keys (fastest — for the hottest loops).** Declare each key once,
typically as a package-level var; its escaping is computed a single time and the
hot path emits it with one copy and **no lookup at all**. This is the zerolog/zap
"pre-declared field" pattern.

```go
// declared once, reused forever
var (
    userID = halolog.Key("user_id")
    action = halolog.Key("action")
)

logger.Typed().Str(userID, "alice").Str(action, "login").Info("login")
// keyed typed methods: Str, Int, Int64, Float64, Bool, Err, Any
```

Rule of thumb: reach for `halolog.Key(...)` in tight, high-frequency logging
loops; use string keys everywhere else. Neither allocates on the hot path.

### Level-first lines (cheapest disabled logging)

`InfoLine`/`DebugLine`/`WarnLine`/`ErrorLine` fix the level when the line opens,
so a filtered-out level costs a single check — no state, no encoding, zero
allocations:

```go
logger.InfoLine().Str(keyUser, "alice").WithInt("status", 200).Msg("handled")
logger.DebugLine().WithString("dump", expensive()).Msg("trace") // ~1ns when Debug is off*
```

\* the level check itself; argument evaluation is still yours to guard.

### Using HaloLog from log/slog

Slog-first codebases switch backends with one line — all existing `slog` call
sites keep working:

```go
import "github.com/go-gen-ecosystem/halolog/slogbridge"

slog.SetDefault(slog.New(slogbridge.New(logger)))
slog.Info("handled", "status", 200, slog.Group("req", "id", "abc"))
// {"time":"...","level":"INFO","message":"handled","status":200,"req.id":"abc"}
```

The bridge passes the standard library's `testing/slogtest` conformance suite
(groups are dot-joined; HaloLog stamps its own clock time on every line).

### Timestamp precision

The JSON formatter renders whole seconds by default (fastest — the header is a
single cached memcpy). For trace correlation, pick a sub-second resolution:

```go
f := json.NewJsonFormatterWithPrecision(json.PrecisionMilli) // .123
// PrecisionSecond | PrecisionMilli | PrecisionMicro | PrecisionNano
```

Every precision is zero-allocation. Honest bound: the default cached clock
refreshes every 10ms, so displayed sub-second digits carry up to ~10ms of
wall-clock skew — sufficient for in-service ordering; run a finer
`cache.NewCachedClock` interval if you need tighter accuracy.

### Advanced Configuration

```go
import (
    "github.com/go-gen-ecosystem/halolog"
    "github.com/go-gen-ecosystem/halolog/config"
    "github.com/go-gen-ecosystem/halolog/types"
)

// Create with custom configuration
cfg := &config.ImmutableConfig{
    Level:         types.InfoLevel,
    EnableMetrics: true,
}
logger := halolog.GetLoggerWithConfig("my-app", cfg)
```

## 🔌 Output Adapters

### Console

```go
import "github.com/go-gen-ecosystem/halolog/adapters/outputs/console"

adapter := console.New()                          // writes to os.Stdout
// adapter := console.NewWithWriter(w, formatter) // custom writer / formatter
```

### File with Rotation

```go
import "github.com/go-gen-ecosystem/halolog/adapters/outputs/file"

adapter, _ := file.NewFileAdapter("app.log", &file.RotationConfig{
    MaxSize:    100 * 1024 * 1024, // 100MB
    MaxBackups: 10,
    Compress:   true,
})
```

### HTTP/Webhook

```go
import "github.com/go-gen-ecosystem/halolog/adapters/outputs/http"

adapter := http.NewHTTPAdapterWithOptions(&http.HTTPAdapterOptions{
    URL:           "https://logs.example.com/ingest",
    BatchSize:     100,
    FlushInterval: 5 * time.Second,
})
```

### Async ring (low caller latency)

Wrap any destination to move serialization and I/O off the calling goroutine.
Producers copy each record into a bounded, lock-free ring and return immediately;
a single background goroutine serializes and writes. This optimizes for low,
predictable **caller latency** (not total throughput, which is bounded by the one
writer). The record is copied into ring-owned storage, so it is safe even though
the logger recycles its entry immediately.

```go
import (
    "os"
    "github.com/go-gen-ecosystem/halolog/adapters/outputs/asyncring"
    jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
)

adapter, _ := asyncring.New(asyncring.Options{
    Writer:    os.Stdout,
    Formatter: jsonfmt.NewJsonFormatter(),
    Capacity:  1024,             // rounded up to a power of two
    OnFull:    asyncring.Drop,   // or asyncring.Block
})
defer adapter.Close()           // drains everything already accepted
// adapter.Dropped() reports records dropped under overload (OnFull=Drop)
```

Caveat: field values are captured by shallow copy. Log values, not mutable
references — a `WithField("x", &mutableStruct)` may be serialized later by the
background goroutine.

### Syslog

```go
import "github.com/go-gen-ecosystem/halolog/adapters/outputs/syslog"

adapter, _ := syslog.NewSyslogAdapter("localhost:514", "myapp")
```

## 🔒 Security Features

### PII Masking

```go
import "github.com/go-gen-ecosystem/halolog/masking"

masker := masking.NewPIIMasker()
masker.AddPattern("email", `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, "[EMAIL]")
masker.AddPattern("ssn", `\b\d{3}-\d{2}-\d{4}\b`, "[SSN]")
```

### Field Encryption

```go
// Initialize global encryptor
masking.InitGlobalEncryptor("your-32-byte-secret-key")

// Use encrypted fields
logger.WithEncryptedField("ssn", "123-45-6789").
    WithEncryptedField("credit_card", "4111-1111-1111-1111").
    Info("Payment processed")
```

## 📊 Sampling

Wire a sampler into the logger with `Sampling(...)`; every Trace–Error line is
then offered to `ShouldSample` before it is written. Fatal and Panic lines are
**never** sampled away — the last line before a crash always lands. Configuring
a sampler keeps the capture path (it disables the direct-append fast path,
since each line must be inspected).

```go
logger := core.New().
    Level(types.InfoLevel).
    Adapter(myAdapter).
    Sampling(sampler).
    MustBuild()
```

### Count-Based Sampling

```go
import "github.com/go-gen-ecosystem/halolog/sampling"

sampler := sampling.NewSamplingManager(sampling.SamplingConfig{
    Strategy:            sampling.SampleByCount,
    SamplingDenominator: 10, // Sample 10%
})
```

### Adaptive Sampling

```go
sampler := sampling.NewAdaptiveSampler(sampling.AdaptiveConfig{
    BaseRate:         0.1,   // 10% base rate
    ErrorBoost:       10.0,  // 10x for errors
    TargetThroughput: 10000, // Adjust based on load
})
```

## 🚨 Alert Integration

### Slack Alerts

```go
import "github.com/go-gen-ecosystem/halolog/types"

sender := types.NewSlackSender(types.SlackAlertConfig{
    WebhookURL: "https://hooks.slack.com/services/...",
    Channel:    "#alerts",
    RateLimit:  time.Minute,
})

sender.Send(&types.AlertPayload{
    Level:   "error",
    Message: "Critical error occurred",
})
```

### PagerDuty

```go
sender := types.NewPagerDutySender(types.PagerDutyAlertConfig{
    IntegrationKey: "your-integration-key",
    Severity:       "critical",
})
```

## ⚙️ Configuration Management

### YAML Configuration

```go
loader := config.NewConfigLoader()
cfg, err := loader.LoadFromYAML([]byte(`
level: info
output: multi
format: json
file:
  path: app.log
  rotation:
    max_size: 104857600
    max_age: 604800
    compress: true
alerts:
  slack:
    webhook_url: https://hooks.slack.com/services/...
    threshold: error
`))
```

### JSON Configuration

```go
cfg, err := loader.LoadFromJSON([]byte(`{
    "level": "info",
    "output": "multi",
    "format": "json",
    "masking": {
        "enabled": true,
        "patterns": [
            {"name": "email", "pattern": "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b", "mask": "[EMAIL]"}
        ]
    }
}`))
```

### Environment Variables

```bash
export HALOLOG_LEVEL=info
export HALOLOG_FORMAT=json
export HALOLOG_OUTPUT=multi
export HALOLOG_FILE_PATH=/var/log/myapp.log
```

## 🏁 Performance Benchmarks

Measured on Go 1.27.0, linux/amd64 (the CI environment; Intel Core Ultra 9
285HX), 5 runs × 1s, benchstat medians. Every logger writes a full structured
JSON line (timestamp + level + message + fields) to `io.Discard` via
`benchmarks/` — the committed, fairness-audited comparison suite that now
includes phuslu/log, the fastest logger on public Go leaderboards. Run it
yourself; numbers vary by machine.

```
                       HaloLog   phuslu   zerolog     zap     slog   logrus
Bare message           46.7 ns   61.2 ns   84.5 ns  142.7 ns  231.6   927 ns
One field              44.8 ns   66.3 ns   95.2 ns  177.2 ns  304.1  1396 ns
Ten fields            139.4 ns  109.2 ns  156.3 ns  396.8 ns  889.4  3713 ns
Twenty fields         236.7 ns  177.2 ns  214.7 ns  351.2 ns      —       —
Disabled level        0.82 ns         —        —        —       —        —
HaloLog allocations   0 B/op, 0 allocs/op in every scenario
```

Honest summary: **HaloLog is the fastest logger in this field for typical log
lines (zero to a few fields) on both linux/amd64 and windows/amd64** — ~25–30%
ahead of phuslu and ~45–50% ahead of zerolog — and allocation-free at every
field count. On very field-heavy lines (ten or more), phuslu leads and zerolog
is competitive; HaloLog beats zerolog at ten fields and trails it at twenty.
Every hot path is guarded at **0 allocs/op** by
`go test ./core -run TestZeroAlloc`.

## 📊 Comparison with Other Loggers

| Feature            | HaloLogger              | Zerolog           | phuslu/log        | Zap             | Logrus            |
| ------------------ | ----------------------- | ----------------- | ----------------- | --------------- | ----------------- |
| **Bare message**   | **46.7 ns · 0 B**       | 84.5 ns · 0 B     | 61.2 ns · 0 B     | 142.7 ns · 0 B  | 927 ns · 797 B    |
| **One field**      | **44.8 ns · 0 B**       | 95.2 ns · 0 B     | 66.3 ns · 0 B     | 177.2 ns · 64 B | 1396 ns · 1.5 KiB |
| **Ten fields**     | 139.4 ns · **0 B**      | 156.3 ns · 0 B    | **109.2 ns** · 0 B| 396.8 ns · 706 B| 3713 ns · 3.4 KiB |
| **Disabled level** | **0.8 ns**              | ~1 ns             | ~1 ns             | ~2 ns           | ~15 ns            |
| **PII Masking**    | **✅ Built-in**         | ❌ External       | ❌ External       | ❌ External     | ❌ External       |
| **File Rotation**  | **✅ Built-in**         | ❌ External       | ✅ Built-in       | ❌ External     | ❌ External       |
| **Sampling**       | **✅ Adaptive**         | ✅ Basic          | ❌ Manual         | ✅ Basic        | ❌ Manual         |
| **Alerting**       | **✅ Integrated**       | ❌ External       | ❌ External       | ❌ External     | ❌ External       |
| **Encryption**     | **✅ Field-level**      | ❌ External       | ❌ External       | ❌ External     | ❌ External       |
| **Configuration**  | **✅ Multi-format**     | ❌ Code-only      | ❌ Code-only      | ❌ Code-only    | ❌ Code-only      |

### Fatal and Panic semantics

`Fatal(...)` writes the line, flushes every adapter, then calls `os.Exit(1)`;
`Panic(...)` writes the line, then panics with the message — the same contract
as zap, zerolog, logrus, and the standard library. Tests and embedders can
intercept termination with `core.Config.ExitFunc`.

## 📚 Documentation

- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation
- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design and patterns
- **[Performance Guide](docs/PERFORMANCE.md)** - Optimization and benchmarks
- **[Security Guide](docs/SECURITY.md)** - Security and compliance
- **[Examples](docs/EXAMPLES.md)** - Usage examples and patterns
- **[Getting Started](docs/GETTING_STARTED.md)** - Installation and setup

## 🧪 Testing

### Unit Testing

```go
func TestUserService(t *testing.T) {
    // Create test logger with discard adapter
    logger := halolog.GetLogger("test")

    service := NewUserService(logger)

    // Capture logs for verification
    captured := logger.StartCapture(types.CaptureConfig{
        MaxEntries: 100,
        Filter: func(entry *types.LogEntry) bool {
            return entry.Level >= types.InfoLevel
        },
    })

    // Run your test
    err := service.CreateUser("test@example.com")

    // Verify logs
    captured.Replay(func(entry *types.LogEntry) {
        if entry.Message == "User created" {
            assert.Equal(t, "test@example.com", entry.Fields["email"])
        }
    })
}
```

### Benchmark Testing

```go
func BenchmarkLogging(b *testing.B) {
    logger := halolog.GetLogger("benchmark")

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        logger.WithField("iteration", i).
            WithField("data", "test data").
            Info("Benchmark message")
    }
}
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

HaloLogger is released under the [Apache License 2.0](LICENSE).

---

<div align="center">

**🚀 HaloLogger - Zero-allocation logging for Go applications**

**[⬆ Back to Top](#-halolog---zero-allocation-logging-framework-for-go)**

</div>

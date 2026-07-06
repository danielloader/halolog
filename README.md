# HaloLog - Zero-Allocation Logging Framework for Go

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
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
import "github.com/go-gen-ecosystem/halolog/core"

adapter := core.NewConsoleAdapter(os.Stdout, nil)
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

```
BenchmarkLogger_Info-20                    100000000    6.5 ns/op    0 B/op    0 allocs/op
BenchmarkLogger_WithField-20                 50000000    12.3 ns/op    0 B/op    0 allocs/op
BenchmarkLogger_5Fields-20                   30000000    45.2 ns/op    0 B/op    0 allocs/op
BenchmarkLogger_Parallel8-8                  50000000    25.3 ns/op    0 B/op    0 allocs/op
```

## 📊 Comparison with Other Loggers

| Feature            | HaloLogger          | Zap           | Logrus          | Zerolog       |
| ------------------ | ------------------- | ------------- | --------------- | ------------- |
| **Performance**    | **15M/sec**         | 5M/sec        | 500K/sec        | 10M/sec       |
| **Allocation**     | **Zero (0 B/op)**   | Low (32 B/op) | High (256 B/op) | Low (16 B/op) |
| **Latency**        | **6.5ns**           | 15ns          | 200ns           | 12ns          |
| **PII Masking**    | **✅ Built-in**     | ❌ External   | ❌ External     | ❌ External   |
| **File Rotation**  | **✅ Built-in**     | ❌ External   | ❌ External     | ❌ External   |
| **Smart Sampling** | **✅ Adaptive**     | ❌ Manual     | ❌ Manual       | ❌ Manual     |
| **Dynamic Levels** | **✅ Runtime**      | ❌ Static     | ❌ Static       | ❌ Static     |
| **Alerting**       | **✅ Integrated**   | ❌ External   | ❌ External     | ❌ External   |
| **Encryption**     | **✅ Field-level**  | ❌ External   | ❌ External     | ❌ External   |
| **Configuration**  | **✅ Multi-format** | ❌ Code-only  | ❌ Code-only    | ❌ Code-only  |

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
        logger.Info("Benchmark message").
            WithField("iteration", i).
            WithField("data", "test data")
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

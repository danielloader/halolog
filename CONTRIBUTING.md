# Contributing to HaloLog

Thank you for your interest in contributing to HaloLog. This document outlines the standards and processes for contributing to this zero-allocation logging library.

## 🚀 Getting Started

```bash
git clone https://github.com/go-gen-ecosystem/halolog.git
cd halolog
go mod download

# Run full validation suite
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## 📋 Development Requirements

### Environment

- **Go 1.21+** (required for atomic operations and generics)
- **Git** for version control
- **Go modules** enabled

### Quality Gates

- ✅ **All tests passing**: `go test ./...`
- ✅ **Zero race conditions**: `go test -race ./...`
- ✅ **Zero allocations on hot path**: Verified via benchmarks
- ✅ **Code coverage ≥80%**: `go test -coverprofile=coverage.out ./...`
- ✅ **Performance regression check**: Benchmark comparison required

## 🎯 Code Standards

### Performance Requirements (Non-Negotiable)

| Scenario                 | Target Latency | Allocation Target | Status      |
| ------------------------ | -------------- | ----------------- | ----------- |
| `logger.Info()`          | < 10ns         | 0 B/op            | ✅ Required |
| `WithField().Info()`     | < 15ns         | 0 B/op            | ✅ Required |
| 10 fields                | < 50ns         | 0 B/op            | ✅ Required |
| `WithFields(...).Info()` | < 100ns        | 0 B/op            | ✅ Required |

### Architecture Standards

#### Hexagonal Architecture

- **Domain layer**: Pure business logic, no external dependencies
- **Application layer**: Use cases and orchestration
- **Infrastructure layer**: Adapters, external integrations
- **Interface layer**: HTTP, CLI, external APIs

#### Design Patterns

- **Strategy Pattern**: Pipeline selection, sampling strategies
- **Factory Pattern**: Logger creation, adapter instantiation
- **Repository Pattern**: Configuration management
- **Unit of Work**: Transactional log operations

### Code Quality Guidelines

#### Style & Formatting

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Run `gofmt -s -w .` before committing
- **Function length**: Keep under 50 lines where practical
- **Cyclomatic complexity**: < 10 for hot paths
- **Documentation**: All exported symbols must be documented

#### Performance-Critical Code

```go
// ✅ GOOD: Zero-allocation pattern
func (l *Logger) Info(msg string) {
    entry := pool.GlobalPool.AcquireEntry() // Reuse from pool
    defer pool.GlobalPool.ReleaseEntry(entry)
    // ... processing
}

// ❌ BAD: Allocation in hot path
func (l *Logger) Info(msg string) {
    entry := &LogEntry{} // ❌ Allocates new object
    // ... processing
}
```

#### Error Handling

```go
// ✅ GOOD: Structured error handling
func processEntry(entry *LogEntry) error {
    if entry == nil {
        return fmt.Errorf("nil entry: %w", ErrInvalidEntry)
    }
    // ... processing
    return nil
}

// ❌ BAD: Silent error swallowing
func processEntry(entry *LogEntry) {
    if entry == nil {
        return // ❌ Error not propagated
    }
    // ... processing
}
```

## 🔄 Pull Request Process

### Pre-Submission Checklist

- [ ] **Feature branch created**: `git checkout -b feature/descriptive-name`
- [ ] **Tests written**: Unit tests for new functionality
- [ ] **Benchmarks updated**: Performance impact measured
- [ ] **Documentation updated**: API docs and examples
- [ ] **Memory profiling**: `go test -memprofile=mem.prof -bench=. ./...`
- [ ] **Race condition check**: `go test -race ./...`

### Submission Process

1. **Fork and branch**: Create feature branch from `main`
2. **Implement with tests**: TDD approach preferred
3. **Performance validation**: Run benchmarks before/after
4. **Documentation**: Update relevant docs
5. **Submit PR**: Use provided template

### PR Template

```markdown
## Description

Brief description of changes

## Type of Change

- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (requires migration)
- [ ] Performance improvement
- [ ] Documentation update

## Performance Impact

- Benchmark results (before/after)
- Allocation impact
- Latency changes

## Testing

- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Benchmark regression check
- [ ] Memory profiling completed

## Checklist

- [ ] Code follows project standards
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added for new functionality
```

## 📝 Commit Message Standards

Use **Conventional Commits** with enterprise context:

```
feat(core): add adaptive sampling strategy for high-throughput scenarios

- Implements exponential backoff for sampling decisions
- Reduces CPU usage by 15% in benchmark tests
- Maintains zero-allocation guarantee

Performance: 12.3ns → 10.8ns improvement
Allocations: 0 B/op (verified)
```

### Commit Types

- `feat`: New feature (minor version bump)
- `fix`: Bug fix (patch version bump)
- `perf`: Performance improvement
- `docs`: Documentation changes
- `test`: Test additions/updates
- `refactor`: Code restructuring (no behavior change)
- `style`: Formatting, naming (no logic change)
- `chore`: Build, dependencies, tooling

## 🧪 Testing Standards

### Test Categories

```bash
# Unit tests - Core functionality
go test ./core/... ./types/... ./config/...

# Integration tests - Adapter interactions
go test ./adapters/... ./pipeline/...

# Performance tests - Benchmark validation
go test -bench=. -benchmem ./benchmarks/...

# Race condition tests - Concurrent safety
go test -race ./...

# Memory profiling - Allocation analysis
go test -memprofile=mem.prof -bench=BenchmarkLogger ./core/
```

### Benchmark Requirements

```bash
# Before submitting performance-critical changes
go test -bench=. -benchmem ./core/ > bench-before.txt
# Make changes
go test -bench=. -benchmem ./core/ > bench-after.txt
# Compare results - regression not allowed
```

### Test Coverage Expectations

- **Core packages**: ≥90% coverage
- **Adapters**: ≥85% coverage
- **Configuration**: ≥95% coverage
- **Performance-critical**: 100% hot path coverage

## 🏗️ Project Architecture

### Directory Structure

```
halolog/
├── core/                    # Core logger implementation
│   ├── logger.go           # Main logger types
│   ├── entry.go            # Log entry management
│   └── pipeline_builder.go # Pipeline construction
├── types/                   # Shared interfaces and types
│   ├── adapter.go          # Adapter interfaces
│   ├── config_types.go     # Configuration types
│   └── log_entry.go        # Entry structures
├── adapters/                # Output adapters
│   ├── formatters/         # JSON, text formatters
│   ├── outputs/           # File, HTTP, syslog outputs
│   └── middleware/        # Async processing
├── config/                  # Configuration management
│   ├── loader.go          # Config loading
│   └── defaults.go        # Default configurations
├── pipeline/               # Pipeline strategies
│   ├── pipeline_direct.go # Zero-field optimization
│   ├── pipeline_simple.go # Basic features
│   └── pipeline_full.go   # All features
├── masking/               # PII masking and encryption
├── sampling/              # Sampling strategies
├── fielddict/             # Field dictionary for O(1) lookup
├── cache/                 # Cached clock system
├── pool/                  # Object pooling
├── interfaces/            # Interface definitions
├── utils/                 # Utility functions
├── examples/              # Usage examples
├── docs/                  # Documentation
├── tests/                 # Integration tests
└── benchmarks/           # Performance benchmarks
```

### Key Interfaces

```go
// Core logger interface
type Logger interface {
    Debug(msg string) Logger
    Info(msg string) Logger
    Warn(msg string) Logger
    Error(msg string) Logger
    Fatal(msg string) Logger
    WithField(key string, value interface{}) Logger
    WithFields(fields ...interface{}) Logger
}

// Adapter interface for outputs
type Adapter interface {
    Write(entry *LogEntry) error
    Close() error
    Name() string
}

// Pipeline strategy interface
type Pipeline interface {
    Process(entry *LogEntry) *LogEntry
    Name() string
}
```

## 🐛 Issue Reporting

### Bug Reports

Include this information for effective debugging:

````markdown
**Environment:**

- Go version: `go version`
- OS/Architecture: `go env GOOS GOARCH`
- HaloLog version: (from go.mod)

**Problem:**

- Expected behavior:
- Actual behavior:
- Frequency: (consistent/intermittent)

**Reproduction:**

```go
// Minimal code example
logger := halolog.NewLogger()
logger.Info("test") // Panics here
```
````

**Logs/Output:**

```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Performance Impact:**

- Allocation regression: (before/after benchmarks)
- Latency impact: (ns measurements)

````

### Feature Requests
Use this template:
```markdown
**Use Case:**
Describe the business/technical need

**Proposed Solution:**
Technical approach and API design

**Performance Requirements:**
- Latency target:
- Allocation target:
- Throughput requirement:

**Breaking Changes:**
Will this require migration?

**Alternatives Considered:**
Other approaches evaluated
````

## 📊 Performance Monitoring

### Continuous Benchmarking

We track these metrics on every PR:

- **Latency**: ns/op for core operations
- **Allocations**: B/op (must remain 0 for hot path)
- **Throughput**: logs/sec capacity
- **Memory**: Peak memory usage
- **GC pressure**: GC cycles per operation

### Regression Detection

```bash
# Automated performance comparison
benchcmp bench-main.txt bench-pr.txt

# Memory profiling comparison
go tool pprof -base mem-main.prof mem-pr.prof
```

## 🔒 Security Considerations

### Code Review Requirements

- **PII handling**: All PII must use masking interfaces
- **Encryption**: Field encryption for sensitive data
- **Audit logging**: Security events must be auditable
- **Input validation**: All external inputs validated
- **Error messages**: No sensitive data in errors

### Security Testing

```bash
# Static analysis
go vet ./...

# Security scanning
gosec ./...

# Dependency checking
go mod verify
```

## 📄 License

By contributing to HaloLog, you agree that your contributions will be licensed under the **Apache License 2.0**. All contributions must be original work or properly licensed third-party code.

## 📞 Getting Help

- **Technical questions**: Open a GitHub discussion
- **Bug reports**: Create a GitHub issue
- **Security issues**: Email security@halolog.dev
- **Performance regressions**: Tag with `performance` label

---

**Thank you for contributing to HaloLog.**

# Contributing to HaloLog

Thank you for helping improve HaloLog. Correctness, reproducibility, and honest
performance reporting are the project’s review standard.

## Prerequisites

- Go 1.24 or newer, matching the `go` directive in `go.mod`
- Git
- Go modules enabled

Clone the repository and download dependencies:

```bash
git clone https://github.com/Go-Gen-Ecosystem/halolog.git
cd halolog
go mod download
```

## Required checks

Run these before submitting a pull request:

```bash
GOWORK=off gofmt -w .
GOWORK=off go vet ./...
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./... -count=1
GOWORK=off go test ./core -run TestZeroAlloc -count=1
```

On Windows PowerShell, set `$env:GOWORK = "off"` first and run the Go commands
without the inline environment prefix.

The CI workflow repeats build, vet, tests, race detection, linting,
`govulncheck`, a 60% repository-wide coverage gate, and the zero-allocation
guards on Linux and Windows.

## Performance changes

Performance-sensitive changes should include before/after results gathered on
the same quiet machine. Use multiple samples and compare distributions rather
than a single best run:

```bash
cd benchmarks
go test -run='^$' -bench=. -benchmem -benchtime=1s -count=6 . > new.txt
go run golang.org/x/perf/cmd/benchstat@latest old.txt new.txt
```

Do not weaken an allocation guard to make a change pass. If a supported hot
path must allocate, explain the API and compatibility trade-off in the pull
request.

Benchmark claims must state the Go version, OS/architecture, CPU, command,
sample count, statistic used, and any asymmetry between implementations. Treat
small deltas as ties unless the samples establish otherwise.

## Pull requests

Keep each pull request focused and include:

- the problem and intended behavior;
- tests covering the behavior or regression;
- benchmark evidence for hot-path changes;
- documentation updates for user-visible behavior;
- any compatibility, security, or operational trade-offs.

Use conventional commit subjects where practical, for example
`fix(core): preserve level filtering on fluent calls`.

## Security

Do not report vulnerabilities in a public issue. Use
**Security → Report a vulnerability** on the GitHub repository so details stay
private while a fix is prepared. See [docs/SECURITY.md](docs/SECURITY.md).

## License

By contributing, you agree that your contribution is licensed under the
[Apache License 2.0](LICENSE). Submit only work you have the right to license.

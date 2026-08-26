# Documentation Map

Everything user-facing lives in this directory as flat Markdown; benchmark
evidence lives with the benchmarks.

```
README.md (repo root)                 # Overview, quick start, measured numbers
CHANGELOG.md (repo root)              # Versioned change history
benchmarks/comprehensive_comparison.md# Canonical benchmark record: method,
                                      # fairness notes, six-logger results,
                                      # reproduction commands

docs/
├── GETTING_STARTED.md                # First steps and common setups
├── API_REFERENCE.md                  # Public API surface
├── ARCHITECTURE.md                   # Packages, layers, and data flow
├── ZERO_ALLOCATION_DESIGN.md         # How the 0 allocs/op design works
├── PERFORMANCE.md                    # Fast-path guide + how to measure
├── EXAMPLES.md                       # Task-oriented recipes
├── SECURITY.md                       # PII masking, encryption, reporting
└── GO_PROPOSAL_INTERFACE_NOESCAPE.md # Upstream Go proposal notes
```

Internal planning notes, article drafts, and historical scratch are kept out
of the repository; prototype code that may return lives under `_parked/`
(excluded from release archives via `.gitattributes` export-ignore).

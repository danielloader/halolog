# Documentation Map

Only verified-accurate documentation is tracked. Everything user-facing lives
here as flat Markdown; benchmark evidence lives with the benchmarks.

```
README.md (repo root)                  # Overview, quick start, measured numbers
CHANGELOG.md (repo root)               # Versioned change history
benchmarks/comprehensive_comparison.md # Canonical benchmark record: method,
                                       # fairness notes, six-logger results,
                                       # reproduction commands

docs/
├── PERFORMANCE.md                     # Fast-path guide + how to measure
├── SECURITY.md                        # Reporting policy + shipped security features
└── DOCUMENTATION_STRUCTURE.md         # This map
```

Working conventions:

- **`TEMPDOCS/` (untracked, gitignored)** — local staging for drafts and docs
  awaiting verification against the current code. Nothing in it is published
  or tracked; a document graduates into `docs/` only after every claim and
  API reference in it has been checked.
- **Parked prototypes (local archive, untracked)** — superseded prototype
  *code* is kept in a local archive outside the published repository, for
  possible un-parking; it is not part of the repo or its history.

Planned graduations from TEMPDOCS: GETTING_STARTED, EXAMPLES, API_REFERENCE,
ARCHITECTURE (each needs a line-by-line pass against the v1.0 API), and a
rewritten zero-allocation design note reflecting the current direct-path
architecture.

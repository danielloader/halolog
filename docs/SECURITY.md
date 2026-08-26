# Security

## Reporting a vulnerability

Please report suspected vulnerabilities privately to the maintainer
(repository owner) rather than opening a public issue. Include a minimal
reproduction; you will receive an acknowledgment and a fix timeline. Please
allow a reasonable disclosure window before publishing details.

## Security-relevant features (implemented and tested)

- **PII masking** (`masking`): rule- and pattern-based masking applied to
  entries before any adapter sees them. **Design invariant, enforced in
  code and pinned by tests:** configuring a masker disables the
  direct-append fast path, so no optimization can ever bypass masking.
- **Sensitive-field registry** (`registry`): O(1) name/ID lookup over a
  pre-seeded set of sensitive field names, with pattern and heuristic
  fallbacks; query-path counters are atomic (race-detector clean).
- **Field-level encryption** (`masking/encryption.go`): AES-GCM field
  encryption with strict key-length validation (a short key is rejected,
  never silently padded — test-pinned).
- **File output hardening** (`adapters/outputs/file`): log files, rotated
  backups, compressed archives, and lock/PID files are created owner-only
  (0600); the cross-process file lock never treats a live process's lock as
  stale (probe via signal 0 / process handle, EPERM counts as alive).
- **JSON output robustness**: control characters, quotes, and backslashes
  are always escaped (the SWAR scanner is verified byte-identical to the
  reference scanner over every byte value in every position); non-finite
  floats render as quoted strings so a hostile value cannot corrupt the log
  stream for downstream parsers.
- **Misuse containment**: reusing a fluent builder after its terminal call
  is a guarded no-op (generation counter), not memory corruption or
  cross-request field leakage; pooled entries clear their used field values
  on release so recycled buffers never retain a previous line's data.

## Scope notes

- The HTTP adapter posts batches to the URL you configure; supplying
  credentials via headers and using TLS endpoints is the caller's
  responsibility.
- Sampling never drops FATAL/PANIC lines.
- This library does not itself provide tamper-evident (hash-chained) audit
  logs; if a document claims otherwise it is a draft, not shipped behavior.

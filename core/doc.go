// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package core provides the high-performance HaloLog logger implementation.
//
// # Architecture
//
// The core package implements the startup-time function composition pattern:
// log functions are built ONCE at startup based on configuration, eliminating
// all runtime feature checks from the hot path.
//
// # Zero Allocations
//
// The guarded logging hot paths achieve 0 B/op, 0 allocs/op through:
//   - Per-processor state pool for LogEntry reuse
//   - Value-type FluentBuilder (no heap escape)
//   - Pre-composed function pointers (no runtime branching)
//   - Concrete adapter types for discard optimization
//
// # Performance
//
// Results are machine- and version-specific. The repository's canonical,
// reproducible comparison is benchmarks/comprehensive_comparison.md.
//
// # Fluent Configuration API
//
//	logger := core.New().
//	    Component("my-service").
//	    Debug().
//	    Discard().
//	    Metrics().
//	    MustBuild()
//
// # Simple Logging
//
//	logger.Info("service started")
//	logger.Debug("processing request")
//	logger.Warn("connection pool low")
//	logger.Error("request failed")
//
// # Logging With Fields
//
//	logger.WithField("user_id", 123).
//	    WithField("action", "login").
//	    Info("user logged in")
//
// # Philosophy
//
// "If you asked for a blue painter, I give you ONLY a blue painter."
//
// Configuration happens ONCE at startup. The hot path contains ONLY what
// you configured - no runtime checks for features you don't use.
//
// Author: Admilson B. F. Cossa
package core

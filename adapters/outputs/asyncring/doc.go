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

// @author Admilson B. F. Cossa

// Package asyncring provides an asynchronous log sink backed by a bounded,
// lock-free multi-producer/single-consumer ring buffer.
//
// # Model
//
// Producers (the goroutines calling the logger) copy each record into a
// ring-owned slot and return immediately; a single background goroutine
// serializes the records and writes them to the destination off the caller's
// thread. This optimizes for low, predictable CALLER latency, not for
// serialization throughput — the JSON still costs the same to encode, it is just
// encoded on another goroutine. Throughput is bounded by that one writer.
//
// # Why a copy (and not a pointer)
//
// The logger reuses a per-P pooled entry and recycles it the instant Write
// returns. Enqueuing a POINTER to that entry — as a naive async adapter does — is
// a use-after-recycle bug: the background writer may serialize an entry that a
// later log call has already overwritten. RingAdapter copies each record's fields
// into its slot, so the pooled entry is free to be reused immediately.
//
// # Value-capture contract
//
// Field values are shallow-copied. String keys and values, and the boxing-free
// typed values (Str/Int/Bool/…), are immutable and always safe. An
// interface-wrapped MUTABLE reference (e.g. WithField("x", &myStruct)) is copied
// by reference; do not mutate such a value after logging it through the async
// ring, because the writer may serialize it later. Log values, not mutable
// references — the same caveat other async loggers carry.
//
// # Overload
//
// The ring is bounded. Under OnFull=Drop (default) a full ring makes the producer
// wait-free: the record is dropped and Dropped() is incremented. Under
// OnFull=Block the producer spins until a slot frees. Ordering is preserved (a
// single consumer). Close stops intake, drains everything already accepted
// (including in-flight producers), and stops the writer without losing a record.
package asyncring

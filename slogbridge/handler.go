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

// Package slogbridge adapts a HaloLog logger to the standard library's
// log/slog Handler interface, so slog-first codebases can adopt HaloLog by
// changing one line:
//
//	slog.SetDefault(slog.New(slogbridge.New(logger)))
//
// Mapping notes (all deliberate, all disclosed):
//   - Groups are encoded as dot-joined key prefixes ("group.key"), the common
//     convention for flat structured backends.
//   - The emitted line uses HaloLog's canonical keys ("message", "time"); the
//     record's own time is superseded by HaloLog's clock (≤10ms skew at the
//     default cached-clock interval).
//   - slog levels map onto HaloLog levels by threshold: <INFO → Debug,
//     <WARN → Info, <ERROR → Warn, otherwise Error.
//
// The bridge is a convenience adapter, not the zero-allocation hot path: group
// prefixing and attr resolution may allocate. Hot loops should use the native
// typed/Line APIs.
//
// @author Admilson B. F. Cossa
package slogbridge

import (
	"context"
	"log/slog"

	"github.com/go-gen-ecosystem/halolog/core"
	"github.com/go-gen-ecosystem/halolog/types"
)

// Handler implements slog.Handler on top of a HaloLog core.Logger. Handlers are
// immutable: WithAttrs and WithGroup return copies, so a Handler is safe for
// concurrent use by multiple goroutines.
type Handler struct {
	logger *core.Logger
	prefix string      // dot-joined open-group prefix, "" at the root
	bound  []boundAttr // attrs pre-bound via WithAttrs, with their prefix
}

// boundAttr pairs a pre-bound attr with the group prefix that was open when it
// was bound, preserving slog's WithGroup/WithAttrs ordering contract.
type boundAttr struct {
	prefix string
	attr   slog.Attr
}

// New returns a slog.Handler that forwards records to l.
func New(l *core.Logger) *Handler {
	return &Handler{logger: l}
}

// Enabled reports whether the mapped HaloLog level passes the logger's filter.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return mapLevel(level) >= h.logger.Level()
}

// Handle renders the record through HaloLog's typed builder.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	b := h.logger.Typed()
	for _, ba := range h.bound {
		b = appendAttr(b, ba.prefix, ba.attr)
	}
	r.Attrs(func(a slog.Attr) bool {
		b = appendAttr(b, h.prefix, a)
		return true
	})
	emit(b, mapLevel(r.Level), r.Message)
	return nil
}

// WithAttrs returns a copy of the handler with attrs bound under the currently
// open group prefix.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	nh := *h
	nh.bound = make([]boundAttr, 0, len(h.bound)+len(attrs))
	nh.bound = append(nh.bound, h.bound...)
	for _, a := range attrs {
		nh.bound = append(nh.bound, boundAttr{prefix: h.prefix, attr: a})
	}
	return &nh
}

// WithGroup returns a copy of the handler whose subsequent attrs are prefixed
// with name. Empty names are inlined per the slog.Handler contract.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := *h
	nh.prefix = h.prefix + name + "."
	return &nh
}

// appendAttr adds one resolved attr to the builder under prefix, recursing into
// groups. Empty attrs and empty groups are elided per the handler contract.
func appendAttr(b core.TypedFieldBuilder, prefix string, a slog.Attr) core.TypedFieldBuilder {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return b
	}
	if a.Value.Kind() == slog.KindGroup {
		gs := a.Value.Group()
		if len(gs) == 0 {
			return b // elide empty groups
		}
		p := prefix
		if a.Key != "" { // empty-keyed groups inline their attrs
			p = prefix + a.Key + "."
		}
		for _, ga := range gs {
			b = appendAttr(b, p, ga)
		}
		return b
	}
	return appendScalar(b, prefix+a.Key, a.Value)
}

// appendScalar adds one non-group value using the boxing-free typed setters.
func appendScalar(b core.TypedFieldBuilder, key string, v slog.Value) core.TypedFieldBuilder {
	switch v.Kind() {
	case slog.KindString:
		return b.WithString(key, v.String())
	case slog.KindInt64:
		return b.WithInt64(key, v.Int64())
	case slog.KindUint64:
		return b.WithAny(key, v.Uint64())
	case slog.KindFloat64:
		return b.WithFloat64(key, v.Float64())
	case slog.KindBool:
		return b.WithBool(key, v.Bool())
	case slog.KindDuration:
		return b.WithString(key, v.Duration().String())
	case slog.KindTime:
		return b.WithString(key, v.Time().Format("2006-01-02T15:04:05.999999999Z07:00"))
	default: // KindAny and anything future
		return b.WithAny(key, v.Any())
	}
}

// mapLevel buckets a slog level onto HaloLog's level scale.
func mapLevel(l slog.Level) types.LogLevel {
	switch {
	case l < slog.LevelInfo:
		return types.DebugLevel
	case l < slog.LevelWarn:
		return types.InfoLevel
	case l < slog.LevelError:
		return types.WarnLevel
	default:
		return types.ErrorLevel
	}
}

// emit dispatches the built line at the mapped level.
func emit(b core.TypedFieldBuilder, level types.LogLevel, msg string) {
	switch level {
	case types.DebugLevel:
		b.Debug(msg)
	case types.InfoLevel:
		b.Info(msg)
	case types.WarnLevel:
		b.Warn(msg)
	default:
		b.Error(msg)
	}
}

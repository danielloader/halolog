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
// Package config provides configuration management
// Author: Admilson B. F. Cossa

package file

import (
	"time"
)

// RotationConfig defines file rotation and performance settings
type RotationConfig struct {
	MaxSize          int64
	MaxAge           time.Duration
	MaxBackups       int
	Compress         bool
	LocalTime        bool
	FlushInterval    time.Duration
	MaxBatchSize     int
	BatchTimeout     time.Duration
	WriteTimeout     time.Duration
	QueueSize        int
	UseOSync         bool
	UseFlock         bool
	EnableMetrics    bool
	EnableBufferPool bool
	CircuitThreshold int
	CircuitTimeout   time.Duration
	RateLimit        int64
}

// DefaultRotationConfig returns production-ready defaults
func DefaultRotationConfig() *RotationConfig {
	return &RotationConfig{
		MaxSize:          100 * 1024 * 1024,  // 100MB
		MaxAge:           7 * 24 * time.Hour, // 7 days
		MaxBackups:       10,
		Compress:         true,
		LocalTime:        true,
		FlushInterval:    30 * time.Second,
		MaxBatchSize:     64 * 1024,
		BatchTimeout:     100 * time.Millisecond,
		WriteTimeout:     5 * time.Second,
		QueueSize:        65536,
		UseOSync:         false,
		UseFlock:         false,
		EnableMetrics:    true,
		EnableBufferPool: true,
		CircuitThreshold: 10,
		CircuitTimeout:   30 * time.Second,
		RateLimit:        0, // Disabled by default
	}
}

// FileConfig provides high-level configuration for the file adapter.
//
//nolint:revive // exported name intentionally kept for a stable public API; renaming to Config would break importers
type FileConfig struct {
	Path       string
	Rotation   *RotationConfig
	MaxSize    int64
	MaxBackups int
	MaxAge     time.Duration
	Compress   bool
}

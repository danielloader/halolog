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
// Package types provides core type definitions
// Author: Admilson B. F. Cossa

package types

import (
	"sync"
	"sync/atomic"
	"time"
)

/* =====================================================================
   HALOLOGGER STRUCT DEFINITION
   ===================================================================== */

// HaloLogger represents the main logging engine with High-performance paths and zero-allocation optimizations
type HaloLogger struct {
	// Core configuration
	level               LogLevel
	component           string
	environment         EnvironmentType
	workerCount         int
	bufferSize          int
	debugBufferSize     int
	healthCheckInterval time.Duration
	disableHealthCheck  bool

	// Adapters and outputs
	adapters   []Adapter
	outputs    []Adapter
	muAdapters sync.RWMutex

	// Performance and optimization
	fastPathEnabled    bool
	autoRegisterFields bool
	useGlobalDict      bool
	fieldDictionary    FieldDictionary // Use interface instead of concrete type
	staticFieldPool    *StaticFieldPool
	cachedClock        *CachedClock

	// Security and masking
	maskRules []MaskingRule
	piiMasker PIIMasker
	muMasks   sync.RWMutex

	// Sampling and filtering
	sampler             Sampler
	environmentDetector EnvironmentDetector

	// Metrics and statistics
	stats          *MetricsStats
	metricsEnabled atomic.Bool

	// State management
	started atomic.Bool
	clock   func() time.Time

	// Debug features
	debugBuffer *DebugBuffer

	// Worker pool and task management
	taskCh    chan *LogEntry
	stopCh    chan struct{}
	wgWorkers sync.WaitGroup

	// Health monitoring
	healthTicker      *time.Ticker
	healthStopCh      chan struct{}
	lastHealthCheck   atomic.Value // stores time.Time
	unhealthyAdapters map[string]int
	muUnhealthy       sync.RWMutex

	// Backpressure and flow control
	backpressureCallback func(dropped int64, bufferUsage float64)
	lastBackpressureWarn atomic.Value // stores time.Time

	// Field management
	pendingFields *FieldChain
	contextFields *FieldChain

	// Retry and error handling
	retryManager RetryManager

	// Additional configuration
	name               string
	samplingRate       atomic.Uint64
	rateLimit          atomic.Int64
	lastRateLimitReset atomic.Value // stores time.Time
}

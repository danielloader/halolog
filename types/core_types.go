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
// Package core provides core logging functionality
// Author: Admilson B. F. Cossa

package types

import (
	"io"
	"sync"
	"time"
)

// StaticFieldPool provides a pool for reusing field objects to reduce allocations
type StaticFieldPool struct {
	pool sync.Pool
}

// NewStaticFieldPool creates a new static field pool
func NewStaticFieldPool() *StaticFieldPool {
	return &StaticFieldPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &TypedFieldData{}
			},
		},
	}
}

// Get retrieves a field from the pool
func (p *StaticFieldPool) Get() *TypedFieldData {
	return p.pool.Get().(*TypedFieldData)
}

// Put returns a field to the pool
func (p *StaticFieldPool) Put(field *TypedFieldData) {
	// Reset field before returning to pool
	field.Key = ""
	field.Value = nil
	field.Type = TypedFieldString
	field.Optimized = false
	p.pool.Put(field)
}

// CachedClock provides a cached time source for performance
type CachedClock struct {
	currentTime    time.Time
	mu             sync.RWMutex
	updateInterval time.Duration
	lastUpdate     time.Time
}

// NewCachedClock creates a new cached clock
func NewCachedClock(updateInterval time.Duration) *CachedClock {
	now := time.Now()
	return &CachedClock{
		currentTime:    now,
		updateInterval: updateInterval,
		lastUpdate:     now,
	}
}

// Now returns the cached current time
func (c *CachedClock) Now() time.Time {
	c.mu.RLock()
	now := time.Now()
	if now.Sub(c.lastUpdate) > c.updateInterval {
		c.mu.RUnlock()
		c.mu.Lock()
		if now.Sub(c.lastUpdate) > c.updateInterval {
			c.currentTime = now
			c.lastUpdate = now
		}
		c.mu.Unlock()
		return c.currentTime
	}
	defer c.mu.RUnlock()
	return c.currentTime
}

// UnixTimestamp represents nanoseconds since Unix epoch
type UnixTimestamp int64

// ToTime converts UnixTimestamp to time.Time
func (ts UnixTimestamp) ToTime() time.Time {
	return time.Unix(0, int64(ts))
}

// CallerInfo contains caller information
type CallerInfo struct {
	File     string
	Line     int
	Function string
}

// RotationConfig configures file rotation
type RotationConfig struct {
	MaxSize    int64
	MaxBackups int
	MaxAge     time.Duration
}

// OutputTarget represents different output targets
type OutputTarget int

// Output target constants
const (
	ConsoleTarget OutputTarget = iota
	FileTarget
	NetworkTarget
)

// PrettyOptions configures pretty printing
type PrettyOptions struct {
	IndentSize     int
	SortFields     bool
	TimeFormat     string
	FieldAlignment bool
	Separator      string
	ShowTypes      bool
	ColorizeFields bool
	MaxLineLength  int
}

// WriterProvider provides io.Writer instances
type WriterProvider interface {
	GetWriter() (io.Writer, error)
	Close() error
}

// RotationType represents different rotation strategies
type RotationType int

// Rotation type constants
const (
	RotationSize RotationType = iota
	RotationTime
	RotationDaily
)

// MetricsStats contains logging metrics
type MetricsStats struct {
	TotalLogs      int64
	DebugCount     int64
	InfoCount      int64
	WarnCount      int64
	ErrorCount     int64
	FatalCount     int64
	ErrorsHandled  int64
	ErrorsTotal    int64 // Write failures
	DroppedCount   int64 // Rate limiting drops
	ComponentStats map[string]int64

	// Legacy fields (kept for compatibility if needed, or remove if unused)
	// ErrorLogs      int64
	// WarningLogs    int64
	// InfoLogs       int64
	// DebugLogs      int64
	// TraceLogs      int64
	// FatalLogs      int64
	// PanicLogs      int64
	// TotalBytes     int64
	// AverageLatency int64
}

// Stats contains comprehensive statistics
type Stats struct {
	Metrics    MetricsStats
	StartTime  time.Time
	LastUpdate time.Time
	Uptime     time.Duration
}

// ConsoleOutput writes to console (stdout/stderr)
type ConsoleOutput struct {
	Target io.Writer
	Stderr bool
	Color  bool
}

// FileOutput writes to file with rotation support
type FileOutput struct {
	Path   string
	Async  bool
	Buffer int

	// Rotation configuration
	Rotate     RotationType
	MaxSize    int64
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// RotationOption configures rotation parameters
type RotationOption func(*FileOutput)

// NetworkOutput writes to network endpoint
type NetworkOutput struct {
	Endpoint       string
	Protocol       string // "tcp", "udp", "http", "grpc"
	Timeout        time.Duration
	Retry          int
	FallbackOutput Output
}

// ConsoleAppenderOptions contains options for console appenders
type ConsoleAppenderOptions struct {
	Target OutputTarget
	Filter Filter
}

// FileAppenderOptions contains options for file appenders
type FileAppenderOptions struct {
	Path     string
	Filter   Filter
	Rotation RotationConfig
}

// NetworkAppenderOptions contains options for network appenders
type NetworkAppenderOptions struct {
	Protocol string
	Host     string
	Port     int
	Filter   Filter
}

// CompactFormatter formats logs compactly (one line)
type CompactFormatter struct {
	TimeFormat    string
	DisableColors bool
	ShowLevel     bool
	ShowTimestamp bool
	ShowComponent bool
}

// PatternLayoutOptions contains options for pattern layouts
type PatternLayoutOptions struct {
	Pattern string
}

// SimpleLayoutOptions contains options for simple layouts
type SimpleLayoutOptions struct {
	ShowTimestamp bool
	ShowLevel     bool
	ShowComponent bool
	ShowFile      bool
}

// MaskingRule defines a field masking rule
type MaskingRule struct {
	Pattern  string
	Replace  string
	Type     string      // "regex", "exact", "prefix", "suffix", "contains", "field"
	Compiled interface{} // Pre-compiled regex for performance (interface to avoid regexp import)
}

// AlertAction represents an action to take when an alert triggers
type AlertAction int

// Alert action constants
const (
	AlertEmail AlertAction = iota
	AlertSlack
	AlertWebhook
	AlertPagerDuty
)

// AlertConfig configures context-aware alerts
type AlertConfig struct {
	Pattern   string
	Context   []string
	Threshold int
	Cooldown  time.Duration
	Actions   []AlertAction
}

// SmartSamplingConfig for adaptive sampling
type SmartSamplingConfig struct {
	Strategy   string
	BaseRate   float64
	ErrorBoost float64
	Preserve   []string
}

// AggregationConfig for log aggregation
type AggregationConfig struct {
	Window    time.Duration
	GroupBy   []string
	Threshold int
}

// CaptureConfig for log replay
type CaptureConfig struct {
	Duration time.Duration
	Filter   string
	MaxLogs  int
}

// RoutingRule for log routing
type RoutingRule struct {
	Pattern string
	Level   LogLevel
	Output  string
}

// RoutingConfig for log routing
type RoutingConfig struct {
	Rules []RoutingRule
}

// DebugBuffer implements a circular buffer for debug logs
type DebugBuffer struct {
	// Internal fields - simplified for interface
	Entries []*LogEntry
	MaxSize int
	MaxAge  time.Duration
}

// Constants for configuration
const (
	// MaxFieldsPerLog is the maximum number of fields in the static buffer
	MaxFieldsPerLog    = 32
	MaxComponentLength = 64
	MaxMessageLength   = 1024

	// Ring buffer configuration (must be power of 2)
	RingBufferSize = 16384
	RingBufferMask = RingBufferSize - 1

	// Time cache update interval
	TimeCacheInterval = time.Millisecond

	// Metrics sampling rate (to avoid cache contention)
	MetricsSampleRate = 1024
)

// Size constants for readability
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

// 	Level       LogLevel
// 	Component   string
// 	Environment EnvironmentType

// 	// Performance configuration
// 	WorkerCount     int
// 	BufferSize      int
// 	DebugBufferSize int

// 	// Feature flags
// 	MetricsEnabled  bool
// 	FastPathEnabled bool

// 	// Internal state (simplified for interface)
// 	Started   bool
// 	Adapters  []Adapter
// 	Stats     *MetricsStats
// 	FieldDict interface{} // Field dictionary reference
// }

// EnvironmentType represents the deployment environment
type EnvironmentType string

// Environment type constants
const (
	EnvironmentDevelopment EnvironmentType = "development"
	EnvironmentTesting     EnvironmentType = "testing"
	EnvironmentStaging     EnvironmentType = "staging"
	EnvironmentProduction  EnvironmentType = "production"
)

// Write writes a log entry to console output (delegated to adapter)
func (o *ConsoleOutput) Write(entry LogEntry) error {
	return nil // Delegated to adapter
}

// Close closes the console output
func (o *ConsoleOutput) Close() error {
	return nil
}

// String returns a string representation of console output
func (o *ConsoleOutput) String() string {
	return "console"
}

// Write writes a log entry to file output (delegated to adapter)
func (o *FileOutput) Write(entry LogEntry) error {
	return nil // Delegated to adapter
}

// Close closes the file output
func (o *FileOutput) Close() error {
	return nil
}

// String returns a string representation of file output
func (o *FileOutput) String() string {
	return "file:" + o.Path
}

// Write writes a log entry to network output (delegated to adapter)
func (o *NetworkOutput) Write(entry LogEntry) error {
	return nil // Delegated to adapter
}

// Close closes the network output
func (o *NetworkOutput) Close() error {
	return nil
}

// String returns a string representation of network output
func (o *NetworkOutput) String() string {
	return "network:" + o.Endpoint
}

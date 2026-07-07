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
// Package fielddict provides field dictionary management
// Author: Admilson B. F. Cossa

package fielddict

import (
	"sync"
	"sync/atomic"

	"github.com/go-gen-ecosystem/halolog/types"
	"github.com/go-gen-ecosystem/halolog/utils"
)

// FieldDict provides predefined field keys for consistent logging
type FieldDict struct {
	TraceID   string
	SpanID    string
	RequestID string
	UserID    string
	Service   string
	Method    string
	Duration  string
	Error     string
}

// FieldDictionary provides optimized field key management with atomic operations
type FieldDictionary struct {
	mu       sync.RWMutex
	keys     map[string]int
	values   []string
	bitmasks map[string]uint64

	// Atomic fast path for field registration
	fieldCounter atomic.Int32  // Current field count for O(1) ID assignment
	fastPathHit  atomic.Uint64 // Counter for fast path hits (existing fields)
	slowPathHit  atomic.Uint64 // Counter for slow path hits (new fields)

	// Startup registration tracking
	startupRegistered atomic.Int32 // Fields registered at startup
	runtimeRegistered atomic.Int32 // Fields registered at runtime
}

// RegistrationStats tracks field registration performance
type RegistrationStats struct {
	StartupRegistered int // Fields registered at startup
	RuntimeRegistered int // Fields registered at runtime
	TotalFields       int // Total registered fields
	PrecomputedHits   int // O(1) lookups for pre-registered fields
	RuntimeMisses     int // Fields not pre-registered (slower path)
}

// StandardFieldDict contains standard field keys
var StandardFieldDict = &FieldDict{
	TraceID:   "trace_id",
	SpanID:    "span_id",
	RequestID: "request_id",
	UserID:    "user_id",
	Service:   "service",
	Method:    "method",
	Duration:  "duration",
	Error:     "error",
}

// NewFieldDictionary creates a new field dictionary
func NewFieldDictionary() *FieldDictionary {
	return &FieldDictionary{
		keys:     make(map[string]int),
		values:   make([]string, 0, 64),
		bitmasks: make(map[string]uint64),
	}
}

// Register registers a new field key and returns its ID
func (fd *FieldDictionary) Register(key string) int {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	if id, exists := fd.keys[key]; exists {
		return id
	}

	id := len(fd.values)
	fd.keys[key] = id
	fd.values = append(fd.values, key)

	// Create bitmask for fast field presence checking
	if id < 64 {
		fd.bitmasks[key] = 1 << uint(id)
	}

	return id
}

// Get retrieves field key by ID
func (fd *FieldDictionary) Get(id int) (string, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	if id < 0 || id >= len(fd.values) {
		return "", false
	}

	return fd.values[id], true
}

// GetID retrieves field ID by key
func (fd *FieldDictionary) GetID(key string) (int, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	id, exists := fd.keys[key]
	return id, exists
}

// GetOrRegisterFieldID gets existing field ID or registers new field - auto-registration
// Fast path: O(1) for existing fields, O(1) amortized for new fields
func (fd *FieldDictionary) GetOrRegisterFieldID(key string) int {
	// Fast path: Check if field exists (read lock only)
	fd.mu.RLock()
	if id, exists := fd.keys[key]; exists {
		fd.mu.RUnlock()
		fd.fastPathHit.Add(1)
		return id
	}
	fd.mu.RUnlock()

	// Slow path: Register new field (write lock required)
	return fd.registerNewField(key)
}

// registerNewField registers a new field with write lock
func (fd *FieldDictionary) registerNewField(key string) int {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	// Double-check after acquiring write lock
	if id, exists := fd.keys[key]; exists {
		fd.fastPathHit.Add(1)
		return id
	}

	// Register new field
	id := len(fd.values)
	fd.keys[key] = id
	fd.values = append(fd.values, key)

	// Create bitmask for fast field presence checking
	if id < 64 {
		fd.bitmasks[key] = 1 << uint(id)
	}

	fd.fieldCounter.Add(1)
	fd.slowPathHit.Add(1)
	fd.runtimeRegistered.Add(1) // Track runtime registration
	return id
}

// GetFieldID checks if a field exists and returns its ID (O(1) direct access)
// This is used for High-performance existing field detection in WithField operations
func (fd *FieldDictionary) GetFieldID(key string) (int, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	id, exists := fd.keys[key]
	if exists {
		fd.fastPathHit.Add(1)
	}
	return id, exists
}

// GetOrRegisterFieldIDFast provides High-performance field registration with atomic operations
// Returns: fieldID, isNewField, totalAllocations
func (fd *FieldDictionary) GetOrRegisterFieldIDFast(key string) (int, bool, int) {
	// High-performance path: Check existing field with read lock
	fd.mu.RLock()
	if id, exists := fd.keys[key]; exists {
		fd.mu.RUnlock()
		fd.fastPathHit.Add(1)
		return id, false, 0 // Existing field: 0 allocations
	}
	fd.mu.RUnlock()

	// New field registration
	id := fd.registerNewField(key)
	return id, true, 1 // New field: 1 allocation for dictionary entry
}

// CreateFieldBitmask creates a bitmask for multiple field keys
func (fd *FieldDictionary) CreateFieldBitmask(keys ...string) uint64 {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	var bitmask uint64
	for _, key := range keys {
		if fieldBitmask, exists := fd.bitmasks[key]; exists {
			bitmask |= fieldBitmask
		}
	}
	return bitmask
}

// IsFieldInBitmask checks if a field ID is present in a bitmask
func (fd *FieldDictionary) IsFieldInBitmask(fieldID int, bitmask uint64) bool {
	if fieldID >= 64 {
		return false // Beyond bitmask range
	}
	return (bitmask & (1 << uint(fieldID))) != 0
}

// GetBitmask returns the bitmask for a field key
func (fd *FieldDictionary) GetBitmask(key string) (uint64, bool) {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	bitmask, exists := fd.bitmasks[key]
	return bitmask, exists
}

// Size returns the number of registered fields
func (fd *FieldDictionary) Size() int {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	return len(fd.values)
}

// Count returns the total number of registered fields (alias for Size to match interface)
func (fd *FieldDictionary) Count() int {
	return fd.Size()
}

// GetAllFields returns all registered field keys
func (fd *FieldDictionary) GetAllFields() []string {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	result := make([]string, len(fd.values))
	copy(result, fd.values)
	return result
}

// GetAllFieldIDs returns a map of field keys to their IDs
func (fd *FieldDictionary) GetAllFieldIDs() map[string]int {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	result := make(map[string]int, len(fd.keys))
	for key, id := range fd.keys {
		result[key] = id
	}
	return result
}

// GetPerformanceStats returns field dictionary performance statistics
func (fd *FieldDictionary) GetPerformanceStats() types.FieldDictionaryStats {
	return types.FieldDictionaryStats{
		TotalFields:  int(fd.fieldCounter.Load()),
		FastPathHits: fd.fastPathHit.Load(),
		SlowPathHits: fd.slowPathHit.Load(),
	}
}

// GetStats returns performance statistics (alias for GetPerformanceStats to match interface)
func (fd *FieldDictionary) GetStats() types.FieldDictionaryStats {
	return fd.GetPerformanceStats()
}

// RegisterBulk registers multiple field keys atomically
func (fd *FieldDictionary) RegisterBulk(keys []string) []int {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	ids := make([]int, len(keys))
	for i, key := range keys {
		if id, exists := fd.keys[key]; exists {
			ids[i] = id
			continue
		}

		id := len(fd.values)
		fd.keys[key] = id
		fd.values = append(fd.values, key)

		// Create bitmask for fast field presence checking
		if id < 64 {
			fd.bitmasks[key] = 1 << uint(id)
		}

		ids[i] = id
	}
	return ids
}

// GetFieldByID returns the field key for a given field ID
func (fd *FieldDictionary) GetFieldByID(fieldID int) string {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	// Check bounds
	if fieldID < 0 || fieldID >= len(fd.values) {
		return ""
	}
	return fd.values[fieldID]
}

// RegisterFromConfig registers all fields from startup configuration
// This enables O(1) field access for pre-registered fields
func (fd *FieldDictionary) RegisterFromConfig(customFields []string, sensitiveFields []string) error {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	// Register custom fields
	for _, fieldName := range customFields {
		if _, exists := fd.keys[fieldName]; !exists {
			id := len(fd.values)
			fd.keys[fieldName] = id
			fd.values = append(fd.values, fieldName)

			// Create bitmask for fast field presence checking
			if id < 64 {
				fd.bitmasks[fieldName] = 1 << uint(id)
			}

			fd.startupRegistered.Add(1)
		}
	}

	// Register sensitive fields
	for _, fieldName := range sensitiveFields {
		if _, exists := fd.keys[fieldName]; !exists {
			id := len(fd.values)
			fd.keys[fieldName] = id
			fd.values = append(fd.values, fieldName)

			// Create bitmask for fast field presence checking
			if id < 64 {
				fd.bitmasks[fieldName] = 1 << uint(id)
			}

			fd.startupRegistered.Add(1)
		}
	}

	return nil
}

// RegisterCustomFieldsWithIDs registers custom fields with pre-assigned IDs
// This is used when field IDs are pre-computed in configuration
func (fd *FieldDictionary) RegisterCustomFieldsWithIDs(fields map[string]int) error {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	for fieldName, fieldID := range fields {
		// Ensure values slice is large enough
		for len(fd.values) <= fieldID {
			fd.values = append(fd.values, "")
		}

		// Register field with specific ID
		fd.keys[fieldName] = fieldID
		fd.values[fieldID] = fieldName

		// Create bitmask for fast field presence checking
		if fieldID < 64 {
			fd.bitmasks[fieldName] = 1 << uint(fieldID)
		}

		fd.startupRegistered.Add(1)
	}

	return nil
}

// GetRegistrationStats returns stats on startup vs runtime registrations
func (fd *FieldDictionary) GetRegistrationStats() RegistrationStats {
	fd.mu.RLock()
	defer fd.mu.RUnlock()

	return RegistrationStats{
		StartupRegistered: int(fd.startupRegistered.Load()),
		RuntimeRegistered: int(fd.runtimeRegistered.Load()),
		TotalFields:       len(fd.values),
		PrecomputedHits:   int(fd.fastPathHit.Load()),
		RuntimeMisses:     int(fd.slowPathHit.Load()),
	}
}

// GlobalFieldDictionary provides a singleton instance for system-wide use
var GlobalFieldDictionary = NewFieldDictionary()

// Field represents a field with dictionary optimization
type Field struct {
	KeyID int
	Value interface{}
}

// NewField creates a new field using the global dictionary with auto-registration
func NewField(key string, value interface{}) Field {
	keyID := GlobalFieldDictionary.GetOrRegisterFieldID(key)
	return Field{KeyID: keyID, Value: value}
}

// NewFieldWithDict creates a new field using a specific dictionary
func NewFieldWithDict(dict *FieldDictionary, key string, value interface{}) Field {
	if dict == nil {
		dict = GlobalFieldDictionary
	}
	keyID := dict.GetOrRegisterFieldID(key)
	return Field{KeyID: keyID, Value: value}
}

// String returns string representation of the field
func (f Field) String() string {
	key, exists := GlobalFieldDictionary.Get(f.KeyID)
	if !exists {
		field_Key := utils.FormatIntWithPrefix("field_", f.KeyID)
		return utils.FormatKeyValueAnySep(field_Key, "=", f.Value)
	}
	return utils.FormatKeyValueAnySep(key, "=", f.Value)
}

// StringWithDict returns string representation using a specific dictionary
func (f Field) StringWithDict(dict *FieldDictionary) string {
	if dict == nil {
		dict = GlobalFieldDictionary
	}
	key, exists := dict.Get(f.KeyID)
	if !exists {
		return utils.FormatKeyValueAnySep(key, "=", f.Value)
	}
	return utils.FormatKeyValueAnySep(key, "=", f.Value)
}

// GetStandardFields returns a map of standard logging fields with their descriptions
func GetStandardFields() map[string]string {
	return map[string]string{
		FieldTimestamp:        "Unix timestamp when the log was created",
		FieldLevel:            "Log level (DEBUG, INFO, WARN, ERROR, FATAL)",
		FieldLogger:           "Name of the logger instance",
		FieldComponent:        "Component or module name",
		FieldMessage:          "Log message content",
		FieldError:            "Error message or stack trace",
		FieldFile:             "Source file where log was generated",
		FieldLine:             "Line number in source file",
		FieldFunction:         "Function name where log was generated",
		FieldGoroutineID:      "Go routine ID for debugging",
		FieldMemoryAlloc:      "Current memory allocation in bytes",
		FieldMemoryTotalAlloc: "Total memory allocated in bytes",
		FieldMemorySys:        "Memory obtained from system in bytes",
		FieldNumCPU:           "Number of CPU cores",
	}
}

// GetRequestFields returns a map of HTTP request/response fields with their descriptions
func GetRequestFields() map[string]string {
	return map[string]string{
		FieldRequestID: "Unique request identifier",
		FieldUserID:    "User identifier",
		FieldSessionID: "Session identifier",
		FieldIP:        "Client IP address",
		FieldUserAgent: "Client user agent string",
		FieldMethod:    "HTTP request method",
		FieldURL:       "Request URL",
		FieldStatus:    "HTTP response status code",
		FieldDuration:  "Request duration in milliseconds",
		FieldBytesIn:   "Bytes received",
		FieldBytesOut:  "Bytes sent",
	}
}

// GetTracingFields returns a map of distributed tracing fields with their descriptions
func GetTracingFields() map[string]string {
	return map[string]string{
		FieldTraceID: "Distributed trace identifier",
		FieldSpanID:  "Span identifier within trace",
	}
}

// GetAllFields returns a combined map of all field categories
func GetAllFields() map[string]string {
	allFields := make(map[string]string)

	// Add other field categories first (lower priority)
	for k, v := range getDatabaseFields() {
		allFields[k] = v
	}

	for k, v := range getSystemFields() {
		allFields[k] = v
	}

	for k, v := range getPerformanceFields() {
		allFields[k] = v
	}

	for k, v := range getMessageQueueFields() {
		allFields[k] = v
	}

	for k, v := range getCacheFields() {
		allFields[k] = v
	}

	for k, v := range getSecurityFields() {
		allFields[k] = v
	}

	for k, v := range getBusinessFields() {
		allFields[k] = v
	}

	for k, v := range getCloudFields() {
		allFields[k] = v
	}

	// Add tracing fields (higher priority)
	for k, v := range GetTracingFields() {
		allFields[k] = v
	}

	// Add request fields (even higher priority)
	for k, v := range GetRequestFields() {
		allFields[k] = v
	}

	// Add standard fields (highest priority)
	for k, v := range GetStandardFields() {
		allFields[k] = v
	}

	return allFields
}

// CreateStandardFieldDictionary creates a field dictionary pre-populated with standard fields
func CreateStandardFieldDictionary() *FieldDictionary {
	fd := NewFieldDictionary()

	// Register all standard fields
	standardFields := GetStandardFields()
	for fieldName := range standardFields {
		fd.Register(fieldName)
	}

	requestFields := GetRequestFields()
	for fieldName := range requestFields {
		fd.Register(fieldName)
	}

	tracingFields := GetTracingFields()
	for fieldName := range tracingFields {
		fd.Register(fieldName)
	}

	databaseFields := getDatabaseFields()
	for fieldName := range databaseFields {
		fd.Register(fieldName)
	}

	systemFields := getSystemFields()
	for fieldName := range systemFields {
		fd.Register(fieldName)
	}

	performanceFields := getPerformanceFields()
	for fieldName := range performanceFields {
		fd.Register(fieldName)
	}

	messageQueueFields := getMessageQueueFields()
	for fieldName := range messageQueueFields {
		fd.Register(fieldName)
	}

	cacheFields := getCacheFields()
	for fieldName := range cacheFields {
		fd.Register(fieldName)
	}

	securityFields := getSecurityFields()
	for fieldName := range securityFields {
		fd.Register(fieldName)
	}

	businessFields := getBusinessFields()
	for fieldName := range businessFields {
		fd.Register(fieldName)
	}

	cloudFields := getCloudFields()
	for fieldName := range cloudFields {
		fd.Register(fieldName)
	}

	return fd
}

// Helper functions for additional field categories
func getDatabaseFields() map[string]string {
	return map[string]string{
		FieldQuery:        "Database query string",
		FieldQueryTime:    "Database query execution time",
		FieldRowsAffected: "Number of rows affected by query",
		FieldDBHost:       "Database host",
		FieldDBPort:       "Database port",
		FieldDBName:       "Database name",
		FieldTable:        "Database table name",
		FieldOperation:    "Database operation type",
	}
}

func getSystemFields() map[string]string {
	return map[string]string{
		FieldHostname:     "System hostname",
		FieldPID:          "Process ID",
		FieldNumGoroutine: "Number of goroutines",
		FieldNumCPU:       "Number of CPU cores",
		FieldUptime:       "System uptime",
	}
}

func getPerformanceFields() map[string]string {
	return map[string]string{
		FieldDuration:         "Operation duration in milliseconds",
		FieldMemoryAlloc:      "Current memory allocation in bytes",
		FieldMemoryTotalAlloc: "Total memory allocated in bytes",
		FieldMemorySys:        "Memory obtained from system in bytes",
		FieldNumGC:            "Number of garbage collection cycles",
		FieldGCTime:           "Time spent in garbage collection",
	}
}

func getMessageQueueFields() map[string]string {
	return map[string]string{
		FieldQueueName:     "Message queue name",
		FieldMessageID:     "Message identifier",
		FieldCorrelationID: "Correlation identifier for message tracing",
		FieldRetryCount:    "Number of retry attempts",
		FieldDeadLetter:    "Dead letter queue indicator",
		FieldBroker:        "Message broker identifier",
	}
}

func getCacheFields() map[string]string {
	return map[string]string{
		FieldKey:       "Cache key",
		FieldValue:     "Cache value",
		FieldTTL:       "Time to live for cache entry",
		FieldHit:       "Cache hit indicator",
		FieldCacheName: "Cache name",
	}
}

func getSecurityFields() map[string]string {
	return map[string]string{
		FieldIP:        "IP address for security logging",
		FieldUserAgent: "User agent for security analysis",
		FieldStatus:    "Security status or result",
		FieldError:     "Security error or violation",
		FieldRequestID: "Request identifier for security tracking",
		FieldSessionID: "Session identifier for security context",
	}
}

func getBusinessFields() map[string]string {
	return map[string]string{
		FieldUserID:    "User identifier for business metrics",
		FieldSessionID: "Session identifier for business tracking",
		FieldRequestID: "Request identifier for business correlation",
		FieldStatus:    "Business operation status",
		FieldError:     "Business logic error",
		FieldDuration:  "Business operation duration",
		FieldComponent: "Business component or service",
	}
}

func getCloudFields() map[string]string {
	return map[string]string{
		FieldInstanceID:   "Cloud instance identifier",
		FieldInstanceType: "Cloud instance type",
		FieldRegion:       "Cloud region",
		FieldZone:         "Cloud availability zone",
		FieldProject:      "Cloud project identifier",
	}
}

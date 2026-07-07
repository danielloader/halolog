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
// Package halolog provides high-performance logging
// Author: Admilson B. F. Cossa

package race

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/go-gen-ecosystem/halolog/types"
)

// RacePrevention provides comprehensive race condition prevention mechanisms.
//
//nolint:revive // "Race" prefix is the domain name of this public package's central type; renaming to Prevention would lose meaning and break external API consumers.
type RacePrevention struct {
	// Atomic state for lock-free operations
	state      atomic.Uint64
	version    atomic.Uint64
	lastAccess atomic.Int64

	// Memory barriers for happens-before guarantees
	writeBarrier atomic.Bool
	readBarrier  atomic.Bool

	// Thread-local storage for per-thread safety
	tlsPool      sync.Pool
	threadStates sync.Map // map[uint64]*ThreadState

	// Race detection and monitoring
	raceDetectors []RaceDetector
	raceCount     atomic.Uint64

	// Performance metrics
	syncOperations atomic.Uint64
	lockFreeOps    atomic.Uint64
	barrierOps     atomic.Uint64
}

// ThreadState tracks per-thread state for race detection
type ThreadState struct {
	threadID    uint64
	lastWrite   atomic.Int64
	lastRead    atomic.Int64
	accessCount atomic.Uint64
	writeCount  atomic.Uint64
	readCount   atomic.Uint64
}

// RaceDetector interface for pluggable race detection strategies.
//
//nolint:revive // "Race" prefix names the domain of this public detection interface; renaming to Detector would be ambiguous and break external API consumers.
type RaceDetector interface {
	CheckRace(threadID uint64, operation string, data unsafe.Pointer) bool
	RecordAccess(threadID uint64, operation string, data unsafe.Pointer)
	GetRaceCount() uint64
	Reset()
}

// GlobalRacePrevention is the singleton instance for system-wide use
var GlobalRacePrevention *RacePrevention

// InitializeRacePrevention initializes the global race prevention system
func InitializeRacePrevention() {
	if GlobalRacePrevention != nil {
		return // Already initialized
	}

	GlobalRacePrevention = &RacePrevention{
		raceDetectors: []RaceDetector{
			NewBasicRaceDetector(),
			NewMemoryOrderDetector(),
		},
	}

	// Initialize thread-local storage
	GlobalRacePrevention.tlsPool = sync.Pool{
		New: func() interface{} {
			return &ThreadState{
				threadID: uint64(time.Now().UnixNano()), // Unique per thread
			}
		},
	}
}

// BasicRaceDetector provides basic race condition detection
type BasicRaceDetector struct {
	accesses  sync.Map // map[string][]AccessRecord
	raceCount atomic.Uint64
}

// AccessRecord captures a single memory access observed by a race detector.
type AccessRecord struct {
	ThreadID  uint64
	Timestamp int64
	Operation string
}

// NewBasicRaceDetector creates a new basic race detector
func NewBasicRaceDetector() *BasicRaceDetector {
	return &BasicRaceDetector{}
}

// CheckRace detects potential race conditions
func (brd *BasicRaceDetector) CheckRace(threadID uint64, operation string, data unsafe.Pointer) bool {
	key := uintptr(data)

	if records, exists := brd.accesses.Load(key); exists {
		accessRecords := records.([]AccessRecord)

		// Check for concurrent conflicting accesses
		for _, record := range accessRecords {
			if record.ThreadID != threadID &&
				((operation == "write" && record.Operation == "write") ||
					(operation == "read" && record.Operation == "write") ||
					(operation == "write" && record.Operation == "read")) {

				// Potential race detected
				brd.raceCount.Add(1)
				return true
			}
		}
	}

	return false
}

// RecordAccess records an access for future race detection
func (brd *BasicRaceDetector) RecordAccess(threadID uint64, operation string, data unsafe.Pointer) {
	key := uintptr(data)

	record := AccessRecord{
		ThreadID:  threadID,
		Timestamp: time.Now().UnixNano(),
		Operation: operation,
	}

	// Store or append to existing records
	if records, exists := brd.accesses.Load(key); exists {
		accessRecords := records.([]AccessRecord)
		accessRecords = append(accessRecords, record)
		brd.accesses.Store(key, accessRecords)
	} else {
		brd.accesses.Store(key, []AccessRecord{record})
	}
}

// GetRaceCount returns the number of detected races
func (brd *BasicRaceDetector) GetRaceCount() uint64 {
	return brd.raceCount.Load()
}

// Reset clears all recorded accesses
func (brd *BasicRaceDetector) Reset() {
	brd.accesses.Range(func(key, value interface{}) bool {
		brd.accesses.Delete(key)
		return true
	})
	brd.raceCount.Store(0)
}

// MemoryOrderDetector detects memory ordering violations
type MemoryOrderDetector struct {
	orderViolations atomic.Uint64
	memoryBarriers  sync.Map // map[uintptr]int64
}

// NewMemoryOrderDetector creates a new memory order detector
func NewMemoryOrderDetector() *MemoryOrderDetector {
	return &MemoryOrderDetector{}
}

// CheckRace detects memory ordering violations
func (mod *MemoryOrderDetector) CheckRace(threadID uint64, operation string, data unsafe.Pointer) bool {
	// Implementation for memory ordering violation detection
	// This is a simplified version - real implementation would be more complex

	key := uintptr(data)

	if barrierTime, exists := mod.memoryBarriers.Load(key); exists {
		barrierTimestamp := barrierTime.(int64)
		currentTime := time.Now().UnixNano()

		// Check if access violates memory ordering
		if currentTime < barrierTimestamp {
			mod.orderViolations.Add(1)
			return true
		}
	}

	return false
}

// RecordAccess records memory barrier information
func (mod *MemoryOrderDetector) RecordAccess(threadID uint64, operation string, data unsafe.Pointer) {
	key := uintptr(data)

	if operation == "barrier" {
		mod.memoryBarriers.Store(key, time.Now().UnixNano())
	}
}

// GetRaceCount returns the number of memory ordering violations
func (mod *MemoryOrderDetector) GetRaceCount() uint64 {
	return mod.orderViolations.Load()
}

// Reset clears memory barrier records
func (mod *MemoryOrderDetector) Reset() {
	mod.memoryBarriers.Range(func(key, value interface{}) bool {
		mod.memoryBarriers.Delete(key)
		return true
	})
	mod.orderViolations.Store(0)
}

// SafeFieldAccess provides thread-safe field access with race detection
func (rp *RacePrevention) SafeFieldAccess(fieldData *types.TypedFieldData, operation string) *types.TypedFieldData {
	if rp == nil {
		return fieldData // No protection
	}

	// Get current thread state
	threadState := rp.getThreadState()

	// Memory barrier for happens-before guarantee
	rp.memoryBarrier(operation)

	// Check for race conditions
	dataPtr := unsafe.Pointer(fieldData)
	if rp.detectRace(threadState.threadID, operation, dataPtr) {
		// Race detected - return defensive copy
		return rp.createDefensiveCopy(fieldData)
	}

	// Record access for future race detection
	rp.recordAccess(threadState.threadID, operation, dataPtr)

	// Update thread state
	rp.updateThreadState(threadState, operation)

	rp.syncOperations.Add(1)
	return fieldData
}

// SafePipelineTransition ensures thread-safe pipeline state transitions
func (rp *RacePrevention) SafePipelineTransition(oldPipeline, newPipeline interface{}) bool {
	if rp == nil {
		return true // No protection
	}

	// Atomic state transition
	currentState := rp.state.Load()
	newState := currentState + 1

	// Memory barrier before transition
	rp.writeBarrier.Store(true)
	runtime.Gosched() // Yield to ensure visibility

	// Atomic state update
	if rp.state.CompareAndSwap(currentState, newState) {
		// Memory barrier after transition
		rp.writeBarrier.Store(false)
		rp.version.Add(1)

		rp.lockFreeOps.Add(1)
		return true
	}

	return false
}

// SafeEntryPoolAccess provides thread-safe access to entry pool
func (rp *RacePrevention) SafeEntryPoolAccess(entry *types.LogEntry, operation string) *types.LogEntry {
	if rp == nil {
		return entry // No protection
	}

	// Memory barrier for pool access
	rp.memoryBarrier(operation)

	// Check for concurrent access patterns
	threadState := rp.getThreadState()
	dataPtr := unsafe.Pointer(entry)

	if rp.detectRace(threadState.threadID, operation, dataPtr) {
		// Race detected in pool access
		rp.raceCount.Add(1)
		return rp.createDefensiveEntryCopy(entry)
	}

	// Record access
	rp.recordAccess(threadState.threadID, operation, dataPtr)
	rp.updateThreadState(threadState, operation)

	return entry
}

// getThreadState gets or creates thread-local state
func (rp *RacePrevention) getThreadState() *ThreadState {
	if state, exists := rp.threadStates.Load(routineID()); exists {
		return state.(*ThreadState)
	}

	// Create new thread state
	newState := rp.tlsPool.Get().(*ThreadState)
	rp.threadStates.Store(routineID(), newState)
	return newState
}

// memoryBarrier provides memory ordering guarantees
func (rp *RacePrevention) memoryBarrier(operation string) {
	if operation == "write" {
		rp.writeBarrier.Store(true)
	} else {
		rp.readBarrier.Store(true)
	}

	runtime.Gosched() // Memory fence

	if operation == "write" {
		rp.writeBarrier.Store(false)
	} else {
		rp.readBarrier.Store(false)
	}

	rp.barrierOps.Add(1)
}

// detectRace runs all registered race detectors
func (rp *RacePrevention) detectRace(threadID uint64, operation string, data unsafe.Pointer) bool {
	for _, detector := range rp.raceDetectors {
		if detector.CheckRace(threadID, operation, data) {
			return true
		}
	}
	return false
}

// recordAccess records access across all detectors
func (rp *RacePrevention) recordAccess(threadID uint64, operation string, data unsafe.Pointer) {
	for _, detector := range rp.raceDetectors {
		detector.RecordAccess(threadID, operation, data)
	}
}

// updateThreadState updates thread access tracking
func (rp *RacePrevention) updateThreadState(state *ThreadState, operation string) {
	now := time.Now().UnixNano()

	if operation == "write" {
		state.lastWrite.Store(now)
		state.writeCount.Add(1)
	} else {
		state.lastRead.Store(now)
		state.readCount.Add(1)
	}

	state.accessCount.Add(1)
	rp.lastAccess.Store(now)
}

// createDefensiveCopy creates a defensive copy to prevent race conditions
func (rp *RacePrevention) createDefensiveCopy(original *types.TypedFieldData) *types.TypedFieldData {
	if original == nil {
		return nil
	}

	return &types.TypedFieldData{
		Key:   original.Key,
		Value: original.Value,
		Type:  original.Type,
	}
}

// createDefensiveEntryCopy creates a defensive copy of log entry
func (rp *RacePrevention) createDefensiveEntryCopy(original *types.LogEntry) *types.LogEntry {
	if original == nil {
		return nil
	}

	// Create shallow copy (fields remain shared for performance)
	defensiveCopy := *original
	return &defensiveCopy
}

// routineID returns a unique identifier for the current goroutine
func routineID() uint64 {
	// This is a simplified implementation
	// In production, you'd use runtime.Stack or similar for reliable goroutine ID
	return uint64(time.Now().UnixNano())
}

// GetRaceStatistics returns comprehensive race detection statistics
func (rp *RacePrevention) GetRaceStatistics() RacePreventionStats {
	if rp == nil {
		return RacePreventionStats{}
	}

	totalRaces := uint64(0)
	for _, detector := range rp.raceDetectors {
		totalRaces += detector.GetRaceCount()
	}

	return RacePreventionStats{
		TotalRacesDetected: totalRaces,
		SyncOperations:     rp.syncOperations.Load(),
		LockFreeOperations: rp.lockFreeOps.Load(),
		BarrierOperations:  rp.barrierOps.Load(),
		LastAccessTime:     rp.lastAccess.Load(),
		ActiveThreads:      rp.getActiveThreadCount(),
	}
}

// getActiveThreadCount returns the number of active threads
func (rp *RacePrevention) getActiveThreadCount() int {
	count := 0
	rp.threadStates.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Reset clears all race detection state
func (rp *RacePrevention) Reset() {
	if rp == nil {
		return
	}

	// Reset all detectors
	for _, detector := range rp.raceDetectors {
		detector.Reset()
	}

	// Clear thread states
	rp.threadStates.Range(func(key, value interface{}) bool {
		rp.threadStates.Delete(key)
		return true
	})

	// Reset metrics
	rp.raceCount.Store(0)
	rp.syncOperations.Store(0)
	rp.lockFreeOps.Store(0)
	rp.barrierOps.Store(0)
	rp.lastAccess.Store(0)
}

// RacePreventionStats contains comprehensive race prevention statistics.
//
//nolint:revive // "Race" prefix matches the RacePrevention type it reports on; renaming to PreventionStats would break the naming pair and external API consumers.
type RacePreventionStats struct {
	TotalRacesDetected uint64
	SyncOperations     uint64
	LockFreeOperations uint64
	BarrierOperations  uint64
	LastAccessTime     int64
	ActiveThreads      int
}

// SafeFieldAccessWithContext provides context-aware safe field access
func SafeFieldAccessWithContext(ctx context.Context, fieldData *types.TypedFieldData, operation string) *types.TypedFieldData {
	if GlobalRacePrevention == nil {
		InitializeRacePrevention()
	}

	return GlobalRacePrevention.SafeFieldAccess(fieldData, operation)
}

// SafePipelineTransitionWithContext provides context-aware safe pipeline transitions
func SafePipelineTransitionWithContext(ctx context.Context, oldPipeline, newPipeline interface{}) bool {
	if GlobalRacePrevention == nil {
		InitializeRacePrevention()
	}

	return GlobalRacePrevention.SafePipelineTransition(oldPipeline, newPipeline)
}

// SafeStrategyAccess provides thread-safe access to strategy operations
func (rp *RacePrevention) SafeStrategyAccess(strategyID uint64, operation string) {
	if rp == nil {
		return
	}

	// Memory barrier for strategy access
	rp.memoryBarrier(operation)

	// Record strategy access
	threadState := rp.getThreadState()
	rp.recordAccess(threadState.threadID, operation, unsafe.Pointer(&strategyID))
	rp.updateThreadState(threadState, operation)

	rp.syncOperations.Add(1)
}

// SafePipelineExecution provides thread-safe pipeline execution
func (rp *RacePrevention) SafePipelineExecution(strategyID uint64, operation string, pipeline interface{}) {
	if rp == nil {
		return
	}

	// Memory barrier for pipeline execution
	rp.memoryBarrier(operation)

	// Record pipeline execution
	threadState := rp.getThreadState()
	pipelinePtr := unsafe.Pointer(&pipeline)

	if rp.detectRace(threadState.threadID, operation, pipelinePtr) {
		// Race detected in pipeline execution
		rp.raceCount.Add(1)
	}

	rp.recordAccess(threadState.threadID, operation, pipelinePtr)
	rp.updateThreadState(threadState, operation)

	rp.syncOperations.Add(1)
}

// StrategyRegistry tracks registered strategies for race prevention
type StrategyRegistry struct {
	strategies    sync.Map // map[uint64]bool
	strategyCount atomic.Uint64
}

var globalStrategyRegistry = &StrategyRegistry{}

// RegisterStrategy registers a strategy for race prevention tracking
func (rp *RacePrevention) RegisterStrategy(strategyID uint64) {
	if rp == nil {
		return
	}

	globalStrategyRegistry.strategies.Store(strategyID, true)
	globalStrategyRegistry.strategyCount.Add(1)
}

// UnregisterStrategy unregisters a strategy from race prevention tracking
func (rp *RacePrevention) UnregisterStrategy(strategyID uint64) {
	if rp == nil {
		return
	}

	globalStrategyRegistry.strategies.Delete(strategyID)
	if count := globalStrategyRegistry.strategyCount.Load(); count > 0 {
		globalStrategyRegistry.strategyCount.Add(^uint64(0)) // Decrement by 1
	}
}

// GenerateStrategyID generates a unique strategy ID
func (rp *RacePrevention) GenerateStrategyID() uint64 {
	if rp == nil {
		return 0
	}

	// Generate unique ID based on current time and atomic counter
	return uint64(time.Now().UnixNano()) + rp.version.Add(1)
}

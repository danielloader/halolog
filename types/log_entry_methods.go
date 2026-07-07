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
	"strconv"
	"strings"
	"time"
)

/* =====================================================================
   LOG ENTRY CONVERSION (SIMPLIFIED - UNIFIED types.LogEntry)
   ===================================================================== */

// ToLogEntry returns the entry itself for compatibility
// Since we've merged types.LogEntry into unified types.LogEntry, no conversion needed
func (e *LogEntry) ToLogEntry() *LogEntry {
	return e
}

/* =====================================================================
   LOG ENTRY METHODS
   ===================================================================== */

// AddField adds a field to the log entry
func (e *LogEntry) AddField(key string, value interface{}) {
	if e.Fields == nil {
		e.Fields = make([]TypedFieldData, 0, 8)
	}
	e.Fields = append(e.Fields, TypedFieldData{
		Key:   key,
		Value: value,
		Type:  inferTypeFromValue(value),
	})
}

// AddFields adds multiple fields to the log entry
func (e *LogEntry) AddFields(fields map[string]interface{}) {
	if e.Fields == nil {
		e.Fields = make([]TypedFieldData, 0, len(fields))
	}
	for k, v := range fields {
		e.Fields = append(e.Fields, TypedFieldData{
			Key:   k,
			Value: v,
			Type:  inferTypeFromValue(v),
		})
	}
}

// AddContext adds context to the log entry
func (e *LogEntry) AddContext(key string, value interface{}) {
	if e.Context == nil {
		e.Context = make([]TypedFieldData, 0, 8)
	}
	e.Context = append(e.Context, TypedFieldData{
		Key:   key,
		Value: value,
		Type:  inferTypeFromValue(value),
	})
}

// SetError sets the error for the log entry
func (e *LogEntry) SetError(err error) {
	e.Error = err
}

// SetComponent sets the component for the log entry
func (e *LogEntry) SetComponent(component string) {
	e.Component = component
}

/* =====================================================================
   QUANTUM STORAGE INTEGRATION METHODS
   ===================================================================== */

// EnableQuantumStorage enables quantum field storage for unlimited fields
func (e *LogEntry) EnableQuantumStorage(chunkSize int) {
	if e.QuantumStore == nil {
		e.QuantumStore = NewEnhancedQuantumFieldStore(chunkSize)
		e.UseQuantumStorage = true
	}
}

// AddQuantumField adds a field to the quantum store (coexists with typed fields)
func (e *LogEntry) AddQuantumField(key string, value interface{}) {
	if e.QuantumStore == nil {
		e.EnableQuantumStorage(64) // Default chunk size
	}

	// Convert value to string for quantum storage
	strValue := formatValue(value)
	e.QuantumStore.Set(key, strValue)
}

// GetQuantumField retrieves a field from the quantum store
func (e *LogEntry) GetQuantumField(key string) (string, bool) {
	if e.QuantumStore == nil {
		return "", false
	}
	return e.QuantumStore.Get(key)
}

// GetAllFields returns all fields (both typed and quantum) as TypedField slices
func (e *LogEntry) GetAllFields() []TypedFieldData {
	// Calculate total field count
	totalFields := len(e.Fields) + e.StaticFieldCount
	if e.QuantumStore != nil {
		totalFields += e.QuantumStore.Size()
	}

	// Pre-allocate with exact capacity
	allFields := make([]TypedFieldData, 0, totalFields)

	// Add static fields first (FastPath optimization)
	for i := 0; i < e.StaticFieldCount; i++ {
		allFields = append(allFields, e.StaticFields[i])
	}

	// Add dynamic fields
	allFields = append(allFields, e.Fields...)

	// Add quantum fields if available
	if e.QuantumStore != nil {
		quantumFields := e.QuantumStore.GetAll()
		allFields = append(allFields, quantumFields...)
	}

	return allFields
}

// EnsureDynamicFields ensures that static fields are also available in the dynamic Fields slice
// This is used for backward compatibility with tests and existing code that expects fields in e.Fields
func (e *LogEntry) EnsureDynamicFields() {
	if e.StaticFieldCount > 0 && len(e.Fields) == 0 {
		// Static fields exist but dynamic slice is empty, populate it
		e.Fields = make([]TypedFieldData, 0, e.StaticFieldCount)
		for i := 0; i < e.StaticFieldCount; i++ {
			e.Fields = append(e.Fields, e.StaticFields[i])
		}
	}

	if e.StaticContextCount > 0 && len(e.Context) == 0 {
		// Static context exists but dynamic slice is empty, populate it
		e.Context = make([]TypedFieldData, 0, e.StaticContextCount)
		for i := 0; i < e.StaticContextCount; i++ {
			e.Context = append(e.Context, e.StaticContext[i])
		}
	}
}

// GetAllContext returns all context fields (both typed and quantum) as TypedField slices
func (e *LogEntry) GetAllContext() []TypedFieldData {
	// Calculate total context count
	totalContext := len(e.Context) + e.StaticContextCount

	// Pre-allocate with exact capacity
	allContext := make([]TypedFieldData, 0, totalContext)

	// Add static context first (FastPath optimization)
	for i := 0; i < e.StaticContextCount; i++ {
		allContext = append(allContext, e.StaticContext[i])
	}

	// Add dynamic context
	allContext = append(allContext, e.Context...)

	// Note: For now, quantum storage is only used for Fields, not Context
	// This can be extended in the future if needed

	return allContext
}

// IsUsingQuantumStorage returns true if quantum storage is enabled
func (e *LogEntry) IsUsingQuantumStorage() bool {
	return e.UseQuantumStorage && e.QuantumStore != nil
}

// MergeQuantumFieldsIntoTyped merges quantum fields into the typed fields array
// This is useful for backward compatibility or when you need all fields as TypedField
func (e *LogEntry) MergeQuantumFieldsIntoTyped() {
	if e.QuantumStore == nil {
		return
	}

	quantumFields := e.QuantumStore.GetAll()
	e.Fields = append(e.Fields, quantumFields...)
}

// Clone creates a deep copy of the log entry
func (e *LogEntry) Clone() *LogEntry {
	clone := &LogEntry{
		Timestamp:          e.Timestamp,
		Level:              e.Level,
		Message:            e.Message,
		Component:          e.Component,
		File:               e.File,
		Line:               e.Line,
		Fields:             make([]TypedFieldData, len(e.Fields)),
		Context:            make([]TypedFieldData, len(e.Context)),
		Error:              e.Error,
		Caller:             e.Caller,
		UseQuantumStorage:  e.UseQuantumStorage,
		StaticFieldCount:   e.StaticFieldCount,
		StaticContextCount: e.StaticContextCount,
	}

	copy(clone.Fields, e.Fields)
	copy(clone.Context, e.Context)

	// Copy static fields
	copy(clone.StaticFields, e.StaticFields)
	copy(clone.StaticContext, e.StaticContext)

	// Clone quantum store if present
	if e.QuantumStore != nil {
		clone.QuantumStore = NewEnhancedQuantumFieldStore(64) // Default chunk size
		// Copy all fields from original quantum store
		e.QuantumStore.Iterate(func(key, value string) {
			clone.QuantumStore.Set(key, value)
		})
	}

	// Ensure backward compatibility: if static fields exist but dynamic slice is empty,
	// populate the dynamic slice for tests and existing code
	clone.EnsureDynamicFields()

	return clone
}

// ToString converts the log entry to a string representation
func (e *LogEntry) ToString() string {
	var sb strings.Builder

	// Build timestamp, level, and message without AppendIntString
	sb.WriteString("[")
	sb.WriteString(e.Timestamp.Format(time.RFC3339))
	sb.WriteString("] ")
	sb.WriteString(e.Level.String())
	sb.WriteString(" - ")
	sb.WriteString(e.Message)

	if e.Component != "" {
		sb.WriteString(" [Component: ")
		sb.WriteString(e.Component)
		sb.WriteString("]")
	}

	if e.File != "" && e.Line > 0 {
		sb.WriteString(" [")
		sb.WriteString(e.File)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(e.Line))
		sb.WriteString("]")
	}

	// Get all fields (both typed and quantum)
	allFields := e.GetAllFields()
	if len(allFields) > 0 {
		sb.WriteString(" Fields: {")
		for i, field := range allFields {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(field.Key)
			sb.WriteString(": ")
			sb.WriteString(formatValue(field.Value))
		}
		sb.WriteString("}")
	}

	if e.Error != nil {
		sb.WriteString(" Error: ")
		sb.WriteString(e.Error.Error())
	}

	return sb.String()
}

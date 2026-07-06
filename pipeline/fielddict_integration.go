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

package pipeline

import "github.com/go-gen-ecosystem/halolog/fielddict"

// FieldDictIntegration provides integration with the global FieldDictionary
type FieldDictIntegration struct {
	dict *fielddict.FieldDictionary
}

// GlobalFieldDictIntegration is the singleton instance
var GlobalFieldDictIntegration *FieldDictIntegration

// InitializeFieldDictIntegration initializes the global integration instance
func InitializeFieldDictIntegration() {
	if GlobalFieldDictIntegration == nil {
		GlobalFieldDictIntegration = &FieldDictIntegration{
			dict: fielddict.GlobalFieldDictionary,
		}
	}
}

// GetOrRegisterFieldID gets an existing ID or registers a new one
func (f *FieldDictIntegration) GetOrRegisterFieldID(key string) int {
	return f.dict.GetOrRegisterFieldID(key)
}

// GetFieldID gets an existing ID if it exists
func (f *FieldDictIntegration) GetFieldID(key string) (int, bool) {
	return f.dict.GetID(key)
}

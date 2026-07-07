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
// Package pipeline tests exercise the FieldDict integration adapter.
// @author Admilson B. F. Cossa

package pipeline

import "testing"

// TestInitializeFieldDictIntegration_Idempotent verifies the global integration
// is created once and reused.
func TestInitializeFieldDictIntegration_Idempotent(t *testing.T) {
	InitializeFieldDictIntegration()
	first := GlobalFieldDictIntegration
	if first == nil {
		t.Fatal("expected global field dict integration to be initialized")
	}

	InitializeFieldDictIntegration()
	if GlobalFieldDictIntegration != first {
		t.Fatal("expected InitializeFieldDictIntegration to be idempotent")
	}
}

// TestFieldDictIntegration_RegisterAndLookup verifies GetOrRegisterFieldID returns
// a stable ID and GetFieldID finds registered keys but not unknown ones.
func TestFieldDictIntegration_RegisterAndLookup(t *testing.T) {
	InitializeFieldDictIntegration()
	f := GlobalFieldDictIntegration
	if f == nil {
		t.Skip("global field dict integration unavailable")
	}

	const key = "fielddict_integration_test_key"

	// First registration returns an ID; a second call returns the same ID.
	id1 := f.GetOrRegisterFieldID(key)
	id2 := f.GetOrRegisterFieldID(key)
	if id1 != id2 {
		t.Fatalf("expected stable field ID, got %d then %d", id1, id2)
	}

	// GetFieldID must now find the registered key.
	gotID, ok := f.GetFieldID(key)
	if !ok {
		t.Fatal("expected registered key to be found")
	}
	if gotID != id1 {
		t.Fatalf("expected GetFieldID to return %d, got %d", id1, gotID)
	}

	// An unregistered key must report ok=false.
	if _, ok := f.GetFieldID("fielddict_integration_definitely_unregistered_key"); ok {
		t.Fatal("expected unregistered key lookup to report ok=false")
	}
}

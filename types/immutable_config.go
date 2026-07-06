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

package types

import (
	"time"
)

// PolicyRule defines a logging policy rule
type PolicyRule struct {
	Name    string
	Pattern string
	Action  string // "allow", "deny", "mask"
}

// FileOutputConfig configures file output with rotation
type FileOutputConfig struct {
	Path       string
	MaxSize    int64
	MaxBackups int
	MaxAge     int
}

// AsyncBufferConfig configures async buffering for performance
type AsyncBufferConfig struct {
	BufferSize    int
	FlushInterval time.Duration
}

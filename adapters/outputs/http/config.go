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

package http

import (
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// HTTPAdapterConfig represents configuration for HTTP adapter.
//
//nolint:revive // HTTP prefix is the intended public API name; renaming would break callers across the module.
type HTTPAdapterConfig struct {
	URL                string
	Method             string
	Headers            map[string]string
	BatchSize          int
	FlushInterval      time.Duration
	Timeout            time.Duration
	Formatter          types.Formatter
	MaxIdleConns       int
	MaxConnsPerHost    int
	IdleConnTimeout    time.Duration
	DisableCompression bool
}

// DefaultHTTPAdapterConfig returns production-ready defaults
func DefaultHTTPAdapterConfig() *HTTPAdapterConfig {
	return &HTTPAdapterConfig{
		Method:             "POST",
		BatchSize:          100,
		FlushInterval:      5 * time.Second,
		Timeout:            30 * time.Second,
		Headers:            map[string]string{"Content-Type": "application/json"},
		MaxIdleConns:       100,
		MaxConnsPerHost:    100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}
}

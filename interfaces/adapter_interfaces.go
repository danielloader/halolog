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
// Package interfaces provides interface definitions
// Author: Admilson B. F. Cossa

package interfaces

import (
	"io"

	"github.com/go-gen-ecosystem/halolog/types"
)

// AdapterFactory creates adapter instances
type AdapterFactory interface {
	CreateConsoleAdapter() types.Adapter
	CreateFileAdapter(path string) (types.Adapter, error)
	CreateHTTPAdapter(url string) (types.Adapter, error)
	CreateSyslogAdapter(network string, address string) (types.Adapter, error)
	CreateDiscardAdapter() types.Adapter
}

// AsyncAdapterFactory creates asynchronous adapter instances
type AsyncAdapterFactory interface {
	CreateAsyncAdapter(adapter types.Adapter) types.AsyncAdapter
	CreateBufferedAdapter(adapter types.Adapter, bufferSize int) types.AsyncAdapter
}

// BatchAdapterFactory creates batch adapter instances
type BatchAdapterFactory interface {
	CreateBatchAdapter(adapter types.Adapter) types.BatchAdapter
	CreateBatchAdapterWithSize(adapter types.Adapter, batchSize int) types.BatchAdapter
}

// FormattedAdapterFactory creates formatted adapter instances
type FormattedAdapterFactory interface {
	CreateFormattedAdapter(adapter types.Adapter, formatter types.Formatter) types.FormattedAdapter
	CreateFormattedWriterAdapter(writer io.Writer, formatter types.Formatter) (types.WriterAdapter, error)
}

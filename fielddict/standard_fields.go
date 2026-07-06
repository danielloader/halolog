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

// Standard field constants for HaloLog best practices
const (
	// Core fields
	FieldUserID      = "user_id"
	FieldSessionID   = "session_id"
	FieldRequestID   = "request_id"
	FieldComponent   = "component"
	FieldLevel       = "level"
	FieldTimestamp   = "timestamp"
	FieldEnvironment = "environment"
	FieldService     = "service"
	FieldVersion     = "version"
	FieldDuration    = "duration"
	FieldStatus      = "status"
	FieldError       = "error"
	FieldStackTrace  = "stack_trace"
	FieldMessage     = "message"
	FieldFile        = "file"
	FieldLine        = "line"
	FieldFunction    = "function"

	// HTTP fields
	FieldMethod       = "method"
	FieldPath         = "path"
	FieldQuery        = "query"
	FieldStatusCode   = "status_code"
	FieldUserAgent    = "user_agent"
	FieldRemoteAddr   = "remote_addr"
	FieldRequestTime  = "request_time"
	FieldResponseTime = "response_time"
	FieldRequestSize  = "request_size"
	FieldResponseSize = "response_size"
	FieldIP           = "ip"
	FieldURL          = "url"
	FieldBytesIn      = "bytes_in"
	FieldBytesOut     = "bytes_out"

	// Database fields
	FieldQueryTime    = "query_time"
	FieldRowsAffected = "rows_affected"
	FieldTable        = "table"
	FieldDBInstance   = "db_instance"
	FieldDBName       = "db_name"
	FieldDBHost       = "db_host"
	FieldDBPort       = "db_port"
	FieldOperation    = "operation"

	// System fields
	FieldMemory       = "memory"
	FieldCPU          = "cpu"
	FieldDisk         = "disk"
	FieldNetwork      = "network"
	FieldUptime       = "uptime"
	FieldHostname     = "hostname"
	FieldPID          = "pid"
	FieldOS           = "os"
	FieldArch         = "arch"
	FieldNumGoroutine = "num_goroutine"

	// Performance monitoring fields
	FieldMemoryAlloc      = "memory_alloc"
	FieldMemoryTotalAlloc = "memory_total_alloc"
	FieldMemorySys        = "memory_sys"
	FieldNumCPU           = "num_cpu"
	FieldGoroutineID      = "goroutine_id"
	FieldLogger           = "logger"
	FieldNumGC            = "num_gc"
	FieldGCTime           = "gc_time"

	// Message Queue fields
	FieldQueueName     = "queue_name"
	FieldMessageID     = "message_id"
	FieldCorrelationID = "correlation_id"
	FieldRetryCount    = "retry_count"
	FieldDeadLetter    = "dead_letter"
	FieldTopic         = "topic"
	FieldPartition     = "partition"
	FieldOffset        = "offset"
	FieldLag           = "lag"
	FieldConsumerGroup = "consumer_group"
	FieldBroker        = "broker"

	// Tracing fields
	FieldTraceID = "trace_id"
	FieldSpanID  = "span_id"

	// Cache fields
	FieldKey       = "key"
	FieldValue     = "value"
	FieldHit       = "hit"
	FieldMiss      = "miss"
	FieldTTL       = "ttl"
	FieldCacheName = "cache_name"
	FieldLatency   = "latency"
	FieldBackend   = "backend"

	// Security fields
	FieldAuthMethod = "auth_method"
	FieldRole       = "role"
	FieldScope      = "scope"
	FieldAction     = "action"
	FieldResource   = "resource"
	FieldAllowed    = "allowed"
	FieldJWT        = "jwt"
	FieldToken      = "token"
	FieldHash       = "hash"
	FieldPassword   = "password"

	// Business fields
	FieldID            = "id"
	FieldCaseID        = "case_id"
	FieldFieldID       = "field_id"
	FieldCustomerID    = "customer_id"
	FieldTransactionID = "transaction_id"
	FieldAmount        = "amount"
	FieldCurrency      = "currency"
	FieldCountry       = "country"
	FieldProductID     = "product_id"

	// Cloud/Infrastructure fields
	FieldCloudProvider    = "cloud_provider"
	FieldRegion           = "region"
	FieldAvailabilityZone = "availability_zone"
	FieldAccountID        = "account_id"
	FieldProjectID        = "project_id"
	FieldInstanceID       = "instance_id"
	FieldInstanceType     = "instance_type"
	FieldZone             = "zone"
	FieldProject          = "project"
)

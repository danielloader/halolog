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
// Package middleware provides logging middleware
// Author: Admilson B. F. Cossa

package retry

import (
	"math"
	"math/rand/v2"
	"strings"
	"time"
)

// List of error patterns that indicate a permanent, non-retryable failure (normalized to lowercase).
// This list is generally used to quickly fail on bad client input, authorization issues, etc.
var nonRetryablePatterns = []string{
	"validation",
	"invalid",
	"malformed",
	"authentication",
	"authorization",
	"permission",
}

// RetryConfig provides retry configuration and logic.
//
//nolint:revive // Public API type; "Retry" prefix names the domain and renaming would break consumers.
type RetryConfig struct {
	maxRetries int
	baseDelay  time.Duration // Initial delay
	maxDelay   time.Duration // Hard cap on delay
	multiplier float64       // Exponential factor
	// Jitter is implicitly enabled by using Full Jitter logic in calculateBackoff
}

// NewRetryConfig creates a new retry configuration with resilient defaults
func NewRetryConfig(maxRetries int) *RetryConfig {
	return &RetryConfig{
		maxRetries: maxRetries,
		baseDelay:  100 * time.Millisecond,
		maxDelay:   30 * time.Second,
		multiplier: 2.0, // Standard exponential backoff
	}
}

// WithBaseDelay sets the base delay for retries. Must be non-negative.
func (d *RetryConfig) WithBaseDelay(delay time.Duration) *RetryConfig {
	if delay > 0 {
		d.baseDelay = delay
	}
	return d
}

// WithMaxDelay sets the maximum delay for retries. Must be non-negative.
func (d *RetryConfig) WithMaxDelay(delay time.Duration) *RetryConfig {
	if delay > 0 {
		d.maxDelay = delay
	}
	return d
}

// WithMultiplier sets the backoff multiplier. Must be > 1.0.
func (d *RetryConfig) WithMultiplier(multiplier float64) *RetryConfig {
	if multiplier > 1.0 {
		d.multiplier = multiplier
	}
	return d
}

// RetryWrite executes a write function with retry logic
func (d *RetryConfig) RetryWrite(writeFunc func() error) error {
	var lastErr error

	// Loop runs 1 initial attempt (attempt=0) + maxRetries additional attempts.
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		// Attempt the write operation
		err := writeFunc()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we have retries left and if the error is retryable
		if attempt >= d.maxRetries || !d.shouldRetry(err) {
			return err
		}

		// Calculate and wait for the backoff delay
		delay := d.calculateBackoff(attempt)
		time.Sleep(delay)
	}

	// Should be unreachable if maxRetries > 0, but serves as a final return.
	return lastErr
}

// calculateBackoff calculates the backoff delay using Exponential Backoff with Full Jitter.
// Full Jitter: delay is a random duration between 0 and the calculated exponential ceiling.
// attempt is 1-indexed (1 for first retry, 2 for second, etc.)
func (d *RetryConfig) calculateBackoff(attempt int) time.Duration {
	// The maximum duration for this attempt: BaseDelay * Multiplier^(attempt)
	// We use `attempt` directly (which is >= 1) for the exponent.
	delayCeiling := float64(d.baseDelay) * math.Pow(d.multiplier, float64(attempt))

	// Cap the deterministic delay ceiling at MaxDelay
	if delayCeiling > float64(d.maxDelay) {
		delayCeiling = float64(d.maxDelay)
	}

	// Apply Full Jitter: return a random duration between 0 and the ceiling
	// We must cast delayCeiling to int64 (nanoseconds) for the rand function.
	maxJitterNs := int64(delayCeiling)

	// RandInt64 returns a uniform random value in [0, n).
	jitterNs := rand.Int64N(maxJitterNs)

	//

	return time.Duration(jitterNs)
}

// shouldRetry determines if an error should be retried
func (d *RetryConfig) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Normalize the error string once for efficient case-insensitive comparison
	errStrLower := strings.ToLower(err.Error())

	// Check against the list of non-retryable patterns
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStrLower, pattern) {
			// Found a permanent failure pattern (e.g., "invalid credentials")
			return false
		}
	}

	// If no non-retryable pattern is found, assume it's a transient network/server error.
	return true
}

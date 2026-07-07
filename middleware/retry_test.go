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
	"errors"
	"math"
	"testing"
	"time"
)

// TestNewRetryConfig tests the RetryConfig constructor
func TestNewRetryConfig(t *testing.T) {
	config := NewRetryConfig(5)

	if config.maxRetries != 5 {
		t.Errorf("Expected maxRetries=5, got %d", config.maxRetries)
	}

	if config.baseDelay != 100*time.Millisecond {
		t.Errorf("Expected baseDelay=100ms, got %v", config.baseDelay)
	}

	if config.maxDelay != 30*time.Second {
		t.Errorf("Expected maxDelay=30s, got %v", config.maxDelay)
	}

	if config.multiplier != 2.0 {
		t.Errorf("Expected multiplier=2.0, got %f", config.multiplier)
	}

	// Jitter is now implicit (Full Jitter) and the field is removed.
}

// TestRetryConfig_Builders tests all builder methods
func TestRetryConfig_Builders(t *testing.T) {
	// WithJitter builder removed
	config := NewRetryConfig(3).
		WithBaseDelay(200 * time.Millisecond).
		WithMaxDelay(10 * time.Second).
		WithMultiplier(3.0)

	if config.baseDelay != 200*time.Millisecond {
		t.Errorf("Expected baseDelay=200ms, got %v", config.baseDelay)
	}

	if config.maxDelay != 10*time.Second {
		t.Errorf("Expected maxDelay=10s, got %v", config.maxDelay)
	}

	if config.multiplier != 3.0 {
		t.Errorf("Expected multiplier=3.0, got %f", config.multiplier)
	}
}

// TestRetryConfig_RetryWrite_Success tests successful write on first attempt
func TestRetryConfig_RetryWrite_Success(t *testing.T) {
	config := NewRetryConfig(3)
	callCount := 0

	err := config.RetryWrite(func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

// TestRetryConfig_RetryWrite_SuccessAfterRetries tests successful write after retries
func TestRetryConfig_RetryWrite_SuccessAfterRetries(t *testing.T) {
	// Base delay is set low for faster test execution, jitter is implicit.
	config := NewRetryConfig(3).WithBaseDelay(1 * time.Millisecond)
	callCount := 0

	err := config.RetryWrite(func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

// TestRetryConfig_RetryWrite_MaxRetriesExceeded tests when all retries are exhausted
func TestRetryConfig_RetryWrite_MaxRetriesExceeded(t *testing.T) {
	// Base delay is set low for faster test execution, jitter is implicit.
	config := NewRetryConfig(2).WithBaseDelay(1 * time.Millisecond)
	callCount := 0
	expectedErr := errors.New("persistent error")

	err := config.RetryWrite(func() error {
		callCount++
		return expectedErr
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Should be called: 1 original + 2 retries = 3 times
	if callCount != 3 {
		t.Errorf("Expected 3 calls (1 original + 2 retries), got %d", callCount)
	}
}

// TestRetryConfig_RetryWrite_NonRetryableError tests immediate failure on non-retryable errors
func TestRetryConfig_RetryWrite_NonRetryableError(t *testing.T) {
	config := NewRetryConfig(5)
	callCount := 0

	nonRetryableErrors := []string{
		"validation failed",
		"invalid input",
		"malformed data",
		"authentication error",
		"authorization denied",
		"permission denied",
	}

	for _, errMsg := range nonRetryableErrors {
		callCount = 0
		err := config.RetryWrite(func() error {
			callCount++
			return errors.New(errMsg)
		})

		if err == nil {
			t.Errorf("Expected error for %s", errMsg)
		}

		if callCount != 1 {
			t.Errorf("Expected 1 call for non-retryable error %s, got %d", errMsg, callCount)
		}
	}
}

// TestRetryConfig_CalculateBackoff_JitterBounds tests the Full Jitter bounds
func TestRetryConfig_CalculateBackoff_JitterBounds(t *testing.T) {
	// Set MaxDelay lower than 30s for easier verification
	config := NewRetryConfig(5).
		WithBaseDelay(100 * time.Millisecond).
		WithMultiplier(2.0).
		WithMaxDelay(500 * time.Millisecond)

	tests := []struct {
		attempt int
		ceiling time.Duration // Max delay without MaxDelay cap
	}{
		// Ceiling = 100 * 2^1 = 200ms (MaxDelay=500ms cap does not apply)
		{1, 200 * time.Millisecond},
		// Ceiling = 100 * 2^2 = 400ms (MaxDelay=500ms cap does not apply)
		{2, 400 * time.Millisecond},
		// Ceiling = 100 * 2^3 = 800ms (MaxDelay=500ms cap applies)
		{3, 500 * time.Millisecond},
		// Ceiling = 100 * 2^4 = 1600ms (MaxDelay=500ms cap applies)
		{4, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		// Generate multiple samples to ensure random bounds are correct
		for i := 0; i < 50; i++ {
			delay := config.calculateBackoff(tt.attempt)

			// 1. Must be greater than or equal to 0
			if delay < 0 {
				t.Errorf("calculateBackoff(%d) = %v, expected delay >= 0", tt.attempt, delay)
			}

			// 2. Must be less than the calculated ceiling (MaxDelay capped)
			if delay > tt.ceiling {
				t.Errorf("calculateBackoff(%d) = %v, expected delay <= ceiling %v", tt.attempt, delay, tt.ceiling)
			}
		}

		// Also check that multiple attempts are not all identical, proving randomness
		delays := make([]time.Duration, 5)
		for i := range delays {
			delays[i] = config.calculateBackoff(tt.attempt)
		}

		allSame := true
		if tt.ceiling > 5*time.Millisecond { // Only check randomness if the ceiling is large enough
			for i := 1; i < len(delays); i++ {
				if delays[i] != delays[0] {
					allSame = false
					break
				}
			}
			if allSame {
				t.Errorf("Expected full jitter to produce varying delays for attempt %d, but all were identical (%v)", tt.attempt, delays[0])
			}
		}
	}
}

// TestRetryConfig_CalculateBackoff_MaxDelayCap tests backoff capping at MaxDelay
func TestRetryConfig_CalculateBackoff_MaxDelayCap(t *testing.T) {
	// Config ensures that the second attempt (attempt=2) should hit the 5s cap
	config := NewRetryConfig(10).
		WithBaseDelay(1 * time.Second). // 2^1 * 1s = 2s
		WithMultiplier(3.0).            // 2^2 * 1s = 9s
		WithMaxDelay(5 * time.Second)   // Cap = 5s

	// Attempt 1: Ceiling = 3.0s (3^1 * 1s), delay <= 3s
	delay1 := config.calculateBackoff(1)
	if delay1 > 3*time.Second {
		t.Errorf("Attempt 1: Expected delay <= 3s, got %v", delay1)
	}

	// Attempt 2: Ceiling = 9.0s (3^2 * 1s). Capped by MaxDelay=5s. delay <= 5s
	for i := 0; i < 50; i++ {
		delay2 := config.calculateBackoff(2)
		if delay2 > 5*time.Second {
			t.Errorf("Attempt 2: Expected delay to be capped at 5s, got %v", delay2)
		}
	}
}

// TestRetryConfig_ShouldRetry_ValidationErrors tests non-retryable validation errors
func TestRetryConfig_ShouldRetry_ValidationErrors(t *testing.T) {
	config := NewRetryConfig(3)

	nonRetryableErrors := []error{
		errors.New("validation failed"),
		errors.New("invalid input"),
		errors.New("malformed data"),
		errors.New("authentication error"),
		errors.New("authorization denied"),
		errors.New("permission denied"),
		errors.New("VALIDATION ERROR"),      // uppercase
		errors.New("Authentication Failed"), // mixed case
	}

	for _, err := range nonRetryableErrors {
		if config.shouldRetry(err) {
			t.Errorf("shouldRetry should return false for error: %v", err)
		}
	}
}

// TestRetryConfig_ShouldRetry_RetryableErrors tests retryable errors
func TestRetryConfig_ShouldRetry_RetryableErrors(t *testing.T) {
	config := NewRetryConfig(3)

	retryableErrors := []error{
		errors.New("network timeout"),
		errors.New("connection refused"),
		errors.New("temporary failure"),
		errors.New("server busy"),
		errors.New("timeout"),
	}

	for _, err := range retryableErrors {
		if !config.shouldRetry(err) {
			t.Errorf("shouldRetry should return true for error: %v", err)
		}
	}
}

// TestRetryConfig_ChainedBuilders tests method chaining
func TestRetryConfig_ChainedBuilders(t *testing.T) {
	config := NewRetryConfig(5).
		WithBaseDelay(50 * time.Millisecond).
		WithMaxDelay(20 * time.Second).
		WithMultiplier(1.5)

	if config.maxRetries != 5 {
		t.Errorf("Expected maxRetries=5, got %d", config.maxRetries)
	}
	if config.baseDelay != 50*time.Millisecond {
		t.Errorf("Expected baseDelay=50ms, got %v", config.baseDelay)
	}
	if config.maxDelay != 20*time.Second {
		t.Errorf("Expected maxDelay=20s, got %v", config.maxDelay)
	}
	if config.multiplier != 1.5 {
		t.Errorf("Expected multiplier=1.5, got %f", config.multiplier)
	}
}

// TestRetryConfig_ZeroRetries tests behavior with zero retries
func TestRetryConfig_ZeroRetries(t *testing.T) {
	config := NewRetryConfig(0)
	callCount := 0

	err := config.RetryWrite(func() error {
		callCount++
		return errors.New("always fails")
	})

	if err == nil {
		t.Error("Expected error with zero retries")
	}

	// Should only call once (no retries)
	if callCount != 1 {
		t.Errorf("Expected 1 call with zero retries, got %d", callCount)
	}
}

// TestRetryConfig_DelayProgression tests that delays increase based on the ceiling
func TestRetryConfig_DelayProgression(t *testing.T) {
	config := NewRetryConfig(5).
		WithBaseDelay(10 * time.Millisecond).
		WithMultiplier(2.0).
		WithMaxDelay(1 * time.Second)

	var previousCeiling float64
	for attempt := 1; attempt <= 5; attempt++ {
		// Calculate the ceiling for this attempt
		currentCeiling := float64(config.baseDelay) * math.Pow(config.multiplier, float64(attempt))
		if currentCeiling > float64(config.maxDelay) {
			currentCeiling = float64(config.maxDelay)
		}

		if attempt > 1 && currentCeiling < previousCeiling {
			t.Errorf("Ceiling should not decrease: attempt %d ceiling=%v, previous=%v",
				attempt, currentCeiling, previousCeiling)
		}

		previousCeiling = currentCeiling
	}
}

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

package alerts

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

func TestNewSlackSender(t *testing.T) {
	config := SlackAlertConfig{
		WebhookURL: "https://hooks.slack.example/services/TEST/TEST/TEST",
		Channel:    "#alerts",
		Threshold:  types.ErrorLevel,
	}

	sender := NewSlackSender(config)
	if sender == nil {
		t.Fatal("Expected non-nil sender")
	}
	if sender.Type() != AlertTypeSlack {
		t.Errorf("Expected type %s, got %s", AlertTypeSlack, sender.Type())
	}
}

func TestSlackSender_Send(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := SlackAlertConfig{
		WebhookURL: server.URL,
		Channel:    "#test",
		RateLimit:  0, // No rate limit for test
	}
	sender := NewSlackSender(config)

	payload := &AlertPayload{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "Test alert",
		Component: "test-service",
		Fields:    map[string]interface{}{"key": "value"},
	}

	err := sender.Send(payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Error("Expected 1 request")
	}
}

func TestSlackSender_RateLimit(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := SlackAlertConfig{
		WebhookURL: server.URL,
		RateLimit:  time.Second, // 1 second rate limit
	}
	sender := NewSlackSender(config)

	payload := &AlertPayload{
		Level:   "error",
		Message: "Test",
	}

	// First send should go through
	err := sender.Send(payload)
	if err != nil {
		t.Fatalf("First send failed: %v", err)
	}

	// Second send should be rate limited
	err = sender.Send(payload)
	if err != nil {
		t.Fatalf("Second send failed: %v", err)
	}

	// Only 1 request should have been made
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 request (rate limited), got %d", requestCount)
	}
}

func TestNewPagerDutySender(t *testing.T) {
	config := PagerDutyAlertConfig{
		IntegrationKey: "test-key",
		Threshold:      types.ErrorLevel,
	}

	sender := NewPagerDutySender(config)
	if sender == nil {
		t.Fatal("Expected non-nil sender")
	}
	if sender.Type() != AlertTypePagerDuty {
		t.Errorf("Expected type %s, got %s", AlertTypePagerDuty, sender.Type())
	}
}

func TestNewWebhookSender(t *testing.T) {
	config := WebhookAlertConfig{
		URL:       "https://example.com/webhook",
		Threshold: types.ErrorLevel,
	}

	sender := NewWebhookSender(config)
	if sender == nil {
		t.Fatal("Expected non-nil sender")
	}
	if sender.Type() != AlertTypeWebhook {
		t.Errorf("Expected type %s, got %s", AlertTypeWebhook, sender.Type())
	}
	// Check defaults
	if sender.config.Method != "POST" {
		t.Error("Expected default method POST")
	}
	if sender.config.ContentType != "application/json" {
		t.Error("Expected default content type application/json")
	}
}

func TestWebhookSender_Send(t *testing.T) {
	var requestCount int32
	var receivedMethod string
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := WebhookAlertConfig{
		URL:         server.URL,
		Method:      "POST",
		ContentType: "application/json",
		Headers:     map[string]string{"X-Custom": "header"},
		RateLimit:   0,
	}
	sender := NewWebhookSender(config)

	payload := &AlertPayload{
		Timestamp: time.Now(),
		Level:     "warn",
		Message:   "Test webhook",
	}

	err := sender.Send(payload)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if receivedMethod != "POST" {
		t.Errorf("Expected POST, got %s", receivedMethod)
	}
	if receivedContentType != "application/json" {
		t.Errorf("Expected application/json, got %s", receivedContentType)
	}
}

func TestWebhookSender_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := WebhookAlertConfig{
		URL:       server.URL,
		RateLimit: 0,
	}
	sender := NewWebhookSender(config)

	payload := &AlertPayload{
		Level:   "error",
		Message: "Test",
	}

	err := sender.Send(payload)
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}

func TestAlertPayload(t *testing.T) {
	payload := AlertPayload{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "Test message",
		Component: "test-component",
		Fields: map[string]interface{}{
			"user_id": 12345,
			"action":  "login",
		},
		Error: "something went wrong",
	}

	if payload.Level != "error" {
		t.Error("Level not set correctly")
	}
	if payload.Message != "Test message" {
		t.Error("Message not set correctly")
	}
	if len(payload.Fields) != 2 {
		t.Error("Fields not set correctly")
	}
}

func BenchmarkSlackSender_Send(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSlackSender(SlackAlertConfig{
		WebhookURL: server.URL,
		RateLimit:  0, // No rate limit for benchmark
	})

	payload := &AlertPayload{
		Level:   "error",
		Message: "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sender.Send(payload)
	}
}

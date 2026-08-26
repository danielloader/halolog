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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-gen-ecosystem/halolog/types"
)

// AlertType defines the type of alert integration
type AlertType string

// Alert type identifiers for the supported alert integrations.
const (
	AlertTypeSlack     AlertType = "slack"
	AlertTypePagerDuty AlertType = "pagerduty"
	AlertTypeWebhook   AlertType = "webhook"
	AlertTypeEmail     AlertType = "email"
)

// SlackAlertConfig configures Slack webhook alerts
type SlackAlertConfig struct {
	WebhookURL string            `json:"webhook_url" yaml:"webhook_url"`
	Channel    string            `json:"channel,omitempty" yaml:"channel,omitempty"`
	Username   string            `json:"username,omitempty" yaml:"username,omitempty"`
	IconEmoji  string            `json:"icon_emoji,omitempty" yaml:"icon_emoji,omitempty"`
	Threshold  types.LogLevel    `json:"threshold" yaml:"threshold"`
	RateLimit  time.Duration     `json:"rate_limit" yaml:"rate_limit"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// PagerDutyAlertConfig configures PagerDuty alerts
type PagerDutyAlertConfig struct {
	IntegrationKey string         `json:"integration_key" yaml:"integration_key"`
	Severity       string         `json:"severity" yaml:"severity"` // critical, error, warning, info
	Threshold      types.LogLevel `json:"threshold" yaml:"threshold"`
	RateLimit      time.Duration  `json:"rate_limit" yaml:"rate_limit"`
}

// WebhookAlertConfig configures generic webhook alerts
type WebhookAlertConfig struct {
	URL         string            `json:"url" yaml:"url"`
	Method      string            `json:"method" yaml:"method"` // POST, PUT
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Threshold   types.LogLevel    `json:"threshold" yaml:"threshold"`
	RateLimit   time.Duration     `json:"rate_limit" yaml:"rate_limit"`
	Timeout     time.Duration     `json:"timeout" yaml:"timeout"`
	ContentType string            `json:"content_type" yaml:"content_type"`
}

// EmailAlertConfig configures email alerts
type EmailAlertConfig struct {
	SMTPHost  string         `json:"smtp_host" yaml:"smtp_host"`
	SMTPPort  int            `json:"smtp_port" yaml:"smtp_port"`
	From      string         `json:"from" yaml:"from"`
	To        []string       `json:"to" yaml:"to"`
	Username  string         `json:"username,omitempty" yaml:"username,omitempty"`
	Password  string         `json:"password,omitempty" yaml:"password,omitempty"`
	Threshold types.LogLevel `json:"threshold" yaml:"threshold"`
	RateLimit time.Duration  `json:"rate_limit" yaml:"rate_limit"`
}

// AlertPayload is the standard payload sent to webhooks
type AlertPayload struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// AlertSender provides a common interface for sending alerts
type AlertSender interface {
	Send(payload *AlertPayload) error
	Type() AlertType
}

// SlackSender implements webhook delivery for Slack
type SlackSender struct {
	config   SlackAlertConfig
	client   *http.Client
	mu       sync.Mutex
	lastSent time.Time
}

// NewSlackSender creates a Slack alert sender
func NewSlackSender(config SlackAlertConfig) *SlackSender {
	if config.RateLimit == 0 {
		config.RateLimit = time.Minute // Default 1 per minute
	}
	return &SlackSender{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Type returns the alert type
func (s *SlackSender) Type() AlertType {
	return AlertTypeSlack
}

// Send sends an alert to Slack
func (s *SlackSender) Send(payload *AlertPayload) error {
	s.mu.Lock()
	if time.Since(s.lastSent) < s.config.RateLimit {
		s.mu.Unlock()
		return nil // Rate limited, skip silently
	}
	s.lastSent = time.Now()
	s.mu.Unlock()

	// Build Slack message
	slackMsg := map[string]interface{}{
		"text": fmt.Sprintf("[%s] %s: %s", payload.Level, payload.Component, payload.Message),
	}
	if s.config.Channel != "" {
		slackMsg["channel"] = s.config.Channel
	}
	if s.config.Username != "" {
		slackMsg["username"] = s.config.Username
	}
	if s.config.IconEmoji != "" {
		slackMsg["icon_emoji"] = s.config.IconEmoji
	}

	// Add attachments for fields
	if len(payload.Fields) > 0 {
		fields := make([]map[string]interface{}, 0, len(payload.Fields))
		for k, v := range payload.Fields {
			fields = append(fields, map[string]interface{}{
				"title": k,
				"value": fmt.Sprintf("%v", v),
				"short": true,
			})
		}
		slackMsg["attachments"] = []map[string]interface{}{
			{"fields": fields},
		}
	}

	body, err := json.Marshal(slackMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// PagerDutySender implements webhook delivery for PagerDuty
type PagerDutySender struct {
	config   PagerDutyAlertConfig
	client   *http.Client
	mu       sync.Mutex
	lastSent time.Time
}

// NewPagerDutySender creates a PagerDuty alert sender
func NewPagerDutySender(config PagerDutyAlertConfig) *PagerDutySender {
	if config.RateLimit == 0 {
		config.RateLimit = time.Minute
	}
	if config.Severity == "" {
		config.Severity = "error"
	}
	return &PagerDutySender{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Type returns the alert type
func (p *PagerDutySender) Type() AlertType {
	return AlertTypePagerDuty
}

// Send sends an alert to PagerDuty
func (p *PagerDutySender) Send(payload *AlertPayload) error {
	p.mu.Lock()
	if time.Since(p.lastSent) < p.config.RateLimit {
		p.mu.Unlock()
		return nil
	}
	p.lastSent = time.Now()
	p.mu.Unlock()

	// PagerDuty Events API v2 format
	event := map[string]interface{}{
		"routing_key":  p.config.IntegrationKey,
		"event_action": "trigger",
		"payload": map[string]interface{}{
			"summary":        payload.Message,
			"severity":       p.config.Severity,
			"source":         payload.Component,
			"timestamp":      payload.Timestamp.Format(time.RFC3339),
			"custom_details": payload.Fields,
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty event: %w", err)
	}

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pagerduty event: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pagerduty returned status %d", resp.StatusCode)
	}

	return nil
}

// WebhookSender implements generic webhook delivery
type WebhookSender struct {
	config   WebhookAlertConfig
	client   *http.Client
	mu       sync.Mutex
	lastSent time.Time
}

// NewWebhookSender creates a generic webhook sender
func NewWebhookSender(config WebhookAlertConfig) *WebhookSender {
	if config.RateLimit == 0 {
		config.RateLimit = time.Minute
	}
	if config.Method == "" {
		config.Method = "POST"
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}
	return &WebhookSender{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// Type returns the alert type
func (w *WebhookSender) Type() AlertType {
	return AlertTypeWebhook
}

// Send sends an alert to the webhook
func (w *WebhookSender) Send(payload *AlertPayload) error {
	w.mu.Lock()
	if time.Since(w.lastSent) < w.config.RateLimit {
		w.mu.Unlock()
		return nil
	}
	w.lastSent = time.Now()
	w.mu.Unlock()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(w.config.Method, w.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", w.config.ContentType)

	for k, v := range w.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

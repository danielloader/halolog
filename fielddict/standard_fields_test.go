package fielddict

import (
	"testing"
)

func TestStandardFields(t *testing.T) {
	// Test standard field definitions
	if FieldUserID != "user_id" {
		t.Errorf("Expected FieldUserID to be 'user_id', got %s", FieldUserID)
	}
	if FieldRequestID != "request_id" {
		t.Errorf("Expected FieldRequestID to be 'request_id', got %s", FieldRequestID)
	}
	if FieldSessionID != "session_id" {
		t.Errorf("Expected FieldSessionID to be 'session_id', got %s", FieldSessionID)
	}
}

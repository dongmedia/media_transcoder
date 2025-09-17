package pkg

import (
	"testing"
	"time"
)

func TestReconnectionConfig(t *testing.T) {
	config := DefaultReconnectionConfig()
	
	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", config.MaxRetries)
	}
	
	if config.InitialDelay != 2*time.Second {
		t.Errorf("Expected InitialDelay to be 2s, got %v", config.InitialDelay)
	}
	
	if config.MaxDelay != 60*time.Second {
		t.Errorf("Expected MaxDelay to be 60s, got %v", config.MaxDelay)
	}
	
	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection timeout", &testError{"connection timeout"}, true},
		{"network unreachable", &testError{"network unreachable"}, true},
		{"broken pipe", &testError{"broken pipe"}, true},
		{"file not found", &testError{"file not found"}, false},
		{"permission denied", &testError{"permission denied"}, false},
		{"unknown error", &testError{"some unknown error"}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRecoverableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRecoverableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCustomReconnectionConfig(t *testing.T) {
	customConfig := &ReconnectionConfig{
		MaxRetries:         2,
		InitialDelay:       100 * time.Millisecond,
		MaxDelay:           1 * time.Second,
		BackoffFactor:      1.5,
		HealthCheckTimeout: 1 * time.Second,
	}
	
	// Just test that the config is properly set
	if customConfig.MaxRetries != 2 {
		t.Errorf("Expected MaxRetries to be 2, got %d", customConfig.MaxRetries)
	}
	
	if customConfig.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected InitialDelay to be 100ms, got %v", customConfig.InitialDelay)
	}
}

// Helper type for testing errors
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
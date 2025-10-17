package cli

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		expected string
	}{
		{
			name:     "Less than an hour",
			start:    time.Now().Add(-30 * time.Minute),
			expected: "0h30m",
		},
		{
			name:     "Exactly one hour",
			start:    time.Now().Add(-1 * time.Hour),
			expected: "1h0m",
		},
		{
			name:     "Multiple hours",
			start:    time.Now().Add(-5*time.Hour - 45*time.Minute),
			expected: "5h45m",
		},
		{
			name:     "One day",
			start:    time.Now().Add(-24 * time.Hour),
			expected: "24h0m",
		},
		{
			name:     "Multiple days",
			start:    time.Now().Add(-48*time.Hour - 15*time.Minute),
			expected: "48h15m",
		},
		{
			name:     "Just started (0 minutes)",
			start:    time.Now(),
			expected: "0h0m",
		},
		{
			name:     "59 minutes",
			start:    time.Now().Add(-59 * time.Minute),
			expected: "0h59m",
		},
		{
			name:     "61 minutes",
			start:    time.Now().Add(-61 * time.Minute),
			expected: "1h1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.start)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.start, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration_Precision(t *testing.T) {
	// Test that seconds are truncated (not rounded)
	start := time.Now().Add(-1*time.Hour - 30*time.Minute - 59*time.Second)
	result := formatDuration(start)
	expected := "1h30m"
	if result != expected {
		t.Errorf("formatDuration should truncate seconds, got %v, want %v", result, expected)
	}
}

func TestFormatDuration_NegativeDuration(t *testing.T) {
	// Test with future time (shouldn't happen in practice, but test for robustness)
	start := time.Now().Add(1 * time.Hour)
	result := formatDuration(start)
	// Should handle gracefully (will show negative or zero)
	// Just ensure it doesn't panic
	if result == "" {
		t.Errorf("formatDuration should return a value even for future times")
	}
}

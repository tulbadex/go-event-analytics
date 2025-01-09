package main

import (
	"testing"
	"time"
)

func TestFormatAsDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "",
		},
		{
			name:     "valid date",
			input:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: "2024/01/15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAsDate(tt.input)
			if result != tt.expected {
				t.Errorf("formatAsDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatDatetime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "",
		},
		{
			name:     "valid datetime",
			input:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			expected: "2024-01-15T14:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDatetime(tt.input)
			if result != tt.expected {
				t.Errorf("formatDatetime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "",
		},
		{
			name:     "valid display time",
			input:    time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			expected: "Jan 15, 2024 2:30 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatForDisplay(tt.input)
			if result != tt.expected {
				t.Errorf("formatForDisplay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

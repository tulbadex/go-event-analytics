package utils

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "Short string",
			input:    "Hello",
			length:   10,
			expected: "Hello",
		},
		{
			name:     "Long string",
			input:    "Hello World This Is A Test",
			length:   10,
			expected: "Hello Worl...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}
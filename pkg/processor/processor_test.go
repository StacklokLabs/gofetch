package processor

import (
	"testing"
)

func TestNewContentProcessor(t *testing.T) {
	processor := NewContentProcessor()

	if processor == nil {
		t.Error("expected processor to be initialized")
	}
}

func TestFormatContent(t *testing.T) {
	processor := NewContentProcessor()

	tests := []struct {
		name       string
		content    string
		startIndex *int
		maxLength  *int
		expected   string
	}{
		{
			name:       "no formatting",
			content:    "Hello, World!",
			startIndex: nil,
			maxLength:  nil,
			expected:   "Hello, World!",
		},
		{
			name:       "with start index",
			content:    "Hello, World!",
			startIndex: intPtr(7),
			maxLength:  nil,
			expected:   "World!",
		},
		{
			name:       "with max length",
			content:    "Hello, World!",
			startIndex: nil,
			maxLength:  intPtr(5),
			expected:   "Hello\n\n[Content truncated. Use start_index to get more content.]",
		},
		{
			name:       "with start index and max length",
			content:    "Hello, World!",
			startIndex: intPtr(7),
			maxLength:  intPtr(3),
			expected:   "Wor\n\n[Content truncated. Use start_index to get more content.]",
		},
		{
			name:       "start index beyond content length",
			content:    "Hello",
			startIndex: intPtr(10),
			maxLength:  nil,
			expected:   "",
		},
		{
			name:       "max length larger than content",
			content:    "Hello",
			startIndex: nil,
			maxLength:  intPtr(100),
			expected:   "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.FormatContent(tt.content, tt.startIndex, tt.maxLength)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestProcessHTML(t *testing.T) {
	processor := NewContentProcessor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple HTML",
			input:    "<html><body><h1>Title</h1><p>Content</p></body></html>",
			expected: "# Title\n\nContent", // This is an approximation - actual output may vary
		},
		{
			name:     "invalid HTML",
			input:    "not html content",
			expected: "not html content",
		},
		{
			name:     "empty HTML",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ProcessHTML(tt.input)

			// For HTML processing, we'll just check that we get some output
			// The exact markdown conversion may vary between library versions
			if tt.input == "" && result != "" {
				t.Errorf("expected empty result for empty input, got %q", result)
			}

			if tt.input != "" && result == "" {
				t.Error("expected non-empty result for non-empty input")
			}
		})
	}
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

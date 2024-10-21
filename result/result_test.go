package result

import (
	"testing"
)

func TestFormatName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "John 😀 Doe", // 含有 emoji
			expected: "John Doe",
		},
		{
			input:    "John    Doe", // 多个空格
			expected: "John Doe",
		},
		{
			input:    "😀John 😀 Doe😀", // 前后和中间含有 emoji
			expected: "John Doe",
		},
		{
			input:    "😀😀😀😀😀", // 纯 emoji
			expected: "",
		},
		{
			input:    "John Doe", // 无 emoji 和多余空格
			expected: "John Doe",
		},
		{
			input:    " John 😀  Doe  ", // 前后空格和 emoji
			expected: "John Doe",
		},
	}

	for _, test := range tests {
		result := formatName(test.input)
		if result != test.expected {
			t.Errorf("formatName(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

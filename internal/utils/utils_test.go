package utils

import (
	"testing"
)

func TestIsNumericType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		expected bool
	}{
		{"int type", "int(11)", true},
		{"varchar type", "varchar(255)", false},
		{"float type", "float", true},
		{"decimal type", "decimal(10,2)", true},
		{"text type", "text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNumericType(tt.typ); got != tt.expected {
				t.Errorf("IsNumericType(%q) = %v, want %v", tt.typ, got, tt.expected)
			}
		})
	}
}

func TestEscapeIdentifier(t *testing.T) {
	if got := EscapeIdentifier("PRIMARY"); got != "" {
		t.Errorf("EscapeIdentifier(PRIMARY) = %q, want empty string", got)
	}

	if got := EscapeIdentifier("id"); got != "`id`" {
		t.Errorf("EscapeIdentifier(id) = %q, want `id`", got)
	}
}

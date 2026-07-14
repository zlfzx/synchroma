package utils

import (
	"database/sql"
	"testing"

	"synchroma/pkg/models"
)

// ========== IsNumericType Tests ==========

func TestIsNumericType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		expected bool
	}{
		{"int type", "int(11)", true},
		{"bigint type", "bigint", true},
		{"smallint type", "smallint", true},
		{"tinyint type", "tinyint(1)", true},
		{"varchar type", "varchar(255)", false},
		{"float type", "float", true},
		{"double type", "double", true},
		{"decimal type", "decimal(10,2)", true},
		{"text type", "text", false},
		{"datetime type", "datetime", false},
		{"boolean type", "boolean", false},
		{"json type", "json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNumericType(tt.typ); got != tt.expected {
				t.Errorf("IsNumericType(%q) = %v, want %v", tt.typ, got, tt.expected)
			}
		})
	}
}

// ========== EscapeIdentifier Tests ==========

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PRIMARY returns empty", "PRIMARY", ""},
		{"simple name", "id", "`id`"},
		{"table name", "users", "`users`"},
		{"name with underscore", "user_name", "`user_name`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeIdentifier(tt.input); got != tt.expected {
				t.Errorf("EscapeIdentifier(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEscapeIdentifierPG(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"PRIMARY returns empty", "PRIMARY", ""},
		{"simple name", "id", `"id"`},
		{"table name", "users", `"users"`},
		{"name with underscore", "user_name", `"user_name"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeIdentifierPG(tt.input); got != tt.expected {
				t.Errorf("EscapeIdentifierPG(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ========== IsSameColumn Tests ==========

func TestIsSameColumn(t *testing.T) {
	colA := models.Column{
		ColumnType:      "varchar(255)",
		IsNullable:      "YES",
		ColumnDefault:   sql.NullString{String: "", Valid: false},
		Extra:           sql.NullString{String: "", Valid: false},
		ColumnComment:   sql.NullString{String: "", Valid: false},
		OrdinalPosition: 1,
	}

	t.Run("identical columns are same", func(t *testing.T) {
		colB := colA
		if !IsSameColumn(colA, colB) {
			t.Error("expected identical columns to be the same")
		}
	})

	t.Run("different type", func(t *testing.T) {
		colB := colA
		colB.ColumnType = "int(11)"
		if IsSameColumn(colA, colB) {
			t.Error("expected different ColumnType to not be the same")
		}
	})

	t.Run("different nullable", func(t *testing.T) {
		colB := colA
		colB.IsNullable = "NO"
		if IsSameColumn(colA, colB) {
			t.Error("expected different IsNullable to not be the same")
		}
	})

	t.Run("different default", func(t *testing.T) {
		colB := colA
		colB.ColumnDefault = sql.NullString{String: "0", Valid: true}
		if IsSameColumn(colA, colB) {
			t.Error("expected different ColumnDefault to not be the same")
		}
	})

	t.Run("different extra", func(t *testing.T) {
		colB := colA
		colB.Extra = sql.NullString{String: "auto_increment", Valid: true}
		if IsSameColumn(colA, colB) {
			t.Error("expected different Extra to not be the same")
		}
	})

	t.Run("different comment", func(t *testing.T) {
		colB := colA
		colB.ColumnComment = sql.NullString{String: "User ID", Valid: true}
		if IsSameColumn(colA, colB) {
			t.Error("expected different ColumnComment to not be the same")
		}
	})

	t.Run("different ordinal position", func(t *testing.T) {
		colB := colA
		colB.OrdinalPosition = 5
		if IsSameColumn(colA, colB) {
			t.Error("expected different OrdinalPosition to not be the same")
		}
	})
}

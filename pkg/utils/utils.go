package utils

import (
	"fmt"
	"strings"
	"synchroma/pkg/models"
)

func IsSameColumn(a, b models.Column) bool {
	return a.ColumnType == b.ColumnType &&
		a.IsNullable == b.IsNullable &&
		a.ColumnDefault.String == b.ColumnDefault.String &&
		a.Extra.String == b.Extra.String &&
		a.ColumnComment.String == b.ColumnComment.String &&
		a.OrdinalPosition == b.OrdinalPosition
}

func IsNumericType(typ string) bool {
	return strings.Contains(typ, "int") ||
		strings.Contains(typ, "float") ||
		strings.Contains(typ, "double") ||
		strings.Contains(typ, "decimal")
}

func EscapeIdentifier(name string) string {
	if name == "PRIMARY" {
		return ""
	}
	return fmt.Sprintf("`%s`", name)
}

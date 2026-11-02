package application

import (
	"fmt"
	"strings"
)

// ValidateUUID verifies the canonical 36-character UUID representation.
func ValidateUUID(value string) error {
	if len(value) != 36 ||
		value[8] != '-' ||
		value[13] != '-' ||
		value[18] != '-' ||
		value[23] != '-' {
		return fmt.Errorf("invalid UUID %q", value)
	}

	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return fmt.Errorf("invalid UUID %q", value)
		}
	}

	return nil
}

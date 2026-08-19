// Package marketdata — shared helpers.
package marketdata

import "strings"

// containsStr checks if a string contains a substring (case-sensitive).
func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

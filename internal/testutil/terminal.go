// Package testutil contains helpers shared by integration-style tests.
package testutil

import "github.com/charmbracelet/x/ansi"

// StripANSI removes terminal control sequences so tests can assert semantics.
func StripANSI(value string) string {
	return ansi.Strip(value)
}

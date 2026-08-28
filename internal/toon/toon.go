// Package toon provides small helpers for emitting TOON (Token-Oriented
// Object Notation) output from gh-axi's converted commands.
//
// The escaping here mirrors the @toon-format/toon encoder used by the TS
// gh-axi: a string value containing a comma, quote, backslash, newline, or
// other control character is double-quoted with escape sequences, so
// untrusted GitHub data (titles, descriptions, names) cannot break the TOON
// line structure or smuggle terminal/agent instruction text (R12).
package toon

import (
	"fmt"
	"strings"
)

// Quote renders s as a safe TOON scalar: plain values pass through
// untouched; values containing characters that would break TOON structure
// (`,`, `"`, `\`, control chars incl. newline) are double-quoted with
// escapes.
func Quote(s string) string {
	if !needsQuoting(s) {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func needsQuoting(s string) bool {
	for _, r := range s {
		if r == ',' || r == '"' || r == '\\' || r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

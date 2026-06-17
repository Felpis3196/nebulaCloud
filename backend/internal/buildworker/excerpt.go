package buildworker

import (
	"fmt"
	"strings"
)

// DefaultExcerptLimit is the total runes kept when summarizing long command output.
const DefaultExcerptLimit = 6000

// Excerpt keeps the head and tail of long command output so the real error
// (usually at the end) is preserved. limit is total runes kept.
func Excerpt(s string, limit int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if limit <= 0 || len(r) <= limit {
		return s
	}
	head := limit / 4
	tail := limit - head
	omitted := len(r) - limit
	return string(r[:head]) +
		fmt.Sprintf("\n…(%d chars omitted)…\n", omitted) +
		string(r[len(r)-tail:])
}

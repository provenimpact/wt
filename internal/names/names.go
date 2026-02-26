package names

import (
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9\-.]`)
var multiDash = regexp.MustCompile(`-{2,}`)

// Sanitize converts a branch name into a safe, flat directory name.
// Characters not matching [a-zA-Z0-9-.] are replaced with "-".
// Consecutive "-" characters are collapsed into a single "-".
// Leading and trailing "-" characters are trimmed.
func Sanitize(branch string) string {
	s := unsafeChars.ReplaceAllString(branch, "-")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

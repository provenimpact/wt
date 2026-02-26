package fuzzy

import "unicode/utf8"

// Scoring constants
const (
	bonusFirstChar    = 16 // pattern[0] matches str[0]
	bonusCamelCase    = 16 // match at camelCase boundary (lowercase→uppercase)
	bonusSeparator    = 16 // match after separator: - _ . /
	bonusAdjacent     = 8  // consecutive matched characters
	penaltyLeadingGap = -3 // per-character penalty for gap before first match
	penaltyGap        = -1 // per-character penalty for gaps between matches
)

// Match holds the result of scoring a string against a pattern.
type Match struct {
	Score     int   // Quality rating; higher is better.
	Matched   bool  // Whether the pattern matched at all.
	Positions []int // Indices of matched characters in str (for highlighting).
}

// Score scores str against pattern using a greedy forward-scan algorithm
// with contextual bonuses. Case-insensitive matching.
func Score(str, pattern string) Match {
	if pattern == "" {
		return Match{Score: 0, Matched: true, Positions: nil}
	}
	if str == "" {
		return Match{Score: 0, Matched: false, Positions: nil}
	}

	strLower := toLower(str)
	patLower := toLower(pattern)

	strRunes := []rune(strLower)
	patRunes := []rune(patLower)
	origRunes := []rune(str)

	if len(patRunes) > len(strRunes) {
		return Match{Score: 0, Matched: false, Positions: nil}
	}

	positions := make([]int, 0, len(patRunes))
	score := 0
	pi := 0
	prevMatchIdx := -1

	for si := 0; si < len(strRunes) && pi < len(patRunes); si++ {
		if strRunes[si] != patRunes[pi] {
			continue
		}

		// Match found
		positions = append(positions, si)

		if pi == 0 {
			// Leading gap penalty
			score += si * penaltyLeadingGap
		} else {
			// Gap penalty between matches
			gap := si - prevMatchIdx - 1
			if gap > 0 {
				score += gap * penaltyGap
			}
		}

		// First character bonus
		if si == 0 {
			score += bonusFirstChar
		}

		// Separator boundary bonus
		if si > 0 && isSeparator(origRunes[si-1]) {
			score += bonusSeparator
		}

		// CamelCase boundary bonus
		if si > 0 && isLower(origRunes[si-1]) && isUpper(origRunes[si]) {
			score += bonusCamelCase
		}

		// Adjacency bonus
		if prevMatchIdx == si-1 {
			score += bonusAdjacent
		}

		prevMatchIdx = si
		pi++
	}

	if pi < len(patRunes) {
		return Match{Score: 0, Matched: false, Positions: nil}
	}

	return Match{Score: score, Matched: true, Positions: positions}
}

func isSeparator(r rune) bool {
	return r == '-' || r == '_' || r == '.' || r == '/'
}

func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func toLower(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		buf := make([]byte, utf8.RuneLen(r))
		utf8.EncodeRune(buf, r)
		b = append(b, buf...)
		i += size
	}
	return string(b)
}

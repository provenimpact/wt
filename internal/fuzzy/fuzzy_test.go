package fuzzy

import (
	"sort"
	"testing"
)

func TestScore_ExactMatch(t *testing.T) {
	m := Score("feature-auth", "feature-auth")
	if !m.Matched {
		t.Fatal("exact match should match")
	}
	if m.Score <= 0 {
		t.Errorf("exact match score should be positive, got %d", m.Score)
	}
	if len(m.Positions) != len("feature-auth") {
		t.Errorf("positions length = %d, want %d", len(m.Positions), len("feature-auth"))
	}
}

func TestScore_PrefixMatch(t *testing.T) {
	m := Score("feature-auth", "fea")
	if !m.Matched {
		t.Fatal("prefix should match")
	}
	if m.Score <= 0 {
		t.Errorf("prefix match score should be positive, got %d", m.Score)
	}
	// First char bonus + adjacency bonuses
	if m.Positions[0] != 0 {
		t.Errorf("first position should be 0, got %d", m.Positions[0])
	}
}

func TestScore_SeparatorBoundary(t *testing.T) {
	// Test that a match right after a separator gets the separator bonus.
	// Use a pattern where the second char can only match after a separator.
	m := Score("feature-auth", "fau")
	if !m.Matched {
		t.Fatal("separator boundary should match")
	}
	// 'f' at 0, 'a' at 2 (greedy first 'a' in "feature"), 'u' at 9
	if len(m.Positions) != 3 {
		t.Fatalf("positions length = %d, want 3", len(m.Positions))
	}

	// Test that separator bonus is awarded: "x-y" matching "y" should get separator bonus
	withSep := Score("x-y", "y")
	withoutSep := Score("xy", "y")
	if !withSep.Matched || !withoutSep.Matched {
		t.Fatal("both should match")
	}
	if withSep.Score <= withoutSep.Score {
		t.Errorf("separator match score (%d) should be greater than non-separator (%d)", withSep.Score, withoutSep.Score)
	}
}

func TestScore_CamelCaseBoundary(t *testing.T) {
	// Test that camelCase boundary gets a bonus.
	// "xAuth" matching "a" -- 'A' at index 1 is a camelCase boundary (after lowercase 'x').
	withCamel := Score("xAuth", "a")
	withoutCamel := Score("xauth", "a")
	if !withCamel.Matched || !withoutCamel.Matched {
		t.Fatal("both should match")
	}
	if withCamel.Score <= withoutCamel.Score {
		t.Errorf("camelCase match score (%d) should be greater than non-camelCase (%d)", withCamel.Score, withoutCamel.Score)
	}
}

func TestScore_AdjacencyBonus(t *testing.T) {
	// "abc" in "abcdef" should score higher than "abc" in "axbxcx"
	adjacent := Score("abcdef", "abc")
	gapped := Score("axbxcx", "abc")
	if !adjacent.Matched || !gapped.Matched {
		t.Fatal("both should match")
	}
	if adjacent.Score <= gapped.Score {
		t.Errorf("adjacent score (%d) should be greater than gapped score (%d)", adjacent.Score, gapped.Score)
	}
}

func TestScore_GapPenalty(t *testing.T) {
	small := Score("a-b", "ab")
	large := Score("a-----b", "ab")
	if !small.Matched || !large.Matched {
		t.Fatal("both should match")
	}
	if small.Score <= large.Score {
		t.Errorf("small gap score (%d) should be greater than large gap score (%d)", small.Score, large.Score)
	}
}

func TestScore_NonMatch(t *testing.T) {
	m := Score("feature-auth", "xyz")
	if m.Matched {
		t.Error("non-matching pattern should not match")
	}
	if m.Score != 0 {
		t.Errorf("non-match score should be 0, got %d", m.Score)
	}
	if m.Positions != nil {
		t.Error("non-match positions should be nil")
	}
}

func TestScore_PatternLongerThanStr(t *testing.T) {
	m := Score("ab", "abcd")
	if m.Matched {
		t.Error("pattern longer than string should not match")
	}
}

func TestScore_EmptyPattern(t *testing.T) {
	m := Score("feature-auth", "")
	if !m.Matched {
		t.Error("empty pattern should always match")
	}
	if m.Score != 0 {
		t.Errorf("empty pattern score should be 0, got %d", m.Score)
	}
	if m.Positions != nil {
		t.Errorf("empty pattern positions should be nil, got %v", m.Positions)
	}
}

func TestScore_EmptyString(t *testing.T) {
	m := Score("", "abc")
	if m.Matched {
		t.Error("empty string with non-empty pattern should not match")
	}
}

func TestScore_CaseInsensitive(t *testing.T) {
	m := Score("FeatureAuth", "featureauth")
	if !m.Matched {
		t.Fatal("case-insensitive match should work")
	}
}

func TestScore_SortOrder(t *testing.T) {
	entries := []string{
		"some-random-thing",
		"switch-main",
		"status-module",
	}
	pattern := "sm"

	type scored struct {
		entry string
		score int
	}
	var results []scored
	for _, e := range entries {
		m := Score(e, pattern)
		if m.Matched {
			results = append(results, scored{entry: e, score: m.Score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// switch-main and status-module both have 's' at position 0 (first char bonus)
	// then 'm' after separator. "some-random-thing" has 's' at 0 then 'm' at index 5 in "random".
	// Both separator-boundary matches should rank above the gapped match.
	if len(results) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(results))
	}
	// The entry "some-random-thing" should rank last (large gap to 'm')
	if results[len(results)-1].entry != "some-random-thing" {
		t.Errorf("expected 'some-random-thing' to rank last, got %q", results[len(results)-1].entry)
	}
}

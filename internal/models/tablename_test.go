package models

import "testing"

func TestCanonicalPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"se7-tomb", "SE7-tomb"},   // lowercase se7 -> SE7, suffix preserved
		{"SE7-tomb", "SE7-tomb"},   // already canonical
		{"Se7-4860", "SE7-4860"},   // mixed case leading literal
		{"se7-AC", "SE7-AC"},       // suffix casing preserved
		{"sE7-LC2", "SE7-LC2"},     // odd casing normalised
		{"dev", "dev"},             // non-se7 untouched
		{"", ""},                   // empty untouched
		{"other-env", "other-env"}, // unrelated prefix untouched
	}
	for _, c := range cases {
		if got := CanonicalPrefix(c.in); got != c.want {
			t.Errorf("CanonicalPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTableName(t *testing.T) {
	cases := []struct {
		prefix, suffix, want string
	}{
		{"se7-tomb", "GameWeekFixtures", "SE7-tomb-GameWeekFixtures"},
		{"SE7-4860", "GameWeek", "SE7-4860-GameWeek"},
		{"se7-tomb", "GameWeek", "SE7-tomb-GameWeek"},
		{"", "GameWeekFixtures", "dev-GameWeekFixtures"},
		{"dev", "GameWeek", "dev-GameWeek"},
	}
	for _, c := range cases {
		if got := TableName(c.prefix, c.suffix); got != c.want {
			t.Errorf("TableName(%q, %q) = %q, want %q", c.prefix, c.suffix, got, c.want)
		}
	}
}

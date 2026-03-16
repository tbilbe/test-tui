package aws

import "testing"

func TestBuildEventSource(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{"with prefix", "SE7-3062", "SE7-3062.gameWeekManagement"},
		{"empty prefix", "", "int-dev.gameWeekManagement"},
		{"dev prefix", "dev", "int-dev.gameWeekManagement"},
		{"int-dev prefix", "int-dev", "int-dev.gameWeekManagement"},
		{"other prefix", "SE7-1234", "SE7-1234.gameWeekManagement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildEventSource(tt.prefix)
			if got != tt.want {
				t.Errorf("BuildEventSource(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

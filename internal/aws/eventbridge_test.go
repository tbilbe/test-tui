package aws

import (
	"context"
	"testing"

	"github.com/angstromsports/seven-test-tui/internal/models"
)

func TestBuildEventSource(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{"with prefix", "SE7-3062", "SE7-3062.gameWeekManagement"},
		{"lowercase prefix", "se7-tomb", "SE7-tomb.gameWeekManagement"},
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

func TestCloseGameWeekRefusesDisallowedPrefixBeforeSendingEvent(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "false")
	client := EventBridgeClient{}

	err := client.CloseGameWeek(context.Background(), "dev", "gameweek-1")
	if err == nil {
		t.Fatal("CloseGameWeek() error = nil, want write permission error")
	}
}

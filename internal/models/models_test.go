package models

import (
	"os"
	"testing"
)

func TestValidatePeriod(t *testing.T) {
	tests := []struct {
		name    string
		period  FixturePeriod
		wantErr bool
	}{
		{"valid PRE_MATCH", PeriodPreMatch, false},
		{"valid FIRST_HALF", PeriodFirstHalf, false},
		{"valid HALF_TIME", PeriodHalfTime, false},
		{"valid SECOND_HALF", PeriodSecondHalf, false},
		{"valid FULL_TIME", PeriodFullTime, false},
		{"invalid period", FixturePeriod("INVALID"), true},
		{"empty period", FixturePeriod(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePeriod(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePeriod(%q) error = %v, wantErr %v", tt.period, err, tt.wantErr)
			}
		})
	}
}

func TestValidateClockTime(t *testing.T) {
	tests := []struct {
		name    string
		period  FixturePeriod
		min     int
		wantErr bool
	}{
		{"PRE_MATCH at 0", PeriodPreMatch, 0, false},
		{"PRE_MATCH at 1 invalid", PeriodPreMatch, 1, true},
		{"FIRST_HALF at 0", PeriodFirstHalf, 0, false},
		{"FIRST_HALF at 45", PeriodFirstHalf, 45, false},
		{"FIRST_HALF at 46 invalid", PeriodFirstHalf, 46, true},
		{"FIRST_HALF negative min invalid", PeriodFirstHalf, -1, true},
		{"HALF_TIME at 45", PeriodHalfTime, 45, false},
		{"HALF_TIME at 0", PeriodHalfTime, 0, false},
		{"HALF_TIME at 46 invalid", PeriodHalfTime, 46, true},
		{"SECOND_HALF at 90", PeriodSecondHalf, 90, false},
		{"SECOND_HALF at 44 invalid", PeriodSecondHalf, 44, true},
		{"FULL_TIME at 90", PeriodFullTime, 90, false},
		{"FULL_TIME at 44 invalid", PeriodFullTime, 44, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClockTime(tt.period, tt.min)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClockTime(%q, %d) error = %v, wantErr %v",
					tt.period, tt.min, err, tt.wantErr)
			}
		})
	}
}

func TestValidateScore(t *testing.T) {
	tests := []struct {
		name    string
		score   *int
		wantErr bool
	}{
		{"nil score valid", nil, false},
		{"zero score valid", intPtr(0), false},
		{"positive score valid", intPtr(5), false},
		{"negative score invalid", intPtr(-1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScore(tt.score)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScore(%v) error = %v, wantErr %v", tt.score, err, tt.wantErr)
			}
		})
	}
}

func TestValidateStartDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		wantErr bool
	}{
		{"valid RFC3339", "2024-03-10T14:30:00Z", false},
		{"valid RFC3339 with offset", "2024-03-10T14:30:00+01:00", false},
		{"invalid format", "2024-03-10", true},
		{"invalid format with time", "2024-03-10 14:30:00", true},
		{"empty string", "", true},
		{"garbage", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStartDate(tt.dateStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStartDate(%q) error = %v, wantErr %v", tt.dateStr, err, tt.wantErr)
			}
		})
	}
}

func TestFixture_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fixture Fixture
		wantErr bool
	}{
		{
			name: "valid fixture PRE_MATCH",
			fixture: Fixture{
				Period:       PeriodPreMatch,
				ClockTimeMin: 0,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: false,
		},
		{
			name: "valid fixture FIRST_HALF with scores",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				HomeScore:    intPtr(1),
				AwayScore:    intPtr(0),
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: false,
		},
		{
			name: "invalid period",
			fixture: Fixture{
				Period:       FixturePeriod("INVALID"),
				ClockTimeMin: 0,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid clock time for period",
			fixture: Fixture{
				Period:       PeriodPreMatch,
				ClockTimeMin: 10,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "negative home score",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				HomeScore:    intPtr(-1),
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "negative away score",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				AwayScore:    intPtr(-1),
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid start date",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				StartDate:    "invalid-date",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fixture.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Fixture.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

func TestFixture_ApplyPreset(t *testing.T) {
	futureStart := "2026-03-12T15:40:00Z"
	pastStart := "2026-03-12T15:20:00Z"

	tests := []struct {
		name          string
		preset        string
		fixtureStatus string
		wantPeriod    FixturePeriod
		wantStatus    string
		wantClockMin  int
		wantStartDate string
	}{
		{"prematch resets all fields", "prematch", "FIXTURE", PeriodPreMatch, "", 0, futureStart},
		{"kickoff preserves fixture status", "kickoff", "FIXTURE", PeriodFirstHalf, "FIXTURE", 0, pastStart},
		{"kickoff preserves non-default fixture status", "kickoff", "POSTPONED", PeriodFirstHalf, "POSTPONED", 0, pastStart},
		{"halftime sets half time", "halftime", "FIXTURE", PeriodHalfTime, "FIXTURE", 45, ""},
		{"secondhalf sets second half", "secondhalf", "FIXTURE", PeriodSecondHalf, "FIXTURE", 45, ""},
		{"fulltime sets full time", "fulltime", "FIXTURE", PeriodFullTime, "FIXTURE", 90, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fixture{
				FixtureStatus: tt.fixtureStatus,
				Period:        PeriodFullTime,
				ClockTimeMin:  90,
				HomeScore:     intPtr(2),
				AwayScore:     intPtr(1),
				Goals:         []Goal{{GoalID: "goal-1"}},
				StartDate:     "2026-03-12T12:00:00Z",
			}

			originalFixtureStatus := f.FixtureStatus
			f.ApplyPreset(tt.preset, futureStart, pastStart)

			if f.Period != tt.wantPeriod {
				t.Errorf("Period = %v, want %v", f.Period, tt.wantPeriod)
			}
			if f.FixtureStatus != tt.wantStatus {
				t.Errorf("FixtureStatus = %v, want %v", f.FixtureStatus, tt.wantStatus)
			}
			if tt.preset == "kickoff" && f.FixtureStatus != originalFixtureStatus {
				t.Errorf("FixtureStatus = %v, want original value %v", f.FixtureStatus, originalFixtureStatus)
			}
			if f.ClockTimeMin != tt.wantClockMin {
				t.Errorf("ClockTimeMin = %v, want %v", f.ClockTimeMin, tt.wantClockMin)
			}
			if tt.preset == "prematch" {
				if f.HomeScore != nil || f.AwayScore != nil || f.Goals != nil {
					t.Error("prematch did not reset scores and goals")
				}
			}
			if tt.wantStartDate != "" && f.StartDate != tt.wantStartDate {
				t.Errorf("StartDate = %v, want %v", f.StartDate, tt.wantStartDate)
			}
		})
	}
}

func TestFixture_ApplyPreset_UnknownPreset(t *testing.T) {
	f := &Fixture{
		Period:       PeriodFullTime,
		ClockTimeMin: 90,
	}

	// Unknown preset should not modify fixture
	f.ApplyPreset("unknown", "future", "past")

	if f.Period != PeriodFullTime {
		t.Errorf("Period changed for unknown preset")
	}
	if f.ClockTimeMin != 90 {
		t.Errorf("ClockTimeMin changed for unknown preset")
	}
}

func TestGameWeek_ApplyPreMatchReset(t *testing.T) {
	gw := &GameWeek{
		GameWeekID:        "30",
		Label:             "30",
		FixturesStartDate: "2026-03-12T12:00:00Z",
		FixturesEndDate:   "2026-03-17T02:54:00Z",
		CustomerStartDate: "2026-03-06T03:00:00Z",
		CustomerEndDate:   "2026-03-17T02:59:59Z",
	}

	futureStart := "2026-03-12T15:40:00Z"
	gw.ApplyPreMatchReset(futureStart)

	if gw.FixturesStartDate != futureStart {
		t.Errorf("FixturesStartDate = %v, want %v", gw.FixturesStartDate, futureStart)
	}

	// Other fields should remain unchanged
	if gw.GameWeekID != "30" {
		t.Errorf("GameWeekID changed unexpectedly")
	}
	if gw.FixturesEndDate != "2026-03-17T02:54:00Z" {
		t.Errorf("FixturesEndDate changed unexpectedly")
	}
	if gw.CustomerStartDate != "2026-03-06T03:00:00Z" {
		t.Errorf("CustomerStartDate changed unexpectedly")
	}
}

func TestGameWeek_ApplyKickoffReset(t *testing.T) {
	gw := &GameWeek{
		GameWeekID:        "30",
		Label:             "30",
		FixturesStartDate: "2026-03-12T16:00:00Z", // Future time (pre-match)
		FixturesEndDate:   "2026-03-17T02:54:00Z",
		CustomerStartDate: "2026-03-06T03:00:00Z",
		CustomerEndDate:   "2026-03-17T02:59:59Z",
	}

	pastStart := "2026-03-12T15:50:00Z" // Now - 5 mins (kickoff)
	gw.ApplyKickoffReset(pastStart)

	if gw.FixturesStartDate != pastStart {
		t.Errorf("FixturesStartDate = %v, want %v", gw.FixturesStartDate, pastStart)
	}

	// Other fields should remain unchanged
	if gw.GameWeekID != "30" {
		t.Errorf("GameWeekID changed unexpectedly")
	}
}

func TestGameWeek_ShouldUpdateStartDate(t *testing.T) {
	tests := []struct {
		name              string
		fixturesStartDate string
		checkTime         string
		want              bool
	}{
		{
			name:              "future start date should update when going live",
			fixturesStartDate: "2026-03-12T16:00:00Z",
			checkTime:         "2026-03-12T15:50:00Z",
			want:              true,
		},
		{
			name:              "past start date should not update",
			fixturesStartDate: "2026-03-12T14:00:00Z",
			checkTime:         "2026-03-12T15:50:00Z",
			want:              false,
		},
		{
			name:              "same time should not update",
			fixturesStartDate: "2026-03-12T15:50:00Z",
			checkTime:         "2026-03-12T15:50:00Z",
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := &GameWeek{
				GameWeekID:        "30",
				FixturesStartDate: tt.fixturesStartDate,
			}

			got := gw.ShouldUpdateStartDate(tt.checkTime)
			if got != tt.want {
				t.Errorf("ShouldUpdateStartDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixture_IsGoingLive(t *testing.T) {
	tests := []struct {
		name   string
		preset string
		want   bool
	}{
		{"kickoff goes live", "kickoff", true},
		{"halftime goes live", "halftime", true},
		{"secondhalf goes live", "secondhalf", true},
		{"fulltime goes live", "fulltime", true},
		{"prematch does not go live", "prematch", false},
		{"unknown does not go live", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLivePreset(tt.preset)
			if got != tt.want {
				t.Errorf("IsLivePreset(%q) = %v, want %v", tt.preset, got, tt.want)
			}
		})
	}
}

func TestAllFixturesPreMatch(t *testing.T) {
	tests := []struct {
		name     string
		fixtures []Fixture
		want     bool
	}{
		{
			name: "all pre-match returns true",
			fixtures: []Fixture{
				{Period: PeriodPreMatch},
				{Period: PeriodPreMatch},
				{Period: PeriodPreMatch},
			},
			want: true,
		},
		{
			name: "one live returns false",
			fixtures: []Fixture{
				{Period: PeriodPreMatch},
				{Period: PeriodFirstHalf},
				{Period: PeriodPreMatch},
			},
			want: false,
		},
		{
			name: "all live returns false",
			fixtures: []Fixture{
				{Period: PeriodFullTime},
				{Period: PeriodSecondHalf},
			},
			want: false,
		},
		{
			name:     "empty fixtures returns true",
			fixtures: []Fixture{},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllFixturesPreMatch(tt.fixtures)
			if got != tt.want {
				t.Errorf("AllFixturesPreMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPositionValues(t *testing.T) {
	tests := []struct {
		name     string
		position Position
		want     string
	}{
		{"forward", PositionForward, "Forward"},
		{"midfielder", PositionMidfielder, "Midfielder"},
		{"defender", PositionDefender, "Defender"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.position) != tt.want {
				t.Errorf("position = %q, want %q", tt.position, tt.want)
			}
		})
	}
}

func TestIsWriteAllowed(t *testing.T) {
	allowWrites := "true"
	tests := []struct {
		name     string
		envValue *string
		prefix   string
		want     bool
	}{
		{"environment variable unset", nil, "SE7-tomb", false},
		{"int-dev requires environment variable", nil, "int-dev", false},
		{"canonical SE7 prefix", &allowWrites, "SE7-tomb", true},
		{"lowercase SE7 prefix", &allowWrites, "se7-tomb", true},
		{"test prefix", &allowWrites, "test", false},
		{"stage prefix", &allowWrites, "stage", false},
		{"prod prefix", &allowWrites, "prod", false},
		{"dev prefix", &allowWrites, "dev", false},
		{"empty prefix", &allowWrites, "", false},
		{"int-dev prefix", &allowWrites, "int-dev", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == nil {
				originalValue, wasSet := os.LookupEnv(SEVEN_TUI_ALLOW_WRITES)
				if err := os.Unsetenv(SEVEN_TUI_ALLOW_WRITES); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if wasSet {
						_ = os.Setenv(SEVEN_TUI_ALLOW_WRITES, originalValue)
						return
					}
					_ = os.Unsetenv(SEVEN_TUI_ALLOW_WRITES)
				})
			} else {
				t.Setenv(SEVEN_TUI_ALLOW_WRITES, *tt.envValue)
			}

			if got := IsWriteAllowed(tt.prefix); got != tt.want {
				t.Errorf("IsWriteAllowed(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

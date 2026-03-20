package models

import (
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
		sec     int
		wantErr bool
	}{
		// PRE_MATCH
		{"PRE_MATCH at 0:00", PeriodPreMatch, 0, 0, false},
		{"PRE_MATCH at 1:00 invalid", PeriodPreMatch, 1, 0, true},
		{"PRE_MATCH at 0:01 invalid", PeriodPreMatch, 0, 1, true},

		// FIRST_HALF
		{"FIRST_HALF at 0:00", PeriodFirstHalf, 0, 0, false},
		{"FIRST_HALF at 22:30", PeriodFirstHalf, 22, 30, false},
		{"FIRST_HALF at 45:00", PeriodFirstHalf, 45, 0, false},
		{"FIRST_HALF at 45:59", PeriodFirstHalf, 45, 59, false},
		{"FIRST_HALF at 46:00 invalid", PeriodFirstHalf, 46, 0, true},
		{"FIRST_HALF negative min invalid", PeriodFirstHalf, -1, 0, true},

		// HALF_TIME
		{"HALF_TIME at 45:00", PeriodHalfTime, 45, 0, false},
		{"HALF_TIME at 0:00", PeriodHalfTime, 0, 0, false},
		{"HALF_TIME at 46:00 invalid", PeriodHalfTime, 46, 0, true},

		// SECOND_HALF
		{"SECOND_HALF at 45:00", PeriodSecondHalf, 45, 0, false},
		{"SECOND_HALF at 60:00", PeriodSecondHalf, 60, 0, false},
		{"SECOND_HALF at 90:00", PeriodSecondHalf, 90, 0, false},
		{"SECOND_HALF at 44:00 invalid", PeriodSecondHalf, 44, 0, true},

		// FULL_TIME
		{"FULL_TIME at 90:00", PeriodFullTime, 90, 0, false},
		{"FULL_TIME at 45:00", PeriodFullTime, 45, 0, false},
		{"FULL_TIME at 44:00 invalid", PeriodFullTime, 44, 0, true},

		// Invalid seconds
		{"negative seconds invalid", PeriodFirstHalf, 30, -1, true},
		{"60 seconds invalid", PeriodFirstHalf, 30, 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClockTime(tt.period, tt.min, tt.sec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateClockTime(%q, %d, %d) error = %v, wantErr %v",
					tt.period, tt.min, tt.sec, err, tt.wantErr)
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
				ClockTimeSec: 0,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: false,
		},
		{
			name: "valid fixture FIRST_HALF with scores",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				ClockTimeSec: 45,
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
				ClockTimeSec: 0,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid clock time for period",
			fixture: Fixture{
				Period:       PeriodPreMatch,
				ClockTimeMin: 10,
				ClockTimeSec: 0,
				StartDate:    "2024-03-10T14:30:00Z",
			},
			wantErr: true,
		},
		{
			name: "negative home score",
			fixture: Fixture{
				Period:       PeriodFirstHalf,
				ClockTimeMin: 30,
				ClockTimeSec: 0,
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
				ClockTimeSec: 0,
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
				ClockTimeSec: 0,
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
		wantPeriod    FixturePeriod
		wantStatus    string
		wantClockMin  int
		wantClockSec  int
		wantHomeScore *int
		wantAwayScore *int
		wantGoalsNil  bool
		wantStartDate string
	}{
		{
			name:          "prematch resets all fields",
			preset:        "prematch",
			wantPeriod:    PeriodPreMatch,
			wantStatus:    "",
			wantClockMin:  0,
			wantClockSec:  0,
			wantHomeScore: nil,
			wantAwayScore: nil,
			wantGoalsNil:  true,
			wantStartDate: futureStart,
		},
		{
			name:          "kickoff sets first half",
			preset:        "kickoff",
			wantPeriod:    PeriodFirstHalf,
			wantStatus:    "IN_PLAY",
			wantClockMin:  0,
			wantClockSec:  0,
			wantStartDate: pastStart,
		},
		{
			name:         "halftime sets half time",
			preset:       "halftime",
			wantPeriod:   PeriodHalfTime,
			wantClockMin: 45,
			wantClockSec: 0,
		},
		{
			name:         "secondhalf sets second half",
			preset:       "secondhalf",
			wantPeriod:   PeriodSecondHalf,
			wantClockMin: 45,
			wantClockSec: 0,
		},
		{
			name:         "fulltime sets full time",
			preset:       "fulltime",
			wantPeriod:   PeriodFullTime,
			wantClockMin: 90,
			wantClockSec: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with a fixture that has scores and goals (simulating post-match state)
			homeScore := 2
			awayScore := 1
			f := &Fixture{
				FixtureID:     "fixture-1",
				GameWeekID:    "30",
				Period:        PeriodFullTime,
				FixtureStatus: "FINISHED",
				ClockTimeMin:  90,
				ClockTimeSec:  0,
				HomeScore:     &homeScore,
				AwayScore:     &awayScore,
				Goals:         []Goal{{GoalID: "goal-1"}},
				StartDate:     "2026-03-12T12:00:00Z",
			}

			f.ApplyPreset(tt.preset, futureStart, pastStart)

			if f.Period != tt.wantPeriod {
				t.Errorf("Period = %v, want %v", f.Period, tt.wantPeriod)
			}
			if f.ClockTimeMin != tt.wantClockMin {
				t.Errorf("ClockTimeMin = %v, want %v", f.ClockTimeMin, tt.wantClockMin)
			}
			if f.ClockTimeSec != tt.wantClockSec {
				t.Errorf("ClockTimeSec = %v, want %v", f.ClockTimeSec, tt.wantClockSec)
			}

			// Check fields specific to prematch
			if tt.preset == "prematch" {
				if f.FixtureStatus != "" {
					t.Errorf("FixtureStatus = %v, want empty", f.FixtureStatus)
				}
				if f.HomeScore != nil {
					t.Errorf("HomeScore = %v, want nil", f.HomeScore)
				}
				if f.AwayScore != nil {
					t.Errorf("AwayScore = %v, want nil", f.AwayScore)
				}
				if f.Goals != nil {
					t.Errorf("Goals = %v, want nil", f.Goals)
				}
				if f.StartDate != futureStart {
					t.Errorf("StartDate = %v, want %v", f.StartDate, futureStart)
				}
			}

			// Check kickoff specific fields
			if tt.preset == "kickoff" {
				if f.FixtureStatus != "IN_PLAY" {
					t.Errorf("FixtureStatus = %v, want IN_PLAY", f.FixtureStatus)
				}
				if f.StartDate != pastStart {
					t.Errorf("StartDate = %v, want %v", f.StartDate, pastStart)
				}
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

func TestIsDevEnv(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   bool
	}{
		{"empty string is dev", "", true},
		{"dev string is dev", "dev", true},
		{"branch prefix is not dev", "SE7-2001", false},
		{"int-dev is not dev", "int-dev", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDevEnv(tt.prefix); got != tt.want {
				t.Errorf("IsDevEnv(%q) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

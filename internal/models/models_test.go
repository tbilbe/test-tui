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

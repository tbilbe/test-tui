package models

import (
	"fmt"
	"time"
)

type FixturePeriod string

const (
	PeriodPreMatch   FixturePeriod = "PRE_MATCH"
	PeriodFirstHalf  FixturePeriod = "FIRST_HALF"
	PeriodHalfTime   FixturePeriod = "HALF_TIME"
	PeriodSecondHalf FixturePeriod = "SECOND_HALF"
	PeriodFullTime   FixturePeriod = "FULL_TIME"
)

var AllPeriods = []FixturePeriod{
	PeriodPreMatch,
	PeriodFirstHalf,
	PeriodHalfTime,
	PeriodSecondHalf,
	PeriodFullTime,
}

type Position string

const (
	PositionForward    Position = "FORWARD"
	PositionMidfielder Position = "MIDFIELDER"
	PositionDefender   Position = "DEFENDER"
)

type Player struct {
	PlayerID    string   `json:"playerId"`
	FirstName   string   `json:"firstName"`
	LastName    string   `json:"lastName"`
	PlayerName  string   `json:"playerName"`
	Position    Position `json:"position"`
	TeamID      string   `json:"teamId"`
	TeamName    string   `json:"teamName"`
	ShirtNumber int      `json:"shirtNumber"`
	FixtureID   string   `json:"fixtureId"`
}

type Selection struct {
	GameWeekID string
	Player1    *Player
	Player2    *Player
	Player3    *Player
	Player4    *Player
	Player5    *Player
	Player6    *Player
	Player7    *Player
}

type GameWeek struct {
	GameWeekID            string `json:"gameWeekId" dynamodbav:"gameWeekId"`
	Label                 string `json:"label" dynamodbav:"label"`
	FixturesStartDate     string `json:"fixturesStartDate" dynamodbav:"fixturesStartDate"`
	FixturesEndDate       string `json:"fixturesEndDate" dynamodbav:"fixturesEndDate"`
	CustomerStartDate     string `json:"customerStartDate" dynamodbav:"customerStartDate"`
	CustomerEndDate       string `json:"customerEndDate" dynamodbav:"customerEndDate"`
	CompetitionCalendarID string `json:"competitionCalendarId" dynamodbav:"competitionCalendarId"`
	Locked                *bool  `json:"locked,omitempty" dynamodbav:"locked,omitempty"`
}

type Team struct {
	TeamID           string `json:"teamId" dynamodbav:"teamId"`
	TeamName         string `json:"teamName" dynamodbav:"teamName"`
	TeamNameShort    string `json:"teamNameShort" dynamodbav:"teamNameShort"`
	TeamNameOfficial string `json:"teamNameOfficial" dynamodbav:"teamNameOfficial"`
	TeamCode         string `json:"teamCode" dynamodbav:"teamCode"`
}

type Participants struct {
	Home Team `json:"home" dynamodbav:"home"`
	Away Team `json:"away" dynamodbav:"away"`
}

type Goal struct {
	GoalID        string        `json:"goalId" dynamodbav:"goalId"`
	TimeMin       int           `json:"timeMin" dynamodbav:"timeMin"`
	TimeSec       int           `json:"timeSec" dynamodbav:"timeSec"`
	Type          string        `json:"type" dynamodbav:"type"`
	PlayerID      string        `json:"playerId" dynamodbav:"playerId"`
	PlayerName    string        `json:"playerName" dynamodbav:"playerName"`
	Period        FixturePeriod `json:"period" dynamodbav:"period"`
	TeamID        string        `json:"teamId" dynamodbav:"teamId"`
	SevenGoalType string        `json:"sevenGoalType" dynamodbav:"sevenGoalType"`
}

type Fixture struct {
	FixtureID     string                 `json:"fixtureId" dynamodbav:"fixtureId"`
	GameWeekID    string                 `json:"gameWeekId" dynamodbav:"gameWeekId"`
	StartDate     string                 `json:"startDate" dynamodbav:"startDate"`
	Period        FixturePeriod          `json:"period" dynamodbav:"period"`
	ClockTimeMin  int                    `json:"clockTimeMin" dynamodbav:"clockTimeMin"`
	ClockTimeSec  int                    `json:"clockTimeSec" dynamodbav:"clockTimeSec"`
	HomeScore     *int                   `json:"homeScore,omitempty" dynamodbav:"homeScore,omitempty"`
	AwayScore     *int                   `json:"awayScore,omitempty" dynamodbav:"awayScore,omitempty"`
	HomeTeamID    string                 `json:"homeTeamId" dynamodbav:"homeTeamId"`
	AwayTeamID    string                 `json:"awayTeamId" dynamodbav:"awayTeamId"`
	Participants  Participants           `json:"participants" dynamodbav:"participants"`
	Goals         []Goal                 `json:"goals,omitempty" dynamodbav:"goals,omitempty"`
	FixtureStatus string                 `json:"fixtureStatus,omitempty" dynamodbav:"fixtureStatus,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" dynamodbav:"metadata,omitempty"`
}

// Validation functions

func ValidatePeriod(period FixturePeriod) error {
	for _, p := range AllPeriods {
		if p == period {
			return nil
		}
	}
	return fmt.Errorf("invalid period: %s", period)
}

func ValidateClockTime(period FixturePeriod, min, sec int) error {
	if sec < 0 || sec > 59 {
		return fmt.Errorf("seconds must be between 0 and 59")
	}

	switch period {
	case PeriodPreMatch:
		if min != 0 || sec != 0 {
			return fmt.Errorf("PRE_MATCH must have time 00:00")
		}
	case PeriodFirstHalf, PeriodHalfTime:
		if min < 0 || min > 45 {
			return fmt.Errorf("first half time must be between 0 and 45 minutes")
		}
	case PeriodSecondHalf, PeriodFullTime:
		if min < 45 {
			return fmt.Errorf("second half time must be 45 minutes or more")
		}
	}

	return nil
}

func ValidateScore(score *int) error {
	if score != nil && *score < 0 {
		return fmt.Errorf("score cannot be negative")
	}
	return nil
}

func ValidateStartDate(dateStr string) error {
	_, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format, expected RFC3339: %w", err)
	}
	return nil
}

func (f *Fixture) Validate() error {
	if err := ValidatePeriod(f.Period); err != nil {
		return err
	}
	if err := ValidateClockTime(f.Period, f.ClockTimeMin, f.ClockTimeSec); err != nil {
		return err
	}
	if err := ValidateScore(f.HomeScore); err != nil {
		return err
	}
	if err := ValidateScore(f.AwayScore); err != nil {
		return err
	}
	if err := ValidateStartDate(f.StartDate); err != nil {
		return err
	}
	return nil
}

// ApplyPreset applies a batch preset to a fixture, modifying it in place.
// futureStart should be RFC3339 formatted time for prematch preset.
// pastStart should be RFC3339 formatted time for kickoff preset.
func (f *Fixture) ApplyPreset(preset string, futureStart, pastStart string) {
	switch preset {
	case "prematch":
		f.Period = PeriodPreMatch
		f.FixtureStatus = ""
		f.ClockTimeMin = 0
		f.ClockTimeSec = 0
		f.HomeScore = nil
		f.AwayScore = nil
		f.Goals = nil
		f.StartDate = futureStart
	case "kickoff":
		f.Period = PeriodFirstHalf
		f.FixtureStatus = "IN_PLAY"
		f.ClockTimeMin = 0
		f.ClockTimeSec = 0
		f.StartDate = pastStart
	case "halftime":
		f.Period = PeriodHalfTime
		f.ClockTimeMin = 45
		f.ClockTimeSec = 0
	case "secondhalf":
		f.Period = PeriodSecondHalf
		f.ClockTimeMin = 45
		f.ClockTimeSec = 0
	case "fulltime":
		f.Period = PeriodFullTime
		f.ClockTimeMin = 90
		f.ClockTimeSec = 0
	}
}

// ApplyPreMatchReset resets a GameWeek's fixturesStartDate for pre-match state.
func (g *GameWeek) ApplyPreMatchReset(futureStart string) {
	g.FixturesStartDate = futureStart
}

// ApplyKickoffReset sets GameWeek's fixturesStartDate for when fixtures go live.
func (g *GameWeek) ApplyKickoffReset(pastStart string) {
	g.FixturesStartDate = pastStart
}

// ShouldUpdateStartDate returns true if the gameweek's fixturesStartDate is in the future
// relative to checkTime, meaning it needs to be updated when a fixture goes live.
func (g *GameWeek) ShouldUpdateStartDate(checkTime string) bool {
	gwStart, err1 := time.Parse(time.RFC3339, g.FixturesStartDate)
	check, err2 := time.Parse(time.RFC3339, checkTime)
	if err1 != nil || err2 != nil {
		return false
	}
	return gwStart.After(check)
}

// IsLivePreset returns true if the preset puts fixtures into a live/in-play state.
func IsLivePreset(preset string) bool {
	switch preset {
	case "kickoff", "halftime", "secondhalf", "fulltime":
		return true
	}
	return false
}

// AllFixturesPreMatch returns true if all fixtures are in PRE_MATCH state.
func AllFixturesPreMatch(fixtures []Fixture) bool {
	for _, f := range fixtures {
		if f.Period != PeriodPreMatch {
			return false
		}
	}
	return true
}

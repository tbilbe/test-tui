package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/angstromsports/seven-test-tui/internal/models"
)

func TestBuildFixtureUpdateExpressionOmitsUnpopulatedFields(t *testing.T) {
	expr, err := buildFixtureUpdateExpression(models.Fixture{
		FixtureID:    "fixture-1",
		GameWeekID:   "gameweek-1",
		StartDate:    "2026-10-02T10:00:00Z",
		Period:       models.PeriodFirstHalf,
		ClockTimeMin: 0,
	})
	if err != nil {
		t.Fatalf("buildFixtureUpdateExpression() error = %v", err)
	}

	names := expressionNames(expr.Names())
	for _, name := range []string{"startDate", "period", "clockTimeMin"} {
		if !names[name] {
			t.Errorf("generated expression omitted populated %q", name)
		}
	}
	for _, name := range []string{
		"participants", "homeScore", "awayScore", "goals", "fixtureStatus",
		"metadata", "homeTeamId", "awayTeamId", "cards", "substitutes",
		"homeTeamLineupConfirmed", "awayTeamLineupConfirmed",
	} {
		if names[name] {
			t.Errorf("generated expression included unpopulated or unmodelled %q", name)
		}
	}
}

func TestBuildGameWeekUpdateExpressionOmitsEmptyCompetitionCalendarID(t *testing.T) {
	expr, err := buildGameWeekUpdateExpression(models.GameWeek{
		GameWeekID:        "gameweek-1",
		Label:             "Week 1",
		FixturesStartDate: "2026-10-02T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("buildGameWeekUpdateExpression() error = %v", err)
	}

	names := expressionNames(expr.Names())
	if names["competitionCalendarId"] {
		t.Error("generated expression included empty competitionCalendarId")
	}
	for _, name := range []string{"label", "fixturesStartDate"} {
		if !names[name] {
			t.Errorf("generated expression omitted populated %q", name)
		}
	}
}

func TestFixtureKeyUsesConfirmedCompositeKey(t *testing.T) {
	key := fixtureKey(models.Fixture{GameWeekID: "gameweek-1", FixtureID: "fixture-1"})
	if len(key) != 2 {
		t.Fatalf("fixture key has %d attributes, want 2", len(key))
	}
	if key["gameWeekId"] == nil || key["fixtureId"] == nil {
		t.Errorf("fixture key = %v, want gameWeekId and fixtureId", key)
	}
}

func TestUpdateFixtureRefusesDisallowedPrefixBeforeCallingDynamoDB(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "false")
	client := DynamoDBClient{prefix: "dev"}

	err := client.UpdateFixture(context.Background(), models.Fixture{})
	if err == nil {
		t.Fatal("UpdateFixture() error = nil, want write permission error")
	}
}

func expressionNames(names map[string]string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}

func TestUpdateGameWeekRefusesDisallowedPrefixBeforeCallingDynamoDB(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "false")
	client := DynamoDBClient{prefix: "dev"}

	err := client.UpdateGameWeek(context.Background(), models.GameWeek{})
	if err == nil {
		t.Fatal("UpdateGameWeek() error = nil, want write permission error")
	}
}

func TestBuildFixtureUpdateExpressionPrematchRemovesScoresAndGoals(t *testing.T) {
	fixture := models.Fixture{
		StartDate:    "2026-10-02T10:00:00Z",
		Period:       models.PeriodFullTime,
		ClockTimeMin: 90,
		HomeScore:    intPointer(2),
		AwayScore:    intPointer(1),
		Goals:        []models.Goal{{GoalID: "goal-1"}},
	}
	fixture.ApplyPreset("prematch", "2026-10-03T10:00:00Z", "")

	expr, err := buildFixtureUpdateExpression(fixture)
	if err != nil {
		t.Fatalf("buildFixtureUpdateExpression() error = %v", err)
	}
	if update := *expr.Update(); !strings.Contains(update, "REMOVE") {
		t.Errorf("UpdateExpression = %q, want REMOVE", update)
	}

	names := expressionNames(expr.Names())
	for _, name := range []string{"homeScore", "awayScore", "goals"} {
		if !names[name] {
			t.Errorf("generated expression omitted removal for %q", name)
		}
	}
}

func TestBuildFixtureUpdateExpressionSetsScoresWithoutRemovingThem(t *testing.T) {
	fixture := models.Fixture{
		HomeScore: intPointer(2),
		AwayScore: intPointer(1),
		Goals:     []models.Goal{{GoalID: "goal-1"}},
	}

	expr, err := buildFixtureUpdateExpression(fixture)
	if err != nil {
		t.Fatalf("buildFixtureUpdateExpression() error = %v", err)
	}
	if update := *expr.Update(); strings.Contains(update, "REMOVE") {
		t.Errorf("UpdateExpression = %q, must not remove populated scores", update)
	}

	names := expressionNames(expr.Names())
	for _, name := range []string{"homeScore", "awayScore", "goals"} {
		if !names[name] {
			t.Errorf("generated expression omitted populated %q", name)
		}
	}
}

func TestBuildFixtureUpdateExpressionPreservesParticipants(t *testing.T) {
	expr, err := buildFixtureUpdateExpression(models.Fixture{
		Participants: models.Participants{
			Home: models.Team{TeamID: "home", TeamNameOfficial: "Home Official"},
			Away: models.Team{TeamID: "away", TeamNameOfficial: "Away Official"},
		},
	})
	if err != nil {
		t.Fatalf("buildFixtureUpdateExpression() error = %v", err)
	}
	if expressionNames(expr.Names())["participants"] {
		t.Error("generated expression overwrote backend-owned participants")
	}
}

func intPointer(value int) *int {
	return &value
}

func TestUpdateExpressionBuildsRemoveOnlyExpression(t *testing.T) {
	var update updateExpression
	update.remove("homeScore")

	expr, err := update.build(errors.New("no update actions"))
	if err != nil {
		t.Fatalf("build() error = %v", err)
	}
	if expression := *expr.Update(); !strings.Contains(expression, "REMOVE") {
		t.Errorf("UpdateExpression = %q, want REMOVE", expression)
	}
}

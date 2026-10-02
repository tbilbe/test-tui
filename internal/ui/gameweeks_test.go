package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/angstromsports/seven-test-tui/internal/aws"
	"github.com/angstromsports/seven-test-tui/internal/models"
)

func TestUpdateFixtureCmdRefusesDisallowedPrefixDirectly(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "false")
	fixture := models.Fixture{
		FixtureID:    "fixture-1",
		GameWeekID:   "gameweek-1",
		StartDate:    "2026-10-02T10:00:00Z",
		Period:       models.PeriodPreMatch,
		ClockTimeMin: 0,
	}

	message := updateFixtureCmd(fixture, "dev", nil, []models.Fixture{fixture})()
	result, ok := message.(fixtureUpdatedMsg)
	if !ok {
		t.Fatalf("command message = %T, want fixtureUpdatedMsg", message)
	}
	if result.err == nil {
		t.Fatal("command error = nil, want write permission error")
	}
	if fixture.FixtureStatus != "" {
		t.Errorf("fixture status = %q, command mutated caller-owned fixture", fixture.FixtureStatus)
	}
}

func TestBatchUpdateFixturesCmdDoesNotMutateInputWhenValidationFails(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "true")
	fixtures := []models.Fixture{
		{
			FixtureID:    "fixture-1",
			GameWeekID:   "gameweek-1",
			StartDate:    "2026-10-02T10:00:00Z",
			Period:       models.PeriodPreMatch,
			ClockTimeMin: 0,
		},
		{
			FixtureID:    "fixture-2",
			GameWeekID:   "gameweek-1",
			StartDate:    "not-a-date",
			Period:       models.PeriodPreMatch,
			ClockTimeMin: 0,
		},
	}

	message := batchUpdateFixturesCmd(fixtures, "unknown", "SE7-test", nil)()
	result, ok := message.(batchUpdatedMsg)
	if !ok {
		t.Fatalf("command message = %T, want batchUpdatedMsg", message)
	}
	if result.err == nil {
		t.Fatal("command error = nil, want validation error")
	}
	if fixtures[0].Metadata != nil {
		t.Error("batch command mutated caller-owned fixture metadata")
	}
}

func TestUpdateMergesWriteResultsWithoutSharingCommandInputs(t *testing.T) {
	callerFixtures := []models.Fixture{{
		FixtureID:     "fixture-1",
		GameWeekID:    "gameweek-1",
		FixtureStatus: "CALLER",
	}}
	callerGameWeek := &models.GameWeek{GameWeekID: "gameweek-1", Label: "caller"}

	_ = updateFixtureCmd(callerFixtures[0], "SE7-test", callerGameWeek, callerFixtures)
	_ = batchUpdateFixturesCmd(callerFixtures, "kickoff", "SE7-test", callerGameWeek)

	model := Model{state: models.NewAppState()}
	model.state.SetFixtures(slices.Clone(callerFixtures))
	model.state.SetCurrentGameWeek(&models.GameWeek{GameWeekID: "gameweek-1", Label: "state"})

	mutationsStarted := make(chan struct{})
	mutationsFinished := make(chan struct{})
	go func() {
		close(mutationsStarted)
		for i := 0; i < 100000; i++ {
			callerFixtures[0].FixtureStatus = "CALLER-MUTATED"
			_ = callerFixtures[0].FixtureStatus
			callerGameWeek.Label = "caller-mutated"
			_ = callerGameWeek.Label
		}
		callerFixtures[0].FixtureStatus = "CALLER-COMPLETE"
		callerGameWeek.Label = "caller-complete"
		close(mutationsFinished)
	}()
	<-mutationsStarted

	fixtureMessage := fixtureUpdatedMsg{
		fixture:  models.Fixture{FixtureID: "fixture-1", GameWeekID: "gameweek-1", FixtureStatus: "FIXTURE-UPDATED"},
		gameWeek: &models.GameWeek{GameWeekID: "gameweek-1", Label: "fixture-merged"},
	}
	for i := 0; i < 1000; i++ {
		updated, _ := model.Update(fixtureMessage)
		model = updated.(Model)
	}
	if got := model.state.Fixtures[0].FixtureStatus; got != "FIXTURE-UPDATED" {
		t.Errorf("fixture merge status = %q, want FIXTURE-UPDATED", got)
	}
	if got := model.state.CurrentGameWeek.Label; got != "fixture-merged" {
		t.Errorf("fixture merge gameweek label = %q, want fixture-merged", got)
	}

	batchMessage := batchUpdatedMsg{
		fixtures: []models.Fixture{{FixtureID: "fixture-1", GameWeekID: "gameweek-1", FixtureStatus: "BATCH-UPDATED"}},
		gameWeek: &models.GameWeek{GameWeekID: "gameweek-1", Label: "batch-merged"},
	}
	for i := 0; i < 1000; i++ {
		updated, _ := model.Update(batchMessage)
		model = updated.(Model)
	}
	<-mutationsFinished

	if got := model.state.Fixtures[0].FixtureStatus; got != "BATCH-UPDATED" {
		t.Errorf("merged fixture status = %q, want BATCH-UPDATED", got)
	}
	if got := model.state.CurrentGameWeek.Label; got != "batch-merged" {
		t.Errorf("merged gameweek label = %q, want batch-merged", got)
	}
	if got := callerFixtures[0].FixtureStatus; got != "CALLER-COMPLETE" {
		t.Errorf("caller fixture status = %q, want CALLER-COMPLETE", got)
	}
	if got := callerGameWeek.Label; got != "caller-complete" {
		t.Errorf("caller gameweek label = %q, want caller-complete", got)
	}
}

func TestCreateDefaultTeamCmdSendsOneEligibleStarPlayer(t *testing.T) {
	t.Setenv(models.SEVEN_TUI_ALLOW_WRITES, "true")
	var received map[string]map[string]interface{}
	players := []map[string]interface{}{
		{"playerId": "forward-1", "position": "Forward", "teamId": "team-1", "ineligible": true},
		{"playerId": "forward-2", "position": "Forward", "teamId": "team-2"},
		{"playerId": "forward-3", "position": "Forward", "teamId": "team-3"},
		{"playerId": "forward-4", "position": "Forward", "teamId": "team-4"},
		{"playerId": "midfielder-1", "position": "Midfielder", "teamId": "team-5"},
		{"playerId": "midfielder-2", "position": "Midfielder", "teamId": "team-6"},
		{"playerId": "midfielder-3", "position": "Midfielder", "teamId": "team-7"},
		{"playerId": "midfielder-4", "position": "Midfielder", "teamId": "team-8"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/game-week/players":
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{"players": players})
		case request.Method == http.MethodPut && request.URL.Path == "/game-week/selections":
			if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
				t.Errorf("decode selections: %v", err)
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", request.Method, request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	message := createDefaultTeamCmd(aws.NewAPIClient(server.URL), "SE7-test")()
	result, ok := message.(teamCreatedMsg)
	if !ok {
		t.Fatalf("command message = %T, want teamCreatedMsg", message)
	}
	if result.err != nil {
		t.Fatalf("createDefaultTeamCmd() error = %v", result.err)
	}
	if len(received) != 7 {
		t.Fatalf("selection count = %d, want 7", len(received))
	}
	if received["player1"]["id"] != "forward-2" {
		t.Errorf("player1 id = %v, want available forward", received["player1"]["id"])
	}

	starPlayerCount := 0
	for _, selection := range received {
		if selection["starPlayer"] == true {
			starPlayerCount++
		}
	}
	if starPlayerCount != 1 {
		t.Errorf("starPlayer count = %d, want 1", starPlayerCount)
	}
}

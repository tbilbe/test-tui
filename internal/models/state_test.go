package models

import "testing"

func TestNewAppState(t *testing.T) {
	state := NewAppState()

	if state == nil {
		t.Fatal("NewAppState() returned nil")
	}
	if state.GameWeeks == nil {
		t.Error("GameWeeks should be initialized to empty slice, not nil")
	}
	if state.Fixtures == nil {
		t.Error("Fixtures should be initialized to empty slice, not nil")
	}
	if state.IsLoading {
		t.Error("IsLoading should default to false")
	}
}

func TestAppState_SetError(t *testing.T) {
	state := NewAppState()
	state.SuccessMessage = "Previous success"

	state.SetError("Something went wrong")

	if state.ErrorMessage != "Something went wrong" {
		t.Errorf("ErrorMessage = %q, want %q", state.ErrorMessage, "Something went wrong")
	}
	if state.SuccessMessage != "" {
		t.Errorf("SuccessMessage should be cleared, got %q", state.SuccessMessage)
	}
}

func TestAppState_SetSuccess(t *testing.T) {
	state := NewAppState()
	state.ErrorMessage = "Previous error"

	state.SetSuccess("Operation completed")

	if state.SuccessMessage != "Operation completed" {
		t.Errorf("SuccessMessage = %q, want %q", state.SuccessMessage, "Operation completed")
	}
	if state.ErrorMessage != "" {
		t.Errorf("ErrorMessage should be cleared, got %q", state.ErrorMessage)
	}
}

func TestAppState_ClearMessages(t *testing.T) {
	state := NewAppState()
	state.ErrorMessage = "Error"
	state.SuccessMessage = "Success"

	state.ClearMessages()

	if state.ErrorMessage != "" {
		t.Errorf("ErrorMessage should be cleared, got %q", state.ErrorMessage)
	}
	if state.SuccessMessage != "" {
		t.Errorf("SuccessMessage should be cleared, got %q", state.SuccessMessage)
	}
}

func TestAppState_SetLoading(t *testing.T) {
	state := NewAppState()

	state.SetLoading(true, "Loading data...")

	if !state.IsLoading {
		t.Error("IsLoading should be true")
	}
	if state.LoadingMessage != "Loading data..." {
		t.Errorf("LoadingMessage = %q, want %q", state.LoadingMessage, "Loading data...")
	}
}

func TestAppState_SetLoading_ClearsWhenDone(t *testing.T) {
	state := NewAppState()
	state.IsLoading = true
	state.LoadingMessage = "Loading..."

	state.SetLoading(false, "")

	if state.IsLoading {
		t.Error("IsLoading should be false")
	}
	if state.LoadingMessage != "" {
		t.Errorf("LoadingMessage should be cleared, got %q", state.LoadingMessage)
	}
}

func TestAppState_SetGameWeeks(t *testing.T) {
	state := NewAppState()
	gameWeeks := []GameWeek{
		{GameWeekID: "1", Label: "Week 1"},
		{GameWeekID: "2", Label: "Week 2"},
	}

	state.SetGameWeeks(gameWeeks)

	if len(state.GameWeeks) != 2 {
		t.Errorf("GameWeeks count = %d, want 2", len(state.GameWeeks))
	}
	if state.GameWeeks[0].GameWeekID != "1" {
		t.Errorf("GameWeeks[0].GameWeekID = %q, want %q", state.GameWeeks[0].GameWeekID, "1")
	}
}

func TestAppState_SetCurrentGameWeek(t *testing.T) {
	state := NewAppState()
	gw := &GameWeek{GameWeekID: "1", Label: "Week 1"}

	state.SetCurrentGameWeek(gw)

	if state.CurrentGameWeek == nil {
		t.Fatal("CurrentGameWeek should not be nil")
	}
	if state.CurrentGameWeek.GameWeekID != "1" {
		t.Errorf("CurrentGameWeek.GameWeekID = %q, want %q", state.CurrentGameWeek.GameWeekID, "1")
	}
}

func TestAppState_SetCurrentGameWeek_Nil(t *testing.T) {
	state := NewAppState()
	state.CurrentGameWeek = &GameWeek{GameWeekID: "1"}

	state.SetCurrentGameWeek(nil)

	if state.CurrentGameWeek != nil {
		t.Error("CurrentGameWeek should be nil")
	}
}

func TestAppState_SetFixtures(t *testing.T) {
	state := NewAppState()
	fixtures := []Fixture{
		{FixtureID: "f1"},
		{FixtureID: "f2"},
		{FixtureID: "f3"},
	}

	state.SetFixtures(fixtures)

	if len(state.Fixtures) != 3 {
		t.Errorf("Fixtures count = %d, want 3", len(state.Fixtures))
	}
}

func TestAppState_SetSelectedFixture(t *testing.T) {
	state := NewAppState()
	fixture := &Fixture{FixtureID: "f1"}

	state.SetSelectedFixture(fixture)

	if state.SelectedFixture == nil {
		t.Fatal("SelectedFixture should not be nil")
	}
	if state.SelectedFixture.FixtureID != "f1" {
		t.Errorf("SelectedFixture.FixtureID = %q, want %q", state.SelectedFixture.FixtureID, "f1")
	}
}

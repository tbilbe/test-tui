package models

type AppState struct {
	GameWeeks        []GameWeek
	CurrentGameWeek  *GameWeek
	Fixtures         []Fixture
	SelectedFixture  *Fixture
	IsAuthenticated  bool
	Username         string
	ErrorMessage     string
	SuccessMessage   string
	IsLoading        bool
	LoadingMessage   string
}

func NewAppState() *AppState {
	return &AppState{
		GameWeeks: []GameWeek{},
		Fixtures:  []Fixture{},
	}
}

func (s *AppState) SetGameWeeks(gameWeeks []GameWeek) {
	s.GameWeeks = gameWeeks
}

func (s *AppState) SetCurrentGameWeek(gw *GameWeek) {
	s.CurrentGameWeek = gw
}

func (s *AppState) SetFixtures(fixtures []Fixture) {
	s.Fixtures = fixtures
}

func (s *AppState) SetSelectedFixture(fixture *Fixture) {
	s.SelectedFixture = fixture
}

func (s *AppState) SetError(msg string) {
	s.ErrorMessage = msg
	s.SuccessMessage = ""
}

func (s *AppState) SetSuccess(msg string) {
	s.SuccessMessage = msg
	s.ErrorMessage = ""
}

func (s *AppState) ClearMessages() {
	s.ErrorMessage = ""
	s.SuccessMessage = ""
}

func (s *AppState) SetLoading(loading bool, message string) {
	s.IsLoading = loading
	s.LoadingMessage = message
}

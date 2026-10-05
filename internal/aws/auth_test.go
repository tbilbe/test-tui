package aws

import "testing"

func TestAccessTokenExpiredErrorExplainsHowToRefreshToken(t *testing.T) {
	want := "access token expired or rejected — copy a fresh token from the mobile app dev tools and re-export SEVEN_ACCESS_TOKEN"
	if ErrAccessTokenExpired.Error() != want {
		t.Errorf("ErrAccessTokenExpired = %q, want %q", ErrAccessTokenExpired, want)
	}
}

func TestNoCurrentGameWeekErrorRemainsDistinct(t *testing.T) {
	want := "no current game week found — seed an active game week and retry"
	if ErrNoCurrentGameWeek.Error() != want {
		t.Errorf("ErrNoCurrentGameWeek = %q, want %q", ErrNoCurrentGameWeek, want)
	}
}

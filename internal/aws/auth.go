package aws

import "errors"

var (
	ErrNoCurrentGameWeek  = errors.New("no current game week found — seed an active game week and retry")
	ErrAccessTokenExpired = errors.New("access token expired or rejected — copy a fresh token from the mobile app dev tools and re-export SEVEN_ACCESS_TOKEN")
)

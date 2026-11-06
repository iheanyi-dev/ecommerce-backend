package use_cases

import "time"

// timeNowUTC returns the current time in UTC.
func timeNowUTC() time.Time {
	return time.Now().UTC()
}

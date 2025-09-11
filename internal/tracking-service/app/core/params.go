package core

type TrackingParams struct {
	Port int
}

const (
	// in seconds for db response
	WaitTime = 20
)

var SleepTime = map[string]int{
	"dine_in":  8,
	"takeout":  10,
	"delivery": 12,
}

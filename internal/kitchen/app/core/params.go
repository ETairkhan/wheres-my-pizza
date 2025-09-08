package core

type WorkerParams struct{
	WorkerName string 
	OrderTypes []string
	HeartbeatInterval int 
	Prefetch int 
}

const (
	JobsSize = 100 

	// in seconds for db response 
	WaitTime = 20 

	DBReconnInterval = 5
	MBReconnInterval = 5
)

var (
	AllowedOrderTypes = map[string]bool{
		"dine_in": true,
		"takeout": true,
		"delivery":true,
	}

	SleepTime = map[string]int{
		"dine_in": 8,
		"takeout": 10, 
		"delivery": 12,
	}
)
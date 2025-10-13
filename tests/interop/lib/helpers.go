package lib

import (
	"time"
)

// GetTimestamp returns the current timestamp in RFC3339 format
func GetTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

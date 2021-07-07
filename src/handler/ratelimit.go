package handler

import (
	"net/http"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"time"
)

// RecentRequests stores some stats about recent requests
var RecentRequests map[string]int

// RecentRequestsPeriod stores the timestamp of the period to track when to reset it
var RecentRequestsPeriod int32

func init() {
	RecentRequests = make(map[string]int)
	RecentRequestsPeriod = 0
}

// RateLimited tries to handle RateLimited requests
func RateLimited(w http.ResponseWriter, r *http.Request) {
	logs.WriteError("IP address is rate limited.")
	r.URL.Path = config.RateLimitedScript

	Handle(w, r)

	logs.WriteError("IP address is rate limited.")

}

// IsRateLimited checks if the ipAddress has exceeded the limit
func IsRateLimited(ipAddress string, urlPath string) bool {

	if urlPath == config.RateLimitedScript[1:] {
		return false
	}

	return RecentRequests[ipAddress] > config.RateLimitRate

}

func addRequestRateLimit(ipAddress string) {

	timestamp := int32(time.Now().Round(time.Duration(config.RateLimitPeriod) * time.Second).Unix()) //Timestamp rounded to RateLimitPeriod seconds

	if timestamp != RecentRequestsPeriod {
		RecentRequests = make(map[string]int)
		RecentRequestsPeriod = timestamp
	}

	RecentRequests[ipAddress]++

}

package logs

import (
	"time"
)

// RequestStats stores some stats about recent requests
var RequestStats map[int32]int

func init() {
	RequestStats = make(map[int32]int)
}

func addRequest() {
	timestamp := int32(time.Now().Round(5 * time.Second).Unix()) //Timestamp rounded to 5 seconds
	RequestStats[timestamp]++

	if timestamp%300 == 0 { //Trim map after 6 minutes to avoid overflow
		RequestStats2 := make(map[int32]int)
		for timestamp2 := timestamp; timestamp-timestamp2 < 300; timestamp2 -= 5 {
			RequestStats2[timestamp2] = RequestStats[timestamp2]
		}
		RequestStats = RequestStats2
	}

}

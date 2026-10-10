package logs

import (
	"sync"
	"time"
)

// StatsInterval is the width of one Stats bucket
const StatsInterval = 5 * time.Second

// statsBuckets covers the last five minutes
const statsBuckets = 60

// Stats counts requests in StatsInterval buckets over the last five minutes.
// It is a ring: a bucket is reused, and reset, when its interval comes round again.
// The zero value is ready to use and safe for concurrent use.
type Stats struct {
	mu      sync.Mutex
	buckets [statsBuckets]bucket
}

type bucket struct {
	interval int64 // Unix time / StatsInterval
	count    int
}

// Add counts one request at time t
func (s *Stats) Add(t time.Time) {
	interval := t.Unix() / int64(StatsInterval/time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &s.buckets[interval%statsBuckets]
	if b.interval != interval {
		*b = bucket{interval: interval}
	}
	b.count++
}

// Recent returns the request counts of the last five minutes before now, newest first,
// with the start time of each bucket
func (s *Stats) Recent(now time.Time) (starts []time.Time, counts []int) {
	current := now.Unix() / int64(StatsInterval/time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	for interval := current; interval > current-statsBuckets; interval-- {
		count := 0
		if b := s.buckets[interval%statsBuckets]; b.interval == interval {
			count = b.count
		}
		starts = append(starts, time.Unix(interval*int64(StatsInterval/time.Second), 0))
		counts = append(counts, count)
	}
	return starts, counts
}

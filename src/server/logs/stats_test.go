package logs

import (
	"sync"
	"testing"
	"time"
)

func TestStats(t *testing.T) {
	var s Stats
	now := time.Unix(1_000_002, 0) // 2s into a bucket

	s.Add(now)
	s.Add(now.Add(-time.Second))
	s.Add(now.Add(-StatsInterval))
	s.Add(now.Add(-6 * time.Minute)) // too old to show, and its bucket is reused below
	s.Add(now.Add(-time.Minute))

	starts, counts := s.Recent(now)
	if len(starts) != 60 || len(counts) != 60 {
		t.Fatalf("got %d buckets, want 60", len(counts))
	}
	if starts[0].Unix() != 1_000_000 || starts[1].Unix() != 999_995 {
		t.Errorf("starts %v, %v", starts[0], starts[1])
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	if counts[0] != 2 || counts[1] != 1 || counts[12] != 1 || total != 4 {
		t.Errorf("counts %v", counts)
	}
}

func TestStatsConcurrent(t *testing.T) {
	var s Stats
	now := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add(now)
		}()
	}
	wg.Wait()
	if _, counts := s.Recent(now); counts[0] != 50 {
		t.Errorf("counted %d requests, want 50", counts[0])
	}
}

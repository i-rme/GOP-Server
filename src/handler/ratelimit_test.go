package handler

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(2, time.Minute)
	start := time.Unix(600, 0) // the start of a window

	// A client is limited once it made more than rate requests in the window
	for i, want := range []bool{true, true, true, false, false} {
		if got := l.allow("a", start.Add(time.Duration(i)*time.Second)); got != want {
			t.Errorf("request %d: allow = %v, want %v", i+1, got, want)
		}
	}
	if !l.allow("b", start.Add(10*time.Second)) {
		t.Error("another client was limited")
	}
	if !l.allow("a", start.Add(time.Minute)) {
		t.Error("the limit was not reset in the next window")
	}
}

package cache

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetBuildsOnce(t *testing.T) {
	c := New(t.TempDir())
	key := Key("package main")
	var builds atomic.Int32
	build := func(dir, binary string) error {
		builds.Add(1)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(binary, nil, 0o755)
	}

	var wg sync.WaitGroup
	hits := make([]bool, 20)
	for i := range hits {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, hit, err := c.Get(key, build)
			if err != nil {
				t.Error(err)
			}
			hits[i] = hit
		}()
	}
	wg.Wait()
	if builds.Load() != 1 {
		t.Errorf("built %d times, want once", builds.Load())
	}

	binary, hit, err := c.Get(key, build)
	if err != nil || !hit || builds.Load() != 1 {
		t.Errorf("after the build: %s, hit %v, err %v, %d builds", binary, hit, err, builds.Load())
	}
}

func TestFailedBuildIsRetried(t *testing.T) {
	c := New(t.TempDir())
	failure := errors.New("compile error")
	calls := 0
	build := func(dir, binary string) error { calls++; return failure }

	for i := 0; i < 2; i++ {
		if _, hit, err := c.Get(Key("broken"), build); err != failure || hit {
			t.Errorf("hit %v, err %v", hit, err)
		}
	}
	if calls != 2 {
		t.Errorf("build called %d times, want 2", calls)
	}
}

func TestKeyDependsOnSource(t *testing.T) {
	if Key("a") == Key("b") || Key("a") != Key("a") || len(Key("a")) != 64 {
		t.Error("keys are not a hash of the source")
	}
}

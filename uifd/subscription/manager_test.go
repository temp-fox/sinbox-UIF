package subscription

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func waitForState(t *testing.T, m *Manager, id string, state JobState) *Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job := m.Get(id)
		if job != nil && job.State == state {
			return job
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("job %s did not reach %s", id, state)
	return nil
}

func TestManagerPreventsReentryAndPersistsTerminalStatus(t *testing.T) {
	m := NewManager(WithMaxConcurrent(1), WithRetention(time.Hour))
	started := make(chan struct{})
	release := make(chan struct{})
	fn := func(ctx context.Context) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	first := m.Start(context.Background(), "sub-1", "refresh", fn)
	<-started
	second := m.Start(context.Background(), "sub-1", "refresh", fn)
	if first.ID != second.ID {
		t.Fatalf("duplicate start returned %q and %q", first.ID, second.ID)
	}
	close(release)
	job := waitForState(t, m, first.ID, Success)
	if job.FinishedAt.IsZero() {
		t.Fatal("terminal status was not persisted")
	}
	if m.Start(context.Background(), "sub-1", "refresh", fn).ID == first.ID {
		t.Fatal("completed job was incorrectly treated as active")
	}
}

func TestManagerCancel(t *testing.T) {
	m := NewManager()
	started := make(chan struct{})
	job := m.Start(context.Background(), "sub-cancel", "refresh", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	if !m.Cancel(job.ID) {
		t.Fatal("Cancel returned false for running job")
	}
	result := waitForState(t, m, job.ID, Cancelled)
	if result.Error == "" {
		t.Fatal("cancelled job has no error")
	}
	if m.Cancel(job.ID) {
		t.Fatal("Cancel returned true for terminal job")
	}
}

func TestManagerConcurrentLimit(t *testing.T) {
	m := NewManager(WithMaxConcurrent(2))
	var running int32
	var maximum int32
	release := make(chan struct{})
	fn := func(ctx context.Context) error {
		current := atomic.AddInt32(&running, 1)
		for {
			old := atomic.LoadInt32(&maximum)
			if current <= old || atomic.CompareAndSwapInt32(&maximum, old, current) {
				break
			}
		}
		<-release
		atomic.AddInt32(&running, -1)
		return nil
	}
	jobs := make([]*Job, 0, 5)
	for i := 0; i < 5; i++ {
		jobs = append(jobs, m.Start(context.Background(), "sub-"+string(rune('a'+i)), "refresh", fn))
	}
	time.Sleep(30 * time.Millisecond)
	if got := atomic.LoadInt32(&maximum); got > 2 {
		t.Fatalf("maximum concurrent callbacks = %d, want <= 2", got)
	}
	close(release)
	for _, job := range jobs {
		waitForState(t, m, job.ID, Success)
	}
}

func TestManagerFailureAndCallback(t *testing.T) {
	want := errors.New("parse failed")
	m := NewManager(WithCallback(func(context.Context) error { return want }))
	job := m.Start(context.Background(), "sub-fail", "parse", nil)
	result := waitForState(t, m, job.ID, Failed)
	if result.Error != want.Error() {
		t.Fatalf("error = %q, want %q", result.Error, want)
	}
}

func TestManagerCleanup(t *testing.T) {
	m := NewManager(WithRetention(time.Second))
	job := m.Start(context.Background(), "sub-clean", "refresh", func(context.Context) error { return nil })
	waitForState(t, m, job.ID, Success)
	m.mu.Lock()
	m.jobs[job.ID].FinishedAt = time.Now().Add(-2 * time.Second)
	m.mu.Unlock()
	if removed := m.Cleanup(); removed != 1 {
		t.Fatalf("Cleanup removed %d jobs, want 1", removed)
	}
	if m.Get(job.ID) != nil {
		t.Fatal("cleaned job is still available")
	}
}

func TestManagerClose(t *testing.T) {
	m := NewManager()
	started := make(chan struct{})
	job := m.Start(context.Background(), "sub-close", "refresh", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	m.Close()
	waitForState(t, m, job.ID, Cancelled)
}

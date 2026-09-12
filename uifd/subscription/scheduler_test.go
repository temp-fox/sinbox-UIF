package subscription

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func schedulerSpec(id string, interval time.Duration, startup bool) SubscriptionSpec {
	return SubscriptionSpec{
		ID:       id,
		Snapshot: NewSnapshot(nil),
		Policy: SchedulePolicy{
			UpdateEnabled:  true,
			UpdateInterval: interval,
			StartupUpdate:  startup,
		},
	}
}

func waitScheduler(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("scheduler condition was not reached")
}

func TestSchedulerStartStopReloadAndNonReentry(t *testing.T) {
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	var running int32
	var maximum int32
	executor := func(ctx context.Context, spec SubscriptionSpec) error {
		current := atomic.AddInt32(&running, 1)
		for {
			old := atomic.LoadInt32(&maximum)
			if current <= old || atomic.CompareAndSwapInt32(&maximum, old, current) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
			atomic.AddInt32(&running, -1)
			return nil
		case <-ctx.Done():
			atomic.AddInt32(&running, -1)
			return ctx.Err()
		}
	}
	s := NewScheduler(executor)
	if err := s.Reload([]SubscriptionSpec{schedulerSpec("one", 5*time.Millisecond, true)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	<-started
	// The short interval must not re-enter the still-running execution.
	time.Sleep(25 * time.Millisecond)
	if got := atomic.LoadInt32(&maximum); got != 1 {
		t.Fatalf("maximum concurrent executions = %d, want 1", got)
	}
	if task := s.GetTask("one"); task == nil || task.State != Running {
		t.Fatalf("task state = %#v, want running", task)
	}
	close(release)
	waitScheduler(t, time.Second, func() bool {
		task := s.GetTask("one")
		return task != nil && task.State == Success
	})

	if err := s.Reload([]SubscriptionSpec{schedulerSpec("two", time.Hour, true)}); err != nil {
		t.Fatal(err)
	}
	if task := s.GetTask("one"); task == nil || task.State != Cancelled {
		t.Fatalf("removed task = %#v, want cancelled", task)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerStopCancelsExecutorAndTracksFailure(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	s := NewScheduler(func(ctx context.Context, _ SubscriptionSpec) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	})
	if err := s.Reload([]SubscriptionSpec{schedulerSpec("cancel", time.Hour, true)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("executor was not cancelled by Stop")
	}
	waitScheduler(t, time.Second, func() bool {
		task := s.GetTask("cancel")
		return task != nil && task.State == Cancelled
	})
}

func TestSchedulerStartupJitterIsStableAndReloadValidates(t *testing.T) {
	var mu sync.Mutex
	starts := make(map[string]time.Time)
	s := NewScheduler(func(_ context.Context, spec SubscriptionSpec) error {
		mu.Lock()
		starts[spec.ID] = time.Now()
		mu.Unlock()
		return nil
	}, WithStartupJitter(40*time.Millisecond))
	if err := s.Reload([]SubscriptionSpec{
		schedulerSpec("alpha", time.Hour, true),
		schedulerSpec("beta", time.Hour, true),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	waitScheduler(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(starts) == 2
	})
	mu.Lock()
	first, second := starts["alpha"], starts["beta"]
	mu.Unlock()
	if first.Equal(second) {
		t.Fatal("startup executions were not staggered")
	}
	if err := s.Reload([]SubscriptionSpec{{ID: "dup"}, {ID: "dup"}}); err == nil {
		t.Fatal("duplicate IDs were accepted")
	}
	if err := s.Reload([]SubscriptionSpec{{ID: ""}}); err == nil {
		t.Fatal("empty ID was accepted")
	}
	_ = s.Stop()
}

func TestSchedulerStartRequiresExecutor(t *testing.T) {
	s := NewScheduler(nil)
	if err := s.Start(); err == nil {
		t.Fatal("nil executor was accepted")
	}
}

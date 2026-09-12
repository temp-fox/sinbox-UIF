package subscription

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/uif/uifd/subscription/parser"
)

type fakeProbeExecutor struct {
	mu        sync.Mutex
	results   map[string]ProbeResult
	delay     time.Duration
	running   int32
	maximum   int32
	started   int32
	cancelled int32
}

func (f *fakeProbeExecutor) Probe(ctx context.Context, target ProbeTarget) (ProbeResult, error) {
	current := atomic.AddInt32(&f.running, 1)
	atomic.AddInt32(&f.started, 1)
	for {
		old := atomic.LoadInt32(&f.maximum)
		if current <= old || atomic.CompareAndSwapInt32(&f.maximum, old, current) {
			break
		}
	}
	defer atomic.AddInt32(&f.running, -1)
	if f.delay > 0 {
		timer := time.NewTimer(f.delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			atomic.AddInt32(&f.cancelled, 1)
			return ProbeResult{}, ctx.Err()
		}
	}
	f.mu.Lock()
	result, ok := f.results[target.ID]
	f.mu.Unlock()
	if !ok {
		return ProbeResult{}, errors.New("missing fake result")
	}
	return result, nil
}

func TestProbePoolBoundsConcurrencyAndAppliesThreshold(t *testing.T) {
	fake := &fakeProbeExecutor{delay: 5 * time.Millisecond, results: map[string]ProbeResult{
		"fast": {Success: true, DelayMs: 20},
		"slow": {Success: true, DelayMs: 150},
		"fail": {Success: false, Error: "refused"},
	}}
	pool := NewProbePool(fake, ProbeOptions{Workers: 2, MaxDelayMs: 100})
	targets := []ProbeTarget{{Index: 0, ID: "fast"}, {Index: 1, ID: "slow"}, {Index: 2, ID: "fail"}}
	results, err := pool.Run(context.Background(), targets)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(targets) {
		t.Fatalf("results = %d, want %d", len(results), len(targets))
	}
	if got := atomic.LoadInt32(&fake.maximum); got > 2 {
		t.Fatalf("maximum concurrency = %d, want at most 2", got)
	}
	if !results[0].Success || results[0].DelayMs != 20 {
		t.Fatalf("fast result = %#v", results[0])
	}
	if results[1].Success || results[1].Error == "" {
		t.Fatalf("threshold result = %#v", results[1])
	}
	if results[2].Success || results[2].Error != "refused" {
		t.Fatalf("failure result = %#v", results[2])
	}
}

func TestProbePoolCancellationCancelsFakeExecutor(t *testing.T) {
	fake := &fakeProbeExecutor{delay: time.Second, results: map[string]ProbeResult{"one": {Success: true, DelayMs: 1}}}
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	wrapped := ProbeExecutorFunc(func(ctx context.Context, target ProbeTarget) (ProbeResult, error) {
		close(started)
		return fake.Probe(ctx, target)
	})
	pool := NewProbePool(wrapped, ProbeOptions{Workers: 1})
	finished := make(chan struct{})
	var results []ProbeResult
	var err error
	go func() {
		results, err = pool.Run(ctx, []ProbeTarget{{Index: 0, ID: "one"}})
		close(finished)
	}()
	<-started
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("probe pool did not stop after cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("cancelled results = %#v", results)
	}
	if atomic.LoadInt32(&fake.cancelled) != 1 {
		t.Fatal("fake executor did not observe cancellation")
	}
}

func TestProbeSnapshotAppliesFailureIsolationAndMinimumKeep(t *testing.T) {
	snapshot := NewSnapshot([]parser.Node{
		testParserNode("one", "one.example"),
		testParserNode("two", "two.example"),
		testParserNode("three", "three.example"),
	})
	for i := range snapshot.Nodes {
		snapshot.Nodes[i].Enabled = true
	}
	fake := &fakeProbeExecutor{results: map[string]ProbeResult{
		snapshot.Nodes[0].ID: {Success: false, Error: "timeout"},
		snapshot.Nodes[1].ID: {Success: true, DelayMs: 30},
		snapshot.Nodes[2].ID: {Success: false, Error: "timeout"},
	}}
	results, err := ProbeSnapshot(context.Background(), &snapshot, fake, ProbeOptions{
		Workers: 2, FailureAction: "delete", MaxConsecutiveFailures: 1, MinKeep: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("result count = %d", len(results))
	}
	if snapshot.Nodes[0].Enabled || !snapshot.Nodes[0].Quarantined || snapshot.Nodes[0].ConsecutiveFailures != 1 {
		t.Fatalf("first failed node state = %#v", snapshot.Nodes[0])
	}
	if !snapshot.Nodes[1].Enabled || snapshot.Nodes[1].LastProbeStatus != "healthy" || snapshot.Nodes[1].LastProbeDelayMs != 30 {
		t.Fatalf("healthy node state = %#v", snapshot.Nodes[1])
	}
	if snapshot.Nodes[2].Enabled || !snapshot.Nodes[2].Quarantined {
		t.Fatalf("second failed node should be isolated while preserving one healthy node: %#v", snapshot.Nodes[2])
	}
	if snapshot.UpdatedAt.IsZero() {
		t.Fatal("snapshot updated time was not set")
	}
}

func TestTargetsForSnapshotSkipsDisabledAndQuarantined(t *testing.T) {
	snapshot := NewSnapshot([]parser.Node{testParserNode("one", "one.example"), testParserNode("two", "two.example")})
	snapshot.Nodes[0].Enabled = false
	snapshot.Nodes[1].Quarantined = true
	if got := TargetsForSnapshot(snapshot); len(got) != 0 {
		t.Fatalf("targets = %#v, want none", got)
	}
}

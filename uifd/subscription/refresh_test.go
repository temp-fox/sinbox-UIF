package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRefreshHTTPServerUpdatesSnapshotAndKeepsItOnFailure(t *testing.T) {
	var body atomic.Value
	body.Store("trojan://secret@example.com:443#one")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if strings.Contains(body.Load().(string), "invalid") {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(body.Load().(string)))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "snapshot.json")
	spec := SubscriptionSpec{ID: "sub", URL: server.URL, SnapshotPath: path}
	fetch := func(ctx context.Context, source string) (string, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return "", "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", "", err
		}
		if resp.StatusCode/100 != 2 {
			return "", "", os.ErrInvalid
		}
		return string(data), "", nil
	}
	first, err := Refresh(context.Background(), spec, fetch)
	if err != nil || first.SnapshotSummary.Added != 1 {
		t.Fatalf("first refresh: %#v, %v", first, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body.Store("invalid")
	if _, err := Refresh(context.Background(), spec, fetch); err == nil {
		t.Fatal("invalid refresh succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed refresh replaced snapshot")
	}
	body.Store("trojan://secret@example.net:443#two")
	second, err := Refresh(context.Background(), spec, fetch)
	if err != nil || second.SnapshotSummary.Added != 1 {
		t.Fatalf("second refresh: %#v, %v", second, err)
	}
}

func TestRefreshRejectsEnabledProbeWithoutExecutorBeforeSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-executor.json")
	_, err := Refresh(context.Background(), SubscriptionSpec{
		ID: "missing-executor", URL: "memory://subscription", SnapshotPath: path,
		Probe: ProbeConfig{Enabled: true},
	}, func(context.Context, string) (string, string, error) {
		return "trojan://secret@example.com:443#one", "", nil
	})
	if err == nil || !strings.Contains(err.Error(), "probe executor") {
		t.Fatalf("refresh error = %v, want missing executor", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("snapshot exists after rejected refresh: %v", statErr)
	}
}
func TestRefreshRunsProbeBeforeAtomicSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probed.json")
	var seen []ProbeTarget
	executor := ProbeExecutorFunc(func(_ context.Context, target ProbeTarget) (ProbeResult, error) {
		seen = append(seen, target)
		if target.Address == "slow.example" {
			return ProbeResult{Success: true, DelayMs: 250}, nil
		}
		return ProbeResult{Success: true, DelayMs: 20}, nil
	})
	spec := SubscriptionSpec{
		ID: "probed", URL: "memory://subscription", SnapshotPath: path,
		Probe: ProbeConfig{Enabled: true, DefaultURL: "https://probe.invalid/204", Executor: executor, ThresholdMs: 100, MaxConsecutiveFailures: 1, FailureAction: "quarantine", MinKeep: 1},
	}
	fetch := func(context.Context, string) (string, string, error) {
		return "trojan://secret@fast.example:443#fast\ntrojan://secret@slow.example:443#slow", "", nil
	}
	if _, err := Refresh(context.Background(), spec, fetch); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[0].URL != "https://probe.invalid/204" || seen[0].Address == "" {
		t.Fatalf("probe targets = %#v", seen)
	}
	if snapshot.Nodes[0].LastProbeStatus != "healthy" || snapshot.Nodes[0].LastProbeDelayMs != 20 {
		t.Fatalf("healthy node = %#v", snapshot.Nodes[0])
	}
	if snapshot.Nodes[1].LastProbeStatus != "failed" || !snapshot.Nodes[1].Quarantined {
		t.Fatalf("slow node = %#v", snapshot.Nodes[1])
	}
}

func TestSchedulerRunsRealRefreshPeriodically(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("trojan://secret@example.com:443#scheduled"))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "scheduled.json")
	fetch := func(ctx context.Context, source string) (string, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return "", "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", "", err
		}
		if resp.StatusCode/100 != 2 {
			return "", "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}
		return string(data), "", nil
	}
	s := NewScheduler(func(ctx context.Context, spec SubscriptionSpec) error {
		_, err := Refresh(ctx, spec, fetch)
		return err
	})
	if err := s.Reload([]SubscriptionSpec{{
		ID: "real-periodic", URL: server.URL, SnapshotPath: path,
		Policy: SchedulePolicy{UpdateEnabled: true, UpdateInterval: time.Millisecond, StartupUpdate: true},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("scheduler did not publish a real refresh snapshot")
}

package subscription

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ProbeTarget identifies one snapshot node to probe. Index is the position in
// Snapshot.Nodes; ID is retained as a fallback when a snapshot is reordered.
type ProbeTarget struct {
	Index   int          `json:"index"`
	ID      string       `json:"id,omitempty"`
	URL     string       `json:"url,omitempty"`
	Address string       `json:"address,omitempty"`
	Port    int          `json:"port,omitempty"`
	Node    SnapshotNode `json:"node"`
}

const DefaultProbeURL = "https://www.gstatic.com/generate_204"

// ProbeConfig enables probing for a refresh and carries its optional targets.
// Targets are normally generated from the refreshed snapshot; explicit targets
// are useful for tests and callers that need a custom endpoint.
type ProbeConfig struct {
	Enabled                bool                 `json:"enabled,omitempty"`
	DefaultURL             string               `json:"default_url,omitempty"`
	TimeoutMs              int                  `json:"timeout_ms,omitempty"`
	Concurrency            int                  `json:"concurrency,omitempty"`
	ThresholdMs            int                  `json:"threshold_ms,omitempty"`
	MaxConsecutiveFailures int                  `json:"max_consecutive_failures,omitempty"`
	FailureAction          string               `json:"failure_action,omitempty"`
	MinKeep                int                  `json:"min_keep,omitempty"`
	Options                ProbeOptions         `json:"options,omitempty"`
	Targets                []ProbeTarget        `json:"targets,omitempty"`
	TargetResolver         *ProbeTargetResolver `json:"-"`
	SubscriptionID         string               `json:"-"`
	Executor               ProbeExecutor        `json:"-"`
	CorePath               string               `json:"core_path,omitempty"`
}

func (c ProbeConfig) options() ProbeOptions {
	o := c.Options
	if c.TimeoutMs > 0 {
		o.TimeoutMs = c.TimeoutMs
	}
	if c.Concurrency > 0 {
		o.Concurrency = c.Concurrency
	}
	if c.ThresholdMs > 0 {
		o.MaxDelayMs = c.ThresholdMs
	}
	if c.MaxConsecutiveFailures > 0 {
		o.MaxConsecutiveFailures = c.MaxConsecutiveFailures
	}
	if c.FailureAction != "" {
		o.FailureAction = c.FailureAction
	}
	if c.MinKeep > 0 {
		o.MinKeep = c.MinKeep
	}
	if o.Timeout <= 0 && o.TimeoutMs > 0 {
		o.Timeout = time.Duration(o.TimeoutMs) * time.Millisecond
	}
	return o
}

func (c ProbeConfig) targets(snapshot Snapshot) []ProbeTarget {
	if len(c.Targets) > 0 {
		return append([]ProbeTarget(nil), c.Targets...)
	}
	probeURL := c.DefaultURL
	if probeURL == "" {
		probeURL = DefaultProbeURL
	}
	if c.TargetResolver != nil {
		resolver := *c.TargetResolver
		resolver.DefaultURL = probeURL
		targets := resolver.ResolveTargets(snapshot, c.SubscriptionID)
		if len(targets) > 0 {
			return targets
		}
	}
	return TargetsForSnapshotWithURL(snapshot, probeURL)
}

// ProbeExecutor performs one node probe. Implementations must honor ctx
// cancellation and must not mutate the target.
type ProbeExecutor interface {
	Probe(context.Context, ProbeTarget) (ProbeResult, error)
}

// ProbeExecutorValidator lets refresh reject an unavailable runtime before
// changing the snapshot. Individual node failures remain health results.
type ProbeExecutorValidator interface {
	Validate() error
}

func validateProbeExecutor(executor ProbeExecutor) error {
	if executor == nil {
		return errors.New("subscription probe executor is nil")
	}
	if validator, ok := executor.(ProbeExecutorValidator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ProbeExecutorFunc adapts a function into a ProbeExecutor.
type ProbeExecutorFunc func(context.Context, ProbeTarget) (ProbeResult, error)

func (f ProbeExecutorFunc) Probe(ctx context.Context, target ProbeTarget) (ProbeResult, error) {
	return f(ctx, target)
}

// UnsupportedProbeExecutor is the safe service default. A URL-level HTTP
// request cannot prove that a particular subscription node works, so the
// scheduler must not mark every node healthy without an injected executor.
type UnsupportedProbeExecutor struct{}

func (UnsupportedProbeExecutor) Probe(context.Context, ProbeTarget) (ProbeResult, error) {
	return ProbeResult{Success: false, Error: "unsupported: per-node proxy probing requires an injected ProbeExecutor"}, errors.New("unsupported: per-node proxy probing requires an injected ProbeExecutor")
}

// HTTPURLProbeExecutor performs only an endpoint-level HTTP probe. It is
// intentionally not used as the service default because a successful request
// does not establish per-node proxy health. Callers may inject it only when
// their integration treats the target URL as the health subject.
type HTTPURLProbeExecutor struct {
	Client *http.Client
	URL    string
}

func NewHTTPURLProbeExecutor(client *http.Client, endpoint string) *HTTPURLProbeExecutor {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPURLProbeExecutor{Client: client, URL: endpoint}
}

func (p *HTTPURLProbeExecutor) Probe(ctx context.Context, target ProbeTarget) (ProbeResult, error) {
	if p == nil || p.Client == nil {
		return ProbeResult{Error: "unsupported: HTTP URL probe is not configured"}, errors.New("unsupported: HTTP URL probe is not configured")
	}
	endpoint := strings.TrimSpace(p.URL)
	if endpoint == "" {
		endpoint = strings.TrimSpace(target.URL)
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return ProbeResult{Error: "unsupported: probe endpoint must be an http(s) URL without credentials"}, errors.New("unsupported: invalid HTTP probe endpoint")
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return ProbeResult{Error: "unsupported: invalid HTTP probe endpoint"}, err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return ProbeResult{Error: "HTTP URL probe failed: " + err.Error()}, err
	}
	resp.Body.Close()
	delay := int(time.Since(start) / time.Millisecond)
	if delay < 1 {
		delay = 1
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("HTTP URL probe returned %s", resp.Status)
		return ProbeResult{DelayMs: delay, Error: err.Error()}, err
	}
	return ProbeResult{Success: true, DelayMs: delay}, nil
}

// ProbeOptions controls the worker pool and the safety policy used when
// applying results to a snapshot.
type ProbeOptions struct {
	Workers                int           `json:"workers,omitempty"`
	Concurrency            int           `json:"concurrency,omitempty"`
	Timeout                time.Duration `json:"-"`
	TimeoutMs              int           `json:"timeout_ms,omitempty"`
	MaxDelayMs             int           `json:"max_delay_ms,omitempty"`
	DelayThresholdMs       int           `json:"threshold_ms,omitempty"`
	MaxConsecutiveFailures int           `json:"max_consecutive_failures,omitempty"`
	FailureAction          string        `json:"failure_action,omitempty"`
	MinKeep                int           `json:"min_keep,omitempty"`
}

func (o ProbeOptions) workers() int {
	if o.Workers > 0 {
		return o.Workers
	}
	if o.Concurrency > 0 {
		return o.Concurrency
	}
	return 1
}

func (o ProbeOptions) maxDelayMs() int {
	if o.MaxDelayMs > 0 {
		return o.MaxDelayMs
	}
	return o.DelayThresholdMs
}

func (o ProbeOptions) mergeOptions() MergeOptions {
	return normalizeOptions(MergeOptions{
		FailureAction:          o.FailureAction,
		MinKeep:                o.MinKeep,
		MaxConsecutiveFailures: o.MaxConsecutiveFailures,
	})
}

// ProbePool is a bounded-concurrency probe worker pool.
type ProbePool struct {
	executor ProbeExecutor
	options  ProbeOptions
}

// ProbeWorkerPool is an explicit alias for callers that prefer the worker-pool
// name in their integration code.
type ProbeWorkerPool = ProbePool

func NewProbePool(executor ProbeExecutor, options ProbeOptions) *ProbePool {
	return &ProbePool{executor: executor, options: options}
}

func NewProbeWorkerPool(executor ProbeExecutor, options ProbeOptions) *ProbePool {
	return NewProbePool(executor, options)
}

// Run probes targets with bounded concurrency. Results retain target order.
// Cancellation stops queued work and is returned after already-started probes
// have observed cancellation. A partial result set is returned with the error.
func (p *ProbePool) Run(ctx context.Context, targets []ProbeTarget) ([]ProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p == nil || p.executor == nil {
		return nil, errors.New("subscription probe executor is nil")
	}
	if len(targets) == 0 {
		return []ProbeResult{}, nil
	}

	type indexedTarget struct {
		position int
		target   ProbeTarget
	}
	jobs := make(chan indexedTarget)
	results := make(chan indexedResult, len(targets))
	workers := p.options.workers()
	if workers > len(targets) {
		workers = len(targets)
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					result := p.runOne(ctx, job.target)
					result.TargetIndex = job.target.Index
					result.TargetID = job.target.ID
					results <- indexedResult{position: job.position, result: result}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for position, target := range targets {
			select {
			case jobs <- indexedTarget{position: position, target: target}:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]*ProbeResult, len(targets))
	for item := range results {
		result := item.result
		ordered[item.position] = &result
	}
	out := make([]ProbeResult, 0, len(targets))
	for _, result := range ordered {
		if result != nil {
			out = append(out, *result)
		}
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	return out, nil
}

type indexedResult struct {
	position int
	result   ProbeResult
}

func (p *ProbePool) runOne(parent context.Context, target ProbeTarget) ProbeResult {
	ctx := parent
	cancel := func() {}
	if p.options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, p.options.Timeout)
	}
	defer cancel()

	result, err := p.executor.Probe(ctx, target)
	if err != nil {
		result.Success = false
		if result.Error == "" {
			result.Error = err.Error()
		}
	}
	if !result.Success || result.DelayMs <= 0 {
		result.Success = false
		if result.Error == "" {
			result.Error = "probe failed"
		}
		return result
	}
	if max := p.options.maxDelayMs(); max > 0 && result.DelayMs > max {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("probe delay %dms exceeds threshold %dms", result.DelayMs, max)
		}
	}
	return result
}

// TargetsForSnapshot creates stable probe targets for enabled, non-quarantined
// nodes. Disabled or quarantined nodes are intentionally not re-enabled by a
// health check run.
func TargetsForSnapshot(snapshot Snapshot) []ProbeTarget {
	return TargetsForSnapshotWithURL(snapshot, "")
}

// TargetsForSnapshotWithURL creates targets using the default URL and the
// node's parsed address. The executor decides which endpoint to use; no
// sing-box process is required by the subscription package.
func TargetsForSnapshotWithURL(snapshot Snapshot, defaultURL string) []ProbeTarget {
	targets := make([]ProbeTarget, 0, len(snapshot.Nodes))
	for index, node := range snapshot.Nodes {
		if !node.Enabled || node.Quarantined {
			continue
		}
		targets = append(targets, ProbeTarget{
			Index: index, ID: node.ID, URL: defaultURL,
			Address: node.Transport.Address, Port: node.Transport.Port, Node: node,
		})
	}
	return targets
}

// ApplyProbeResults applies pool results to snapshot node health state. It
// matches by index first and by stable ID when the target index is stale.
func ApplyProbeResults(snapshot *Snapshot, results []ProbeResult, options ProbeOptions) int {
	if snapshot == nil {
		return 0
	}
	merge := options.mergeOptions()
	applied := 0
	for _, result := range results {
		index := result.TargetIndex
		if index < 0 || index >= len(snapshot.Nodes) || (result.TargetID != "" && snapshot.Nodes[index].ID != result.TargetID) {
			index = -1
			if result.TargetID != "" {
				for i := range snapshot.Nodes {
					if snapshot.Nodes[i].ID == result.TargetID {
						index = i
						break
					}
				}
			}
		}
		if index < 0 || index >= len(snapshot.Nodes) {
			continue
		}
		ApplyProbeResult(snapshot.Nodes, index, result, merge)
		applied++
	}
	if applied > 0 {
		snapshot.UpdatedAt = time.Now()
	}
	return applied
}

func SummarizeProbes(total int, results []ProbeResult, err error) *ProbeSummary {
	summary := &ProbeSummary{Total: total, Completed: len(results)}
	for _, result := range results {
		if result.Success && result.DelayMs > 0 {
			summary.Healthy++
		} else {
			summary.Failed++
		}
	}
	if err != nil {
		summary.Status = "cancelled"
	} else if summary.Failed > 0 {
		summary.Status = "degraded"
	} else if total == 0 {
		summary.Status = "skipped"
	} else {
		summary.Status = "healthy"
	}
	return summary
}

// ProbeSnapshot probes the currently eligible nodes and applies any completed
// results, including partial results returned on cancellation.
func ProbeSnapshot(ctx context.Context, snapshot *Snapshot, executor ProbeExecutor, options ProbeOptions) ([]ProbeResult, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}
	return ProbeSnapshotWithTargets(ctx, snapshot, executor, TargetsForSnapshot(*snapshot), options)
}

// ProbeSnapshotWithTargets runs the worker pool and applies all completed
// results. It is used by Refresh so targets are derived from the newly merged
// snapshot before the single atomic save.
func ProbeSnapshotWithTargets(ctx context.Context, snapshot *Snapshot, executor ProbeExecutor, targets []ProbeTarget, options ProbeOptions) ([]ProbeResult, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}
	results, err := NewProbePool(executor, options).Run(ctx, targets)
	ApplyProbeResults(snapshot, results, options)
	return results, err
}

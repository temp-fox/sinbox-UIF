package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/uif/uifd/subscription/parser"
)

// FetchFunc retrieves a subscription body and its response metadata.
type FetchFunc func(context.Context, string) (body, extraInfo string, err error)

// RefreshResult contains the data returned by one successful subscription
// refresh. Body and ExtraInfo preserve the legacy job response contract.
type RefreshResult struct {
	Body            string          `json:"body"`
	ExtraInfo       string          `json:"extra_info,omitempty"`
	Nodes           []parser.Node    `json:"nodes"`
	ParseSummary    ParseSummary    `json:"parse_summary"`
	SnapshotSummary SnapshotSummary `json:"snapshot_summary"`
	ProbeSummary    *ProbeSummary   `json:"probe_summary,omitempty"`
}

// Refresh performs one real fetch, parse, merge, and atomic snapshot publish.
// It does not modify the existing snapshot unless every preceding step
// succeeds, so a network or parser failure leaves the old file untouched.
func Refresh(ctx context.Context, spec SubscriptionSpec, fetch FetchFunc) (RefreshResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if fetch == nil {
		return RefreshResult{}, errors.New("subscription refresh fetcher is nil")
	}
	id := strings.TrimSpace(spec.ID)
	if id == "" {
		return RefreshResult{}, errors.New("subscription id is empty")
	}
	source := strings.TrimSpace(spec.URL)
	if source == "" {
		source = strings.TrimSpace(spec.Source)
	}
	if source == "" {
		return RefreshResult{}, errors.New("subscription source is empty")
	}
	if strings.TrimSpace(spec.SnapshotPath) == "" {
		return RefreshResult{}, errors.New("subscription snapshot path is empty")
	}
	body, extraInfo, err := fetch(ctx, source)
	if err != nil {
		return RefreshResult{}, err
	}
	if strings.TrimSpace(body) == "" {
		return RefreshResult{}, errors.New("subscription response body is empty")
	}
	parsed, err := parser.Parse(body)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("parse subscription: %w", err)
	}
	if len(parsed.Nodes) == 0 {
		return RefreshResult{}, errors.New("subscription contains no nodes")
	}
	if spec.Probe.Enabled {
		if err := validateProbeExecutor(spec.Probe.Executor); err != nil {
			return RefreshResult{}, err
		}
	}
	old := spec.Snapshot
	loaded, err := Load(spec.SnapshotPath)
	if err == nil {
		old = loaded
	} else if !os.IsNotExist(err) {
		return RefreshResult{}, fmt.Errorf("load subscription snapshot: %w", err)
	}
	merged, err := ApplyParse(old, parsed, nil, mergeOptions(spec.Policy))
	if err != nil {
		return RefreshResult{}, err
	}
	var probeSummary *ProbeSummary
	if spec.Probe.Enabled && spec.Probe.Executor != nil {
		targets := spec.Probe.targets(merged)
		if len(spec.ProbeTargets) > 0 {
			targets = append([]ProbeTarget(nil), spec.ProbeTargets...)
		}
		results, probeErr := ProbeSnapshotWithTargets(ctx, &merged, spec.Probe.Executor, targets, spec.Probe.options())
		probeSummary = SummarizeProbes(len(targets), results, probeErr)
		if probeErr != nil {
			return RefreshResult{}, probeErr
		}
	}
	if err := os.MkdirAll(filepath.Dir(spec.SnapshotPath), 0700); err != nil {
		return RefreshResult{}, fmt.Errorf("create snapshot directory: %w", err)
	}
	if err := SaveWithHistory(spec.SnapshotPath, merged, DefaultHistoryLimit); err != nil {
		return RefreshResult{}, fmt.Errorf("save subscription snapshot: %w", err)
	}
	return RefreshResult{
		Body: body, ExtraInfo: extraInfo, Nodes: parsed.Nodes,
		ParseSummary:    ParseSummary{Format: parsed.Format, Nodes: len(parsed.Nodes), Skipped: parsed.Skipped},
		SnapshotSummary: SummarizeMerge(old.Nodes, parsed.Nodes, merged.Nodes),
		ProbeSummary:    probeSummary,
	}, nil
}

func mergeOptions(policy SchedulePolicy) MergeOptions {
	defaults := DefaultMergeOptions()
	options := MergeOptions{
		Mode: policy.UpdateMode, RemoveMissing: policy.RemoveMissing,
		MissingGraceRuns: policy.MissingGraceRuns, FailureAction: policy.FailureAction,
		MinKeep: policy.MinKeep, MaxConsecutiveFailures: policy.MaxConsecutiveFailures,
	}
	if options.Mode == "" && !options.RemoveMissing && options.MissingGraceRuns == 0 && options.FailureAction == "" && options.MinKeep == 0 && options.MaxConsecutiveFailures == 0 {
		return defaults
	}
	if options.Mode == "" {
		options.Mode = defaults.Mode
	}
	if options.MissingGraceRuns == 0 {
		options.MissingGraceRuns = defaults.MissingGraceRuns
	}
	if options.FailureAction == "" {
		options.FailureAction = defaults.FailureAction
	}
	if options.MinKeep == 0 {
		options.MinKeep = defaults.MinKeep
	}
	return options
}

// MarshalResult returns the JSON envelope consumed by the legacy job API.
func MarshalResult(result RefreshResult) string {
	data, _ := json.Marshal(result)
	return string(data)
}

package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/uif/uifd/subscription/parser"
)

// Snapshot is the Go-side representation of subscription state. It is kept
// independent from uif.json so callers can validate and merge an update before
// deciding when to publish it.
type Snapshot struct {
	SchemaVersion int            `json:"schema_version"`
	Nodes         []SnapshotNode `json:"nodes"`
	Policy        SnapshotPolicy `json:"policy,omitempty"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty"`
}

// SnapshotNode combines parsed node data with local state. The embedded parser
// node is flattened in JSON, which keeps the snapshot compatible with the
// existing outbound-node shape.
type SnapshotNode struct {
	parser.Node
	ID                  string `json:"id,omitempty"`
	Enabled             bool   `json:"enabled"`
	CoreTag             string `json:"core_tag,omitempty"`
	Delay               string `json:"delay,omitempty"`
	LastSeenAt          int64  `json:"last_seen_at,omitempty"`
	LastProbeAt         int64  `json:"last_probe_at,omitempty"`
	LastProbeDelayMs    int    `json:"last_probe_delay_ms"`
	LastProbeStatus     string `json:"last_probe_status,omitempty"`
	ConsecutiveFailures int    `json:"consecutive_failures,omitempty"`
	Quarantined         bool   `json:"quarantined,omitempty"`
	StaleRuns           int    `json:"stale_runs,omitempty"`
	ProbeError          string `json:"probe_error,omitempty"`
}

// SnapshotPolicy contains only the safety controls needed by snapshot merge.
type SnapshotPolicy struct {
	UpdateMode             string `json:"update_mode,omitempty"`
	RemoveMissing          bool   `json:"remove_missing,omitempty"`
	MissingGraceRuns       int    `json:"missing_grace_runs,omitempty"`
	FailureAction          string `json:"failure_action,omitempty"`
	MinKeep                int    `json:"min_keep,omitempty"`
	MaxConsecutiveFailures int    `json:"max_consecutive_failures,omitempty"`
}

// MergeOptions controls a parse result update. Replace never carries old
// missing nodes; merge carries them for MissingGraceRuns successful updates.
type MergeOptions struct {
	Mode                   string
	RemoveMissing          bool
	MissingGraceRuns       int
	FailureAction          string
	MinKeep                int
	MaxConsecutiveFailures int
}

// SnapshotSummary describes the effect of a successful parse/merge.
// Missing nodes are still retained by the grace policy; removed nodes are no
// longer present in the resulting snapshot.
type SnapshotSummary struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Missing int `json:"missing"`
	Removed int `json:"removed"`
}

// SummarizeMerge compares the identities before and after a successful merge.
// Updated counts matching identities, including a source-side tag/content
// refresh that keeps the same stable identity.
func SummarizeMerge(old []SnapshotNode, incoming []parser.Node, merged []SnapshotNode) SnapshotSummary {
	oldSet := make(map[string]struct{}, len(old))
	for _, node := range old {
		oldSet[nodeFingerprint(node)] = struct{}{}
	}
	incomingSet := make(map[string]struct{}, len(incoming))
	for _, node := range incoming {
		incomingSet[parser.Fingerprint(node)] = struct{}{}
	}
	mergedSet := make(map[string]struct{}, len(merged))
	for _, node := range merged {
		mergedSet[nodeFingerprint(node)] = struct{}{}
	}

	var summary SnapshotSummary
	for fingerprint := range incomingSet {
		if _, ok := oldSet[fingerprint]; ok {
			summary.Updated++
		} else {
			summary.Added++
		}
	}
	for fingerprint := range oldSet {
		if _, ok := incomingSet[fingerprint]; ok {
			continue
		}
		summary.Missing++
		if _, ok := mergedSet[fingerprint]; !ok {
			summary.Removed++
		}
	}
	return summary
}

func DefaultMergeOptions() MergeOptions {
	return MergeOptions{Mode: "merge", RemoveMissing: true, MissingGraceRuns: 3, FailureAction: "quarantine", MinKeep: 1}
}

// NewSnapshot converts parser nodes and assigns a deterministic ID when no
// local ID exists. Parsing is deliberately the only source of node content.
func NewSnapshot(nodes []parser.Node) Snapshot {
	result := Snapshot{SchemaVersion: 1, Nodes: make([]SnapshotNode, 0, len(nodes))}
	for _, node := range nodes {
		result.Nodes = append(result.Nodes, newSnapshotNode(node))
	}
	return result
}

func newSnapshotNode(node parser.Node) SnapshotNode {
	// Always derive identity from the normalized parser content. A supplied
	// fingerprint is treated as derived/cache data, never as an authority.
	node.Fingerprint = parser.Fingerprint(node)
	return SnapshotNode{
		Node:             node,
		ID:               node.Fingerprint,
		Enabled:          true,
		LastProbeDelayMs: -1,
		LastProbeStatus:  "unknown",
	}
}

func nodeFingerprint(node SnapshotNode) string {
	// Recompute from parser content so snapshots made by older versions or
	// callers with a stale fingerprint still match the same node.
	return parser.Fingerprint(node.Node)
}

func normalizeOptions(options MergeOptions) MergeOptions {
	if options.Mode == "" {
		options.Mode = "merge"
	}
	if options.MinKeep < 0 {
		options.MinKeep = 0
	}
	if options.MissingGraceRuns < 0 {
		options.MissingGraceRuns = 0
	}
	if options.FailureAction != "delete" && options.FailureAction != "quarantine" {
		options.FailureAction = "quarantine"
	}
	return options
}

// MergeNodes applies a successful parser result to old snapshot nodes. It
// preserves local ID, enabled state, and health fields for matching nodes.
func MergeNodes(old []SnapshotNode, incoming []parser.Node, options MergeOptions) []SnapshotNode {
	options = normalizeOptions(options)
	oldByFingerprint := make(map[string]SnapshotNode, len(old))
	for _, node := range old {
		oldByFingerprint[nodeFingerprint(node)] = node
	}
	seen := make(map[string]bool, len(incoming))
	merged := make([]SnapshotNode, 0, len(incoming)+len(old))
	for _, parsed := range incoming {
		fingerprint := parser.Fingerprint(parsed)
		if seen[fingerprint] {
			continue
		}
		seen[fingerprint] = true
		fresh := newSnapshotNode(parsed)
		if previous, ok := oldByFingerprint[fingerprint]; ok {
			fresh.ID = previous.ID
			fresh.Enabled = previous.Enabled
			fresh.CoreTag = previous.CoreTag
			fresh.Delay = previous.Delay
			fresh.LastSeenAt = previous.LastSeenAt
			fresh.LastProbeAt = previous.LastProbeAt
			fresh.LastProbeDelayMs = previous.LastProbeDelayMs
			fresh.LastProbeStatus = previous.LastProbeStatus
			fresh.ConsecutiveFailures = previous.ConsecutiveFailures
			fresh.Quarantined = previous.Quarantined
			fresh.ProbeError = previous.ProbeError
		}
		fresh.StaleRuns = 0
		fresh.LastSeenAt = time.Now().UnixMilli()
		merged = append(merged, fresh)
	}
	if options.Mode == "replace" {
		return protectMinimum(merged, old, options)
	}
	for _, previous := range old {
		fingerprint := nodeFingerprint(previous)
		if seen[fingerprint] {
			continue
		}
		previous.Fingerprint = fingerprint
		previous.StaleRuns++
		if options.RemoveMissing && options.MissingGraceRuns > 0 && previous.StaleRuns >= options.MissingGraceRuns {
			continue
		}
		merged = append(merged, previous)
	}
	return protectMinimum(merged, old, options)
}

func protectMinimum(nodes, old []SnapshotNode, options MergeOptions) []SnapshotNode {
	if options.MinKeep <= 0 {
		return nodes
	}
	available := 0
	for _, node := range nodes {
		if node.Enabled && !node.Quarantined {
			available++
		}
	}
	if available >= options.MinKeep {
		return nodes
	}
	present := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		present[nodeFingerprint(node)] = true
	}
	for _, previous := range old {
		if available >= options.MinKeep {
			break
		}
		if present[nodeFingerprint(previous)] || !previous.Enabled || previous.Quarantined {
			continue
		}
		previous.Fingerprint = nodeFingerprint(previous)
		previous.StaleRuns++
		nodes = append(nodes, previous)
		present[previous.Fingerprint] = true
		available++
	}
	return nodes
}

// ProbeResult is the normalized outcome of a node health check.
type ProbeResult struct {
	Success bool
	DelayMs int
	Error   string
}

// ApplyProbeResult updates health state and applies the delete/quarantine
// policy without allowing failure handling to remove the last min_keep nodes.
func ApplyProbeResult(nodes []SnapshotNode, index int, result ProbeResult, options MergeOptions) *SnapshotNode {
	if index < 0 || index >= len(nodes) {
		return nil
	}
	options = normalizeOptions(options)
	node := &nodes[index]
	healthy := result.Success && result.DelayMs > 0
	node.LastProbeDelayMs = -1
	if result.Success && result.DelayMs > 0 {
		node.LastProbeDelayMs = result.DelayMs
		node.Delay = fmt.Sprintf("%d", result.DelayMs)
	}
	if healthy {
		node.LastProbeStatus = "healthy"
		node.ConsecutiveFailures = 0
		node.Quarantined = false
		node.ProbeError = ""
		return node
	}
	node.Delay = "-1"
	node.LastProbeStatus = "failed"
	node.ConsecutiveFailures++
	node.ProbeError = result.Error
	maxFailures := options.MaxConsecutiveFailures
	if maxFailures <= 0 {
		maxFailures = 3
	}
	if node.ConsecutiveFailures < maxFailures {
		return node
	}
	if options.FailureAction != "delete" {
		node.Quarantined = true
		return node
	}
	// Deleting means disabling the failed node. Keep at least min_keep healthy
	// enabled nodes; if that would be violated, quarantine instead.
	if options.MinKeep <= 0 {
		node.Enabled = false
		node.Quarantined = true
		return node
	}
	available := 0
	for i, candidate := range nodes {
		if i != index && candidate.Enabled && !candidate.Quarantined {
			available++
		}
	}
	if available >= options.MinKeep {
		node.Enabled = false
		node.Quarantined = true
		return node
	}
	node.Quarantined = true
	return node
}

// ApplyParse updates a snapshot only when parsing succeeded and produced at
// least one node. On failure it returns the original snapshot unchanged.
func ApplyParse(old Snapshot, result parser.ParseResult, parseErr error, options MergeOptions) (Snapshot, error) {
	if parseErr != nil {
		return old, parseErr
	}
	if len(result.Nodes) == 0 {
		return old, errors.New("subscription parse produced no nodes")
	}
	options = normalizeOptions(options)
	old.Nodes = MergeNodes(old.Nodes, result.Nodes, options)
	if old.SchemaVersion == 0 {
		old.SchemaVersion = 1
	}
	old.UpdatedAt = time.Now()
	old.Policy = SnapshotPolicy{UpdateMode: options.Mode, RemoveMissing: options.RemoveMissing, MissingGraceRuns: options.MissingGraceRuns, FailureAction: options.FailureAction, MinKeep: options.MinKeep, MaxConsecutiveFailures: options.MaxConsecutiveFailures}
	return old, nil
}

// SaveAtomicJSON writes a complete snapshot beside the destination and renames
// it into place. A failed marshal, write, sync, or rename never truncates the
// previous file.
func SaveAtomicJSON(path string, snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmp := file.Name()
	cleanup := func() { _ = file.Close(); _ = os.Remove(tmp) }
	if err := file.Chmod(0600); err != nil {
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename snapshot: %w", err)
	}
	return nil
}

func LoadJSON(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

// Save is a concise alias for callers that do not need to distinguish the
// persistence format from the snapshot itself.
func Save(path string, snapshot Snapshot) error { return SaveAtomicJSON(path, snapshot) }

// Load is the matching alias for Save.
func Load(path string) (Snapshot, error) { return LoadJSON(path) }

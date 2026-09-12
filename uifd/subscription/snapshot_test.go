package subscription

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/uif/uifd/subscription/parser"
)

func testParserNode(tag, address string) parser.Node {
	return parser.Node{
		Protocol:  "trojan",
		Tag:       tag,
		Transport: parser.Transport{Address: address, Port: 443, Protocol: "tcp", TLSType: "tls", TLS: map[string]interface{}{"server_name": "example.com"}},
		Setting:   map[string]interface{}{"password": "secret"},
	}
}

func TestNewSnapshotUsesStableParserFingerprint(t *testing.T) {
	first := NewSnapshot([]parser.Node{testParserNode("one", "node.example")})
	second := NewSnapshot([]parser.Node{testParserNode("renamed", "node.example")})
	if first.Nodes[0].Fingerprint != second.Nodes[0].Fingerprint {
		t.Fatalf("renaming a node changed fingerprint: %q != %q", first.Nodes[0].Fingerprint, second.Nodes[0].Fingerprint)
	}
	if first.Nodes[0].ID != first.Nodes[0].Fingerprint {
		t.Fatalf("new node id = %q, want fingerprint", first.Nodes[0].ID)
	}
}

func TestMergeNodesPreservesLocalAndHealthState(t *testing.T) {
	old := NewSnapshot([]parser.Node{testParserNode("old", "node.example")})
	old.Nodes[0].ID = "local-id"
	old.Nodes[0].Enabled = false
	old.Nodes[0].LastProbeStatus = "failed"
	old.Nodes[0].LastProbeDelayMs = -1
	old.Nodes[0].ConsecutiveFailures = 2
	old.Nodes[0].Quarantined = true
	old.Nodes[0].Delay = "-1"

	merged := MergeNodes(old.Nodes, []parser.Node{testParserNode("new", "node.example")}, MergeOptions{Mode: "replace"})
	if len(merged) != 1 {
		t.Fatalf("merged node count = %d, want 1", len(merged))
	}
	node := merged[0]
	if node.ID != "local-id" || node.Enabled || node.LastProbeStatus != "failed" || node.ConsecutiveFailures != 2 || !node.Quarantined || node.Delay != "-1" {
		t.Fatalf("local state was not preserved: %#v", node)
	}
	if node.Tag != "new" {
		t.Fatalf("parsed content was not replaced: tag = %q", node.Tag)
	}
}

func TestMergeMissingGraceAndMinimumKeep(t *testing.T) {
	old := NewSnapshot([]parser.Node{testParserNode("one", "one.example"), testParserNode("two", "two.example")})
	options := MergeOptions{Mode: "merge", RemoveMissing: true, MissingGraceRuns: 2, MinKeep: 1, FailureAction: "delete"}

	first := MergeNodes(old.Nodes, []parser.Node{testParserNode("one", "one.example")}, options)
	if len(first) != 2 || first[1].StaleRuns != 1 {
		t.Fatalf("first missing run = %#v", first)
	}
	second := MergeNodes(first, []parser.Node{testParserNode("one", "one.example")}, options)
	if len(second) != 1 {
		t.Fatalf("second missing run did not remove stale node: %d", len(second))
	}

	kept := MergeNodes(old.Nodes, nil, MergeOptions{Mode: "replace", MinKeep: 1, FailureAction: "delete"})
	if len(kept) != 1 || !kept[0].Enabled {
		t.Fatalf("minimum keep protection failed: %#v", kept)
	}
}

func TestApplyParseFailureDoesNotOverwrite(t *testing.T) {
	old := NewSnapshot([]parser.Node{testParserNode("old", "old.example")})
	result, err := ApplyParse(old, parser.ParseResult{}, os.ErrInvalid, DefaultMergeOptions())
	if err == nil || len(result.Nodes) != 1 || result.Nodes[0].Tag != "old" {
		t.Fatalf("failed parse handling mismatch: %#v, %v", result, err)
	}
	if _, err = ApplyParse(old, parser.ParseResult{}, nil, DefaultMergeOptions()); err == nil {
		t.Fatal("empty successful parse should be rejected")
	}
}

func TestApplyProbeResultHonorsFailureActionAndMinKeep(t *testing.T) {
	nodes := NewSnapshot([]parser.Node{testParserNode("one", "one.example"), testParserNode("two", "two.example")}).Nodes
	options := MergeOptions{FailureAction: "delete", MinKeep: 1, MaxConsecutiveFailures: 1}
	ApplyProbeResult(nodes, 0, ProbeResult{Success: false, Error: "timeout"}, options)
	if nodes[0].Enabled || !nodes[0].Quarantined {
		t.Fatalf("failed node should be disabled with healthy spare: %#v", nodes[0])
	}
	ApplyProbeResult(nodes, 1, ProbeResult{Success: false, Error: "timeout"}, options)
	if !nodes[1].Enabled || !nodes[1].Quarantined {
		t.Fatalf("minimum keep protection failed: %#v", nodes[1])
	}
}

func TestSaveAtomicJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	want := NewSnapshot([]parser.Node{testParserNode("node", "node.example")})
	if err := SaveAtomicJSON(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].Fingerprint != want.Nodes[0].Fingerprint {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

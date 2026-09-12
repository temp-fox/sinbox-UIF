package subscription

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/uif/uifd/subscription/parser"
)

func TestSaveWithHistoryRetainsAndLimitsRevisions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	first := NewSnapshot([]parser.Node{testParserNode("one", "one.example")})
	if err := SaveWithHistory(path, first, 2); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		next := NewSnapshot([]parser.Node{testParserNode("node", string(rune('a'+i))+".example")})
		if err := SaveWithHistory(path, next, 2); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := SnapshotHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("history count = %d, want 2", len(entries))
	}
}

func TestRestoreSnapshotOnlyAllowsListedHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	first := NewSnapshot([]parser.Node{testParserNode("one", "one.example")})
	second := NewSnapshot([]parser.Node{testParserNode("two", "two.example")})
	if err := SaveWithHistory(path, first, 2); err != nil {
		t.Fatal(err)
	}
	if err := SaveWithHistory(path, second, 2); err != nil {
		t.Fatal(err)
	}
	entries, err := SnapshotHistory(path)
	if err != nil || len(entries) != 1 {
		t.Fatalf("history = %#v, err=%v", entries, err)
	}
	if _, err := RestoreSnapshot(path, filepath.Join(dir, "../outside.json"), 2); err == nil {
		t.Fatal("unlisted history accepted")
	}
	restored, err := RestoreSnapshot(path, entries[0].Path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Nodes) != 1 || restored.Nodes[0].Tag != "one" {
		t.Fatalf("restored = %#v", restored)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/uif/uifd/subscription"
)

func TestParseAndSaveSubscriptionSnapshotIntegration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "snapshot.json")
	first, err := parseAndSaveSubscriptionSnapshot("trojan://secret@example.com:443#one\ntrojan://secret@example.net:443#two", "quota", path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Body         string `json:"body"`
		ParseSummary struct {
			Nodes int `json:"nodes"`
		} `json:"parse_summary"`
		SnapshotSummary subscription.SnapshotSummary `json:"snapshot_summary"`
	}
	if err := json.Unmarshal([]byte(first), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Body == "" || envelope.ParseSummary.Nodes != 2 || envelope.SnapshotSummary.Added != 2 {
		t.Fatalf("unexpected first result: %#v", envelope)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	second, err := parseAndSaveSubscriptionSnapshot("trojan://secret@example.com:443#renamed", "", path)
	if err != nil {
		t.Fatal(err)
	}
	var secondEnvelope struct {
		SnapshotSummary subscription.SnapshotSummary `json:"snapshot_summary"`
	}
	if err := json.Unmarshal([]byte(second), &secondEnvelope); err != nil {
		t.Fatal(err)
	}
	if secondEnvelope.SnapshotSummary.Updated != 1 || secondEnvelope.SnapshotSummary.Missing != 1 || secondEnvelope.SnapshotSummary.Removed != 0 {
		t.Fatalf("unexpected second result: %#v", secondEnvelope)
	}
}

func TestParseAndSaveSubscriptionSnapshotFailureKeepsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if _, err := parseAndSaveSubscriptionSnapshot("trojan://secret@example.com:443#one", "", path); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseAndSaveSubscriptionSnapshot("not a subscription", "", path); err == nil {
		t.Fatal("invalid input accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed parse overwrote snapshot")
	}
}

func TestSubscriptionSnapshotPathStaysInWorkspace(t *testing.T) {
	if _, err := subscriptionSnapshotPath("../../outside.json", "sub"); err == nil {
		t.Fatal("path traversal accepted")
	}
}

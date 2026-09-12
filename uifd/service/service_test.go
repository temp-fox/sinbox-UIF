package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uif/uifd/uif"
)

func TestSubscriptionSpecsFromCompleteUIFConfig(t *testing.T) {
	config := map[string]interface{}{
		"uif": map[string]interface{}{"version": "test"},
		"data": map[string]interface{}{
			"subscribe": []interface{}{
				map[string]interface{}{
					"id":  "nested-sub",
					"url": "https://subscription.example/list",
					"policy": map[string]interface{}{
						"update_enabled":      true,
						"update_interval_sec": float64(60),
					},
				},
			},
			"routes": []interface{}{
				map[string]interface{}{
					"domain": "probe.example",
					"target": map[string]interface{}{
						"kind":            "subscription",
						"subscription_id": "nested-sub",
					},
				},
			},
		},
	}

	specs, err := subscriptionSpecsFromConfig(config)
	if err != nil {
		t.Fatalf("subscriptionSpecsFromConfig returned error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("got %d specs, want 1", len(specs))
	}
	spec := specs[0]
	if spec.ID != "nested-sub" || spec.URL != "https://subscription.example/list" {
		t.Fatalf("unexpected spec: %#v", spec)
	}
	if !spec.Policy.UpdateEnabled || spec.Policy.UpdateIntervalSec != 60 {
		t.Fatalf("policy was not loaded from data.subscribe: %#v", spec.Policy)
	}
	if spec.Probe.TargetResolver == nil {
		t.Fatal("probe target resolver was not configured")
	}
	if got := spec.Probe.TargetResolver.Resolve("nested-sub"); got != "https://probe.example" {
		t.Fatalf("probe route endpoint = %q, want https://probe.example", got)
	}
}

func TestSubscriptionSpecsFromLegacySectionConfig(t *testing.T) {
	config := map[string]interface{}{
		"subscribe": []interface{}{map[string]interface{}{"id": "legacy-sub", "source": "https://legacy.example/list"}},
	}
	specs, err := subscriptionSpecsFromConfig(config)
	if err != nil {
		t.Fatalf("subscriptionSpecsFromConfig returned error: %v", err)
	}
	if len(specs) != 1 || specs[0].ID != "legacy-sub" {
		t.Fatalf("unexpected legacy specs: %#v", specs)
	}
}

func TestSubscriptionSpecsAssignStableIDsForLegacyEntries(t *testing.T) {
	config := map[string]interface{}{
		"subscribe": []interface{}{
			map[string]interface{}{"tag": "legacy", "data": "https://legacy.example/list"},
		},
	}
	first, err := subscriptionSpecsFromConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := subscriptionSpecsFromConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 || first[0].ID == "" || first[0].ID != second[0].ID {
		t.Fatalf("legacy ID was not stable: first=%#v second=%#v", first, second)
	}
	if !strings.HasPrefix(first[0].ID, "legacy-subscription-") {
		t.Fatalf("unexpected legacy ID: %q", first[0].ID)
	}
}

func TestValidateSubscriptionResultRejectsEmptyBody(t *testing.T) {
	if _, err := validateSubscriptionResult(" \n\t"); err == nil {
		t.Fatal("empty subscription response was accepted")
	}
}

func TestValidateSubscriptionResultKeepsBody(t *testing.T) {
	const want = "profile-content"
	got, err := validateSubscriptionResult(want)
	if err != nil {
		t.Fatalf("validateSubscriptionResult returned error: %v", err)
	}
	if got != want {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestParseSubscriptionResultReturnsCompatibleEnvelopeAndSummary(t *testing.T) {
	result, err := parseSubscriptionResult("trojan://secret@example.com:443#node", "quota")
	if err != nil {
		t.Fatalf("parseSubscriptionResult returned error: %v", err)
	}
	var envelope struct {
		Body         string `json:"body"`
		ExtraInfo    string `json:"extra_info"`
		ParseSummary struct {
			Format string `json:"format"`
			Nodes  int    `json:"nodes"`
		} `json:"parse_summary"`
	}
	if err := json.Unmarshal([]byte(result), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Body == "" || envelope.ExtraInfo != "quota" || envelope.ParseSummary.Format != "v2rayn" || envelope.ParseSummary.Nodes != 1 {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}

func TestParseSubscriptionResultRejectsUnparseableBody(t *testing.T) {
	if _, err := parseSubscriptionResult("not a subscription", ""); err == nil {
		t.Fatal("unparseable subscription was accepted")
	}
}

func TestSetQuicLink(t *testing.T) {
	// err := SetQuicLink()
	// assert.Nil(t, err)
}

func TestCheckPort(t *testing.T) {
	_, err := uif.TCPPortCheck("4544")
	assert.NotNil(t, err)

	// err = CheckPort()
	// assert.NotNil(t, err)
}

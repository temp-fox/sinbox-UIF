package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uif/uifd/uif"
)

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

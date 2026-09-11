package main

import (
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

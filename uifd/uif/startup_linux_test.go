//go:build linux
// +build linux

package uif

import (
	"strings"
	"testing"
)

func TestOpenWrtServiceScriptUsesVersionedService(t *testing.T) {
	servicePath := "/opt/uif-adapted-20260914/uifd/26.03.11/uif_service"
	script := openWrtServiceScript(servicePath)

	for _, want := range []string{
		"#!/bin/sh /etc/rc.common",
		"USE_PROCD=1",
		"START=96",
		"APP_PATH='" + servicePath + "'",
		"procd_set_param command \"$APP_PATH\"",
		"procd_set_param respawn",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("service script does not contain %q:\n%s", want, script)
		}
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("/opt/uif'service")
	want := "'/opt/uif'\\''service'"
	if got != want {
		t.Fatalf("shellQuote() = %q, want %q", got, want)
	}
}

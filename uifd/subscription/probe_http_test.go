package subscription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnsupportedProbeExecutorIsExplicit(t *testing.T) {
	result, err := (UnsupportedProbeExecutor{}).Probe(context.Background(), ProbeTarget{})
	if err == nil || result.Success || result.Error == "" {
		t.Fatalf("unsupported executor result = %#v, err = %v", result, err)
	}
}

func TestHTTPURLProbeExecutorOnlyProbesConfiguredURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %s, want HEAD", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	result, err := NewHTTPURLProbeExecutor(nil, server.URL).Probe(context.Background(), ProbeTarget{Address: "not-used.invalid"})
	if err != nil || !result.Success || result.DelayMs <= 0 {
		t.Fatalf("URL probe result = %#v, err = %v", result, err)
	}
}

func TestHTTPURLProbeExecutorRejectsCredentialsAndNonHTTP(t *testing.T) {
	for _, endpoint := range []string{"file:///tmp/probe", "http://user:pass@example.com/"} {
		result, err := NewHTTPURLProbeExecutor(nil, endpoint).Probe(context.Background(), ProbeTarget{})
		if err == nil || result.Success {
			t.Fatalf("endpoint %q was accepted: %#v, %v", endpoint, result, err)
		}
	}
}

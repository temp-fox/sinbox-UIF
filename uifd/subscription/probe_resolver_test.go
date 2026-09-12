package subscription

import (
	"testing"

	"github.com/uif/uifd/subscription/parser"
)

func TestProbeTargetResolverMatchesSubscriptionRoute(t *testing.T) {
	resolver := ProbeTargetResolver{
		Routes: []map[string]interface{}{
			{"target": map[string]interface{}{"kind": "other", "subscription_id": "sub"}, "domain": []interface{}{"wrong.example"}},
			{"target": map[string]interface{}{"kind": "subscription", "subscription_id": "sub"}, "domain_suffix": []interface{}{"probe.example"}, "port": []interface{}{float64(80)}},
		},
		DefaultURL: "https://fallback.example/204",
	}
	if got := resolver.Resolve("sub"); got != "http://probe.example:80" {
		t.Fatalf("resolved URL = %q", got)
	}
}

func TestProbeTargetResolverSupportsIPCIDRAndFallback(t *testing.T) {
	resolver := ProbeTargetResolver{
		Routes: []map[string]interface{}{
			{"target": map[string]interface{}{"kind": "subscription", "subscription_id": "ip"}, "ip_cidr": []interface{}{"192.0.2.0/24"}, "port": []interface{}{"8443"}},
		},
		DefaultURL: "https://fallback.example/204",
	}
	if got := resolver.Resolve("ip"); got != "https://192.0.2.0:8443" {
		t.Fatalf("CIDR URL = %q", got)
	}
	if got := resolver.Resolve("missing"); got != "https://fallback.example/204" {
		t.Fatalf("fallback URL = %q", got)
	}
}

func TestProbeTargetResolverRejectsUnsafeRouteAndFallbackURLs(t *testing.T) {
	resolver := ProbeTargetResolver{
		Routes: []map[string]interface{}{
			{"target": map[string]interface{}{"kind": "subscription", "subscription_id": "sub"}, "domain": []interface{}{"https://user:pass@example/secret", "bad host"}},
		},
		DefaultURL: "file:///tmp/probe",
	}
	if got := resolver.Resolve("sub"); got != DefaultProbeURL {
		t.Fatalf("unsafe URL = %q, want %q", got, DefaultProbeURL)
	}
}

func TestProbeTargetResolverAssignsURLToEligibleTargets(t *testing.T) {
	snapshot := NewSnapshot([]parser.Node{
		testParserNode("one", "one.example"),
		testParserNode("two", "two.example"),
	})
	resolver := ProbeTargetResolver{DefaultURL: "https://fallback.example/204"}
	targets := resolver.ResolveTargets(snapshot, "sub")
	if len(targets) != 2 || targets[0].URL != "https://fallback.example/204" || targets[1].URL != targets[0].URL {
		t.Fatalf("targets = %#v", targets)
	}
}

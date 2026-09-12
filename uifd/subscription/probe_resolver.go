package subscription

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ProbeTargetResolver derives one endpoint-level probe URL from legacy UIF
// routes. It only consumes routes that explicitly target a subscription; it
// does not attempt to emulate the route engine or a per-node proxy.
type ProbeTargetResolver struct {
	Routes     []map[string]interface{}
	DefaultURL string
}

// Resolve returns a safe HTTP(S) URL for subscriptionID. A matching route is
// preferred; when no usable route match exists, DefaultURL is used. Invalid
// fallback values are replaced with the package default.
func (r ProbeTargetResolver) Resolve(subscriptionID string) string {
	id := strings.TrimSpace(subscriptionID)
	if id != "" {
		for _, route := range r.Routes {
			if routeSubscriptionID(route) != id {
				continue
			}
			if endpoint := routeEndpoint(route); endpoint != "" {
				return endpoint
			}
		}
	}
	return safeProbeURL(r.DefaultURL)
}

// ResolveURL is an explicit alias for callers that prefer URL terminology.
func (r ProbeTargetResolver) ResolveURL(subscriptionID string) string {
	return r.Resolve(subscriptionID)
}

// ResolveTargets creates normal snapshot targets and assigns the resolved
// endpoint to each target. The resolver only chooses the HTTP(S) endpoint;
// it does not claim that the endpoint exercises an individual node.
func (r ProbeTargetResolver) ResolveTargets(snapshot Snapshot, subscriptionID string) []ProbeTarget {
	targets := TargetsForSnapshot(snapshot)
	endpoint := r.Resolve(subscriptionID)
	for i := range targets {
		targets[i].URL = endpoint
	}
	return targets
}

func routeSubscriptionID(route map[string]interface{}) string {
	target, ok := route["target"].(map[string]interface{})
	if !ok {
		return ""
	}
	kind, _ := target["kind"].(string)
	if strings.TrimSpace(kind) != "subscription" {
		return ""
	}
	id, _ := target["subscription_id"].(string)
	return strings.TrimSpace(id)
}

func routeEndpoint(route map[string]interface{}) string {
	var host string
	for _, key := range []string{"domain", "domain_suffix", "ip_cidr"} {
		for _, value := range stringValues(route[key]) {
			if candidate := routeHost(key, value); candidate != "" {
				host = candidate
				break
			}
		}
		if host != "" {
			break
		}
	}
	if host == "" {
		return ""
	}
	port := 0
	for _, value := range stringValues(route["port"]) {
		if candidate, ok := validPort(value); ok {
			port = candidate
			break
		}
	}
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}
	endpoint := scheme + "://" + host
	if port != 0 && !(scheme == "https" && port == 443) {
		endpoint = fmt.Sprintf("%s:%d", endpoint, port)
	}
	return safeProbeURL(endpoint)
}

func routeHost(kind, value string) string {
	value = strings.TrimSpace(strings.TrimSuffix(value, "."))
	if kind == "domain_suffix" {
		value = strings.TrimLeft(value, ".")
	}
	if value == "" || strings.ContainsAny(value, "?#@\\% ") || strings.Contains(value, "*") {
		return ""
	}
	if kind != "ip_cidr" && strings.Contains(value, "/") {
		return ""
	}
	if kind == "ip_cidr" {
		ip, _, err := net.ParseCIDR(value)
		if err != nil || ip == nil {
			return ""
		}
		return formatHost(ip.String())
	}
	if net.ParseIP(value) != nil {
		return formatHost(value)
	}
	if strings.Contains(value, ":") || !strings.Contains(value, ".") {
		return ""
	}
	candidate, err := url.Parse("https://" + value)
	if err != nil || candidate.Hostname() != value || candidate.User != nil {
		return ""
	}
	return value
}

func formatHost(host string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func stringValues(value interface{}) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []string:
		return typed
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			} else if number, ok := item.(float64); ok {
				values = append(values, strconv.Itoa(int(number)))
			}
		}
		return values
	case float64:
		return []string{strconv.Itoa(int(typed))}
	case int:
		return []string{strconv.Itoa(typed)}
	default:
		return nil
	}
}

func validPort(value string) (int, bool) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	return port, err == nil && port > 0 && port <= 65535
}

func safeProbeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultProbeURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return DefaultProbeURL
	}
	return u.String()
}

package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Parse detects the input format and returns normalized nodes. Detection is
// deliberately conservative: explicit UIF and URL prefixes win, followed by
// JSON and YAML shape checks.
func Parse(input string) (ParseResult, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(trimmed), "uif://") || strings.Contains(strings.ToLower(trimmed), "\nuif://") {
		nodes, err := ParseUIFRaw(input)
		return ParseResult{Format: "uif", Nodes: nodes}, err
	}
	if looksLikeLinkList(trimmed) || strings.HasPrefix(strings.ToLower(trimmed), "vmess://") {
		nodes, err := ParseV2rayN(input)
		return ParseResult{Format: "v2rayn", Nodes: nodes}, err
	}
	// V2rayN subscriptions are often one opaque base64 string, so try the
	// decoder before falling through to YAML. ParseV2rayN returns an error for
	// ordinary YAML/JSON text and therefore does not steal those formats.
	if nodes, err := ParseV2rayN(input); err == nil {
		return ParseResult{Format: "v2rayn", Nodes: nodes}, nil
	}
	if json.Valid([]byte(trimmed)) {
		var shape map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &shape); err == nil {
			if _, ok := shape["outbounds"]; ok {
				nodes, parseErr := ParseSingBox(input)
				return ParseResult{Format: "sing-box", Nodes: nodes}, parseErr
			}
		}
	}
	if nodes, err := ParseClash(input); err == nil {
		return ParseResult{Format: "clash", Nodes: nodes}, nil
	}
	return ParseResult{}, fmt.Errorf("unsupported subscription format")
}

func looksLikeLinkList(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return bytes.Contains([]byte(line), []byte("://"))
	}
	return false
}

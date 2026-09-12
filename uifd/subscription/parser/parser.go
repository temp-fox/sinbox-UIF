package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse detects the input format and returns normalized nodes. Detection is
// deliberately conservative: explicit UIF and URL prefixes win, followed by
// JSON and YAML shape checks.
func Parse(input string) (ParseResult, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(trimmed), "uif://") || strings.Contains(strings.ToLower(trimmed), "\nuif://") {
		nodes, err := ParseUIFRaw(input)
		return parseResult("uif", input, nodes, err)
	}
	if looksLikeLinkList(trimmed) || strings.HasPrefix(strings.ToLower(trimmed), "vmess://") {
		nodes, err := ParseV2rayN(input)
		return parseResult("v2rayn", input, nodes, err)
	}
	// V2rayN subscriptions are often one opaque base64 string, so try the
	// decoder before falling through to YAML. ParseV2rayN returns an error for
	// ordinary YAML/JSON text and therefore does not steal those formats.
	if nodes, err := ParseV2rayN(input); err == nil {
		return parseResult("v2rayn", input, nodes, nil)
	}
	if json.Valid([]byte(trimmed)) {
		var shape map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &shape); err == nil {
			if _, ok := shape["outbounds"]; ok {
				nodes, parseErr := ParseSingBox(input)
				return parseResult("sing-box", input, nodes, parseErr)
			}
		}
	}
	if nodes, err := ParseClash(input); err == nil {
		return parseResult("clash", input, nodes, nil)
	}
	return ParseResult{}, fmt.Errorf("unsupported subscription format")
}

func parseResult(format, input string, nodes []Node, err error) (ParseResult, error) {
	result := ParseResult{Format: format, Nodes: nodes}
	if err != nil {
		return result, err
	}
	candidates := candidateCount(format, input)
	if candidates > len(nodes) {
		result.Skipped = candidates - len(nodes)
	}
	return result, nil
}

func candidateCount(format, input string) int {
	if format == "sing-box" {
		var document struct {
			Outbounds []map[string]interface{} `json:"outbounds"`
		}
		if json.Unmarshal([]byte(input), &document) == nil {
			return len(document.Outbounds)
		}
	}
	if format == "clash" {
		var document struct {
			Proxies []map[string]interface{} `yaml:"proxies"`
		}
		if yaml.Unmarshal([]byte(input), &document) == nil {
			return len(document.Proxies)
		}
	}
	count := 0
	text := input
	if format == "v2rayn" {
		if decoded, err := decodeBase64(strings.TrimSpace(input)); err == nil && strings.Contains(string(decoded), "://") {
			text = string(decoded)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if format != "uif" || strings.HasPrefix(strings.ToLower(line), "uif://") {
			count++
		}
	}
	return count
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

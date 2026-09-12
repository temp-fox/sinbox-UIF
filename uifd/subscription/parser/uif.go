package parser

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ParseUIFRaw parses one or more uif:// base64(JSON) lines.
func ParseUIFRaw(input string) ([]Node, error) {
	var nodes []Node
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(strings.ToLower(line), "uif://") {
			continue
		}
		payload, err := decodeBase64(line[6:])
		if err != nil {
			return nil, fmt.Errorf("uif line: %w", err)
		}
		var node Node
		if err := json.Unmarshal(payload, &node); err != nil {
			return nil, fmt.Errorf("uif json: %w", err)
		}
		if node.Protocol == "" {
			continue
		}
		nodes = append(nodes, node)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no UIF nodes found")
	}
	return withFingerprint(nodes), nil
}

func decodeBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	decoders := []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.URLEncoding}
	var last error
	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
		last = err
	}
	return nil, last
}

// ParseJSON dispatches a JSON document to the Sing-box parser. It is kept
// separate from Parse so callers can require strict JSON input.
func ParseJSON(input string) ([]Node, error) { return ParseSingBox(input) }

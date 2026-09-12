package parser

import (
	"encoding/json"
	"fmt"
)

// ParseSingBox converts sing-box JSON outbounds to the normalized model.
func ParseSingBox(input string) ([]Node, error) {
	var document struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(input), &document); err != nil {
		return nil, fmt.Errorf("sing-box json: %w", err)
	}
	if document.Outbounds == nil {
		return nil, fmt.Errorf("sing-box document has no outbounds")
	}
	var nodes []Node
	for _, outbound := range document.Outbounds {
		typ := stringValue(outbound["type"])
		if isIgnoredOutbound(typ) || typ == "" {
			continue
		}
		node := Node{Protocol: mapSingProtocol(typ), Tag: stringValue(outbound["tag"]), Setting: cloneMap(outbound)}
		delete(node.Setting, "type")
		delete(node.Setting, "tag")
		delete(node.Setting, "server")
		delete(node.Setting, "server_port")
		delete(node.Setting, "tls")
		delete(node.Setting, "transport")
		delete(node.Setting, "multiplex")
		node.Transport = transportFromSing(outbound)
		if node.Transport.Address == "" && node.Transport.Port == 0 && !isEndpointOptional(typ) {
			continue
		}
		nodes = append(nodes, node)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported sing-box outbounds found")
	}
	return withFingerprint(nodes), nil
}

func transportFromSing(out map[string]interface{}) Transport {
	t := Transport{Protocol: "tcp", TLSType: "none", Setting: map[string]interface{}{}, TLS: map[string]interface{}{}}
	t.Address = stringValue(out["server"])
	t.Port = intValue(out["server_port"])
	if value, ok := out["transport"].(map[string]interface{}); ok {
		t.Protocol = stringValue(value["type"])
		if t.Protocol == "" {
			t.Protocol = "tcp"
		}
		t.Setting = cloneMap(value)
		delete(t.Setting, "type")
	}
	if value, ok := out["tls"].(map[string]interface{}); ok && boolValue(value["enabled"]) {
		t.TLSType = "tls"
		t.TLS = cloneMap(value)
		delete(t.TLS, "enabled")
	}
	if value, ok := out["multiplex"].(map[string]interface{}); ok {
		t.Multiplex = cloneMap(value)
	}
	return t
}

func mapSingProtocol(value string) string {
	if value == "ss" {
		return "shadowsocks"
	}
	return value
}

func isIgnoredOutbound(value string) bool {
	switch value {
	case "selector", "urltest", "direct", "block", "dns", "一线多拨":
		return true
	}
	return false
}

func isEndpointOptional(value string) bool {
	return value == "wireguard" || value == "http" || value == "socks"
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func stringValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
func boolValue(value interface{}) bool { result, _ := value.(bool); return result }
func intValue(value interface{}) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	case json.Number:
		n, _ := number.Int64()
		return int(n)
	}
	return 0
}

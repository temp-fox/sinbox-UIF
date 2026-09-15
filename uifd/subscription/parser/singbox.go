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
		t.Protocol = normalizeTransportType(stringValue(value["type"]))
		if t.Protocol == "" {
			t.Protocol = "tcp"
		}
		t.Setting = cloneMap(value)
		delete(t.Setting, "type")
	}
	if value, ok := out["tls"].(map[string]interface{}); ok && boolValue(value["enabled"]) {
		t.TLSType = "tls"
		t.TLS = cloneMap(value)
		// 保留 enabled，与 clash.go / v2rayn.go 及前端 uif2singbox 保持一致。
		// 前端用 transport.tls_type != 'none' 决定是否挂 tls，但把整个 tls
		// 对象原样写进 sing-box 配置；hysteria2/tuic/trojan 等协议强制要求
		// tls.enabled=true，删掉会让内核以 "TLS required" 拒绝启动。
		t.TLS["enabled"] = true
	} else if requiresTLS(stringValue(out["type"])) {
		// hysteria2/tuic/trojan 协议本身强制 TLS，订阅源经常省略 tls.enabled
		// （或干脆不写 tls 字段）。若只在 tls.enabled==true 时才挂 tls，
		// 这些节点会被标记成 tls_type=none，前端不补 tls，内核以
		// "TLS required" 拒绝启动。这里按协议强制补上。
		t.TLSType = "tls"
		if value, ok := out["tls"].(map[string]interface{}); ok {
			t.TLS = cloneMap(value)
		}
		t.TLS["enabled"] = true
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

// normalizeTransportType 把 sing-box 1.12.x 已移除的 xhttp/splithttp 传输类型
// 归一为 httpupgrade（XHTTP 的 sing-box 原生等价物）。旧订阅快照仍可能携带
// xhttp，若不归一，前端会把它原样写成 transport.type，导致内核解码失败。
func normalizeTransportType(value string) string {
	switch value {
	case "xhttp", "splithttp":
		return "httpupgrade"
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

// requiresTLS 返回该协议是否强制要求 TLS 传输层（即使订阅源省略 tls 字段）。
func requiresTLS(typ string) bool {
	switch typ {
	case "hysteria2", "tuic", "trojan":
		return true
	}
	return false
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

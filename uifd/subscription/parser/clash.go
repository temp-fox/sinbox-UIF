package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseClash parses a Clash YAML document's proxies list.
func ParseClash(input string) ([]Node, error) {
	var document struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(input), &document); err != nil {
		return nil, fmt.Errorf("clash yaml: %w", err)
	}
	var nodes []Node
	for _, proxy := range document.Proxies {
		node, ok := clashProxy(proxy)
		if ok {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported Clash proxies found")
	}
	return withFingerprint(nodes), nil
}

func clashProxy(proxy map[string]interface{}) (Node, bool) {
	typ := strings.ToLower(stringValue(proxy["type"]))
	protocols := map[string]string{"ss": "shadowsocks", "socks5": "socks", "http": "http"}
	protocol, ok := protocols[typ]
	if !ok {
		protocol = typ
	}
	supported := map[string]bool{"shadowsocks": true, "trojan": true, "vmess": true, "vless": true, "hysteria": true, "hysteria2": true, "tuic": true, "socks": true, "http": true}
	if !supported[protocol] {
		return Node{}, false
	}
	node := Node{Protocol: protocol, Tag: stringValue(proxy["name"]), Setting: map[string]interface{}{}}
	node.Transport = Transport{Address: stringValue(proxy["server"]), Port: intValue(proxy["port"]), Protocol: stringValue(proxy["network"]), TLSType: "none", TLS: map[string]interface{}{}, Setting: map[string]interface{}{}}
	if node.Transport.Protocol == "" {
		node.Transport.Protocol = "tcp"
	}
	for key, value := range proxy {
		switch key {
		case "name", "type", "server", "port", "network", "tls", "skip-cert-verify", "sni", "servername", "alpn", "ws-opts", "grpc-opts", "reality-opts":
		default:
			node.Setting[key] = value
		}
	}
	if proxy["tls"] == true || protocol == "trojan" || protocol == "vmess" || protocol == "vless" || protocol == "hysteria" || protocol == "hysteria2" || protocol == "tuic" {
		node.Transport.TLSType = "tls"
		node.Transport.TLS["enabled"] = true
	}
	if value := stringValue(proxy["skip-cert-verify"]); value != "" {
		node.Transport.TLS["insecure"] = value == "true"
	}
	if value, ok := proxy["skip-cert-verify"].(bool); ok {
		node.Transport.TLS["insecure"] = value
	}
	if value := stringValue(proxy["sni"]); value != "" {
		node.Transport.TLS["server_name"] = value
	} else if value := stringValue(proxy["servername"]); value != "" {
		node.Transport.TLS["server_name"] = value
	}
	if value, ok := proxy["alpn"].([]interface{}); ok {
		node.Transport.TLS["alpn"] = value
	}
	if value, ok := proxy["reality-opts"].(map[string]interface{}); ok {
		node.Transport.TLSType = "reality"
		node.Transport.TLS["reality"] = map[string]interface{}{"enabled": true}
		if key := stringValue(value["public-key"]); key != "" {
			node.Transport.TLS["reality"].(map[string]interface{})["public_key"] = key
		}
		if id := stringValue(value["short-id"]); id != "" {
			node.Transport.TLS["reality"].(map[string]interface{})["short_id"] = id
		}
	}
	if value, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		if path := stringValue(value["path"]); path != "" {
			node.Transport.Setting["path"] = path
		} else {
			node.Transport.Setting["path"] = "/"
		}
		if headers, ok := value["headers"].(map[string]interface{}); ok {
			node.Transport.Setting["headers"] = headers
		}
	}
	if value, ok := proxy["grpc-opts"].(map[string]interface{}); ok {
		if service := stringValue(value["grpc-service-name"]); service != "" {
			node.Transport.Setting["service_name"] = service
		}
	}
	switch protocol {
	case "shadowsocks":
		node.Setting["method"], node.Setting["password"] = proxy["cipher"], proxy["password"]
	case "trojan":
		node.Setting["password"] = proxy["password"]
	case "vmess":
		node.Setting["uuid"], node.Setting["security"], node.Setting["alter_id"] = proxy["uuid"], proxy["cipher"], proxy["alterId"]
	case "vless":
		node.Setting["uuid"], node.Setting["flow"] = proxy["uuid"], proxy["flow"]
	case "hysteria2":
		node.Setting["password"] = proxy["password"]
	}
	return node, node.Transport.Address != "" && node.Transport.Port > 0
}

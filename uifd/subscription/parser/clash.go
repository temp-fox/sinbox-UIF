package parser

import (
	"fmt"
	"strconv"
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
	node.Transport = Transport{Address: stringValue(proxy["server"]), Port: intValue(proxy["port"]), Protocol: normalizeTransportType(stringValue(proxy["network"])), TLSType: "none", TLS: map[string]interface{}{}, Setting: map[string]interface{}{}}
	if node.Transport.Protocol == "" {
		node.Transport.Protocol = "tcp"
	}
	// 不再用 default 分支把 clash 独有字段塞进 Setting：前端会把这些字段
	// 原样展开成 sing-box outbound 顶层字段，sing-box 遇到 cipher/alterId/udp
	// 等未知字段会直接启动失败（unknown field）。协议字段一律由下方 switch
	// 显式转换为 sing-box 原生字段名。
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
		if plugin := stringValue(proxy["plugin"]); plugin != "" {
			node.Setting["plugin"] = plugin
			if opts, ok := proxy["plugin-opts"].(map[string]interface{}); ok {
				node.Setting["plugin_opts"] = cloneMap(opts)
			}
		}
	case "trojan":
		node.Setting["password"] = proxy["password"]
	case "vmess":
		node.Setting["uuid"], node.Setting["security"], node.Setting["alter_id"] = proxy["uuid"], proxy["cipher"], proxy["alterId"]
	case "vless":
		node.Setting["uuid"], node.Setting["flow"] = proxy["uuid"], proxy["flow"]
	case "hysteria":
		if auth := stringValue(proxy["auth-str"]); auth != "" {
			node.Setting["auth_str"] = auth
		} else if auth := stringValue(proxy["auth_str"]); auth != "" {
			node.Setting["auth_str"] = auth
		}
		node.Setting["up_mbps"], node.Setting["down_mbps"] = clashSpeed(proxy["up"]), clashSpeed(proxy["down"])
	case "hysteria2":
		node.Setting["password"] = proxy["password"]
		if obfs := stringValue(proxy["obfs"]); obfs != "" && obfs != "none" {
			node.Setting["obfs"] = map[string]interface{}{"type": obfs, "password": stringValue(proxy["obfs-password"])}
		}
		node.Setting["up_mbps"], node.Setting["down_mbps"] = clashSpeed(proxy["up"]), clashSpeed(proxy["down"])
	case "tuic":
		node.Setting["uuid"], node.Setting["password"] = proxy["uuid"], proxy["password"]
		if cc := stringValue(proxy["congestion-controller"]); cc != "" {
			node.Setting["congestion_control"] = cc
		}
		if urm := stringValue(proxy["udp-relay-mode"]); urm != "" {
			node.Setting["udp_relay_mode"] = urm
		}
		if rr, ok := proxy["reduce-rtt"].(bool); ok && rr {
			node.Setting["zero_rtt_handshake"] = true
		}
	case "socks", "http":
		if user := stringValue(proxy["username"]); user != "" {
			node.Setting["username"] = user
		}
		if pass := stringValue(proxy["password"]); pass != "" {
			node.Setting["password"] = pass
		}
	}
	return node, node.Transport.Address != "" && node.Transport.Port > 0
}

// clashSpeed 把 clash 的速度字段（"50 mbps" 或数字）解析为 Mbps 整数。
func clashSpeed(value interface{}) int {
	switch number := value.(type) {
	case int:
		return number
	case float64:
		return int(number)
	case string:
		parts := strings.Fields(number)
		if len(parts) > 0 {
			if parsed, err := strconv.Atoi(parts[0]); err == nil {
				return parsed
			}
		}
	}
	return 0
}

package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseV2rayN accepts newline-separated URLs or a base64-encoded subscription.
func ParseV2rayN(input string) ([]Node, error) {
	text := strings.TrimSpace(input)
	if decoded, err := decodeBase64(text); err == nil && strings.Contains(string(decoded), "://") {
		text = string(decoded)
	}
	var nodes []Node
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		node, err := parseV2URL(line)
		if err == nil {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no supported V2rayN links found")
	}
	return withFingerprint(nodes), nil
}

func parseV2URL(raw string) (Node, error) {
	if strings.HasPrefix(strings.ToLower(raw), "vmess://") {
		return parseVMess(raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Node{}, err
	}
	protocol := strings.TrimSuffix(strings.ToLower(u.Scheme), "-go")
	if protocol == "ss" {
		return parseSS(u)
	}
	if protocol != "trojan" && protocol != "vless" && protocol != "hysteria2" {
		return Node{}, fmt.Errorf("unsupported scheme")
	}
	port, _ := strconv.Atoi(u.Port())
	if u.Hostname() == "" || port == 0 {
		return Node{}, fmt.Errorf("missing endpoint")
	}
	node := Node{Protocol: protocol, Tag: fragment(u), Setting: map[string]interface{}{}, Transport: Transport{Address: u.Hostname(), Port: port, Protocol: "tcp", TLSType: "tls", TLS: map[string]interface{}{}, Setting: map[string]interface{}{}}}
	password := ""
	username := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		if password == "" {
			password = username
		}
	}
	if protocol == "vless" {
		node.Setting["uuid"] = username
		if flow := u.Query().Get("flow"); flow != "" {
			node.Setting["flow"] = flow
		}
	} else {
		node.Setting["password"] = password
	}
	applyURLTransport(&node, u)
	if protocol == "hysteria2" {
		if obfs := u.Query().Get("obfs"); obfs != "" && obfs != "none" {
			node.Setting["obfs"] = map[string]interface{}{"type": obfs, "password": u.Query().Get("obfs-password")}
		}
	}
	return node, nil
}

func parseSS(u *url.URL) (Node, error) {
	port, _ := strconv.Atoi(u.Port())
	if u.Hostname() == "" || port == 0 {
		return Node{}, fmt.Errorf("missing endpoint")
	}
	user := u.User.Username()
	password, _ := u.User.Password()
	if password == "" {
		decoded, err := decodeBase64(user)
		if err != nil {
			return Node{}, err
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return Node{}, fmt.Errorf("invalid ss userinfo")
		}
		user, password = parts[0], parts[1]
	}
	return Node{Protocol: "shadowsocks", Tag: fragment(u), Setting: map[string]interface{}{"method": user, "password": password}, Transport: Transport{Address: u.Hostname(), Port: port, Protocol: "tcp", TLSType: "none", TLS: map[string]interface{}{}, Setting: map[string]interface{}{}}}, nil
}

func parseVMess(raw string) (Node, error) {
	payload, err := decodeBase64(strings.TrimPrefix(raw, "vmess://"))
	if err != nil {
		return Node{}, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return Node{}, err
	}
	port := intValue(data["port"])
	if port == 0 {
		port, _ = strconv.Atoi(stringValue(data["port"]))
	}
	node := Node{Protocol: "vmess", Tag: stringValue(data["ps"]), Setting: map[string]interface{}{"uuid": data["id"], "alter_id": intValue(data["aid"]), "security": stringValue(data["scy"])}, Transport: Transport{Address: stringValue(data["add"]), Port: port, Protocol: stringValue(data["net"]), TLSType: "none", TLS: map[string]interface{}{}, Setting: map[string]interface{}{}}}
	if node.Setting["security"] == "" {
		node.Setting["security"] = "auto"
	}
	if node.Transport.Protocol == "" {
		node.Transport.Protocol = "tcp"
	}
	if node.Transport.Protocol == "ws" {
		path := stringValue(data["path"])
		if path == "" {
			path = "/"
		}
		node.Transport.Setting["path"] = path
		if host := stringValue(data["host"]); host != "" {
			node.Transport.Setting["headers"] = map[string]interface{}{"Host": host}
		}
	}
	if stringValue(data["tls"]) == "tls" {
		node.Transport.TLSType = "tls"
		node.Transport.TLS["enabled"] = true
		if sni := stringValue(data["sni"]); sni != "" {
			node.Transport.TLS["server_name"] = sni
		}
		if fp := stringValue(data["fp"]); fp != "" {
			node.Transport.TLS["utls"] = map[string]interface{}{"enabled": true, "fingerprint": fp}
		}
	}
	return node, nil
}

func applyURLTransport(node *Node, u *url.URL) {
	query := u.Query()
	security := query.Get("security")
	if security == "none" {
		node.Transport.TLSType = "none"
		node.Transport.TLS = map[string]interface{}{}
	}
	if sni := query.Get("sni"); sni != "" {
		node.Transport.TLS["server_name"] = sni
	}
	if insecure := query.Get("insecure"); insecure == "1" || insecure == "true" {
		node.Transport.TLS["insecure"] = true
	}
	if query.Get("type") == "ws" || query.Get("type") == "httpupgrade" {
		node.Transport.Protocol = "ws"
		node.Transport.Setting["path"] = query.Get("path")
		if host := query.Get("host"); host != "" {
			node.Transport.Setting["headers"] = map[string]interface{}{"Host": host}
		}
	}
	if security == "reality" {
		node.Transport.TLSType = "reality"
		node.Transport.TLS["reality"] = map[string]interface{}{"enabled": true, "public_key": query.Get("pbk"), "short_id": query.Get("sid")}
	}
	if node.Transport.TLSType != "none" {
		node.Transport.TLS["enabled"] = true
	}
}
func fragment(u *url.URL) string {
	value, _ := url.QueryUnescape(strings.TrimPrefix(u.Fragment, "#"))
	return value
}

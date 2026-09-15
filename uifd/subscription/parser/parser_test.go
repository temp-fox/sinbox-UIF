package parser

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParseUIFRaw(t *testing.T) {
	nodes, err := ParseUIFRaw(fixture(t, "uif.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].Protocol != "trojan" || nodes[0].Transport.Port != 443 || nodes[0].Fingerprint == "" {
		t.Fatalf("unexpected UIF nodes: %#v", nodes)
	}
}

func TestParseSingBox(t *testing.T) {
	nodes, err := ParseSingBox(fixture(t, "singbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].Protocol != "vmess" || nodes[0].Transport.Protocol != "ws" || nodes[0].Transport.TLSType != "tls" {
		t.Fatalf("unexpected sing-box nodes: %#v", nodes)
	}
}

func TestParseClash(t *testing.T) {
	nodes, err := ParseClash(fixture(t, "clash.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 || nodes[0].Protocol != "trojan" || nodes[0].Transport.Setting["path"] != "/proxy" {
		t.Fatalf("unexpected Clash nodes: %#v", nodes)
	}
}

// Clash 节点不能把 cipher/alterId/udp 等 clash 独有字段泄漏进 Setting：
// 前端会把 Setting 原样展开成 sing-box outbound 顶层字段，未知字段会让
// sing-box 以 FATAL "unknown field" 拒绝启动。
func TestParseClashDoesNotLeakClashFields(t *testing.T) {
	input := `proxies:
  - name: vmess-node
    type: vmess
    server: example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
    alterId: 0
    cipher: auto
    udp: true
    tls: true
  - name: ss-node
    type: ss
    server: example.com
    port: 8388
    cipher: aes-128-gcm
    password: secret
    udp: true
`
	nodes, err := ParseClash(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	for _, node := range nodes {
		for _, banned := range []string{"cipher", "alterId", "alter_id_legacy", "udp", "network", "skip-cert-verify", "ws-opts", "grpc-opts"} {
			if _, exists := node.Setting[banned]; exists {
				t.Fatalf("node %q leaked banned field %q into Setting: %#v", node.Tag, banned, node.Setting)
			}
		}
	}
	if nodes[0].Setting["uuid"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("vmess uuid not normalized: %#v", nodes[0].Setting)
	}
	if nodes[0].Setting["alter_id"] == nil {
		t.Fatalf("vmess alter_id missing: %#v", nodes[0].Setting)
	}
	if nodes[1].Setting["method"] != "aes-128-gcm" || nodes[1].Setting["password"] != "secret" {
		t.Fatalf("ss fields not normalized: %#v", nodes[1].Setting)
	}
}

// Sing-box 解析必须保留 transport.TLS["enabled"]=true：前端把整个 tls
// 对象原样写进 sing-box 配置，hysteria2/tuic/trojan 等强制 TLS 的协议
// 缺少 enabled 会让内核以 "TLS required" 拒绝启动。
func TestParseSingBoxPreservesTLSEnabled(t *testing.T) {
	input := `{"outbounds":[{"type":"hysteria2","tag":"hy2","server":"example.com","server_port":443,"password":"secret","tls":{"enabled":true,"insecure":true}}]}`
	nodes, err := ParseSingBox(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Transport.TLSType != "tls" {
		t.Fatalf("expected tls transport, got %#v", nodes[0].Transport)
	}
	if enabled, _ := nodes[0].Transport.TLS["enabled"].(bool); !enabled {
		t.Fatalf("sing-box tls.enabled must be preserved: %#v", nodes[0].Transport.TLS)
	}
}

// 强制 TLS 协议（hysteria2/tuic/trojan）即使订阅源省略 tls 字段，
// 也必须被标记为 tls_type=tls 并补齐 enabled=true，否则内核 "TLS required"。
func TestParseSingBoxForcesTLSForRequiredProtocols(t *testing.T) {
	input := `{"outbounds":[
		{"type":"hysteria2","tag":"hy2","server":"example.com","server_port":443,"password":"secret"},
		{"type":"tuic","tag":"tuic","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111","password":"secret"},
		{"type":"vmess","tag":"vmess","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"}
	]}`
	nodes, err := ParseSingBox(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	for _, node := range nodes {
		switch node.Protocol {
		case "hysteria2", "tuic":
			if node.Transport.TLSType != "tls" {
				t.Fatalf("%s 应强制 tls，实际 %s: %#v", node.Protocol, node.Transport.TLSType, node.Transport)
			}
			if enabled, _ := node.Transport.TLS["enabled"].(bool); !enabled {
				t.Fatalf("%s 缺 tls.enabled=true: %#v", node.Protocol, node.Transport.TLS)
			}
		case "vmess":
			if node.Transport.TLSType != "none" {
				t.Fatalf("vmess 无 tls 字段不应被强制 tls，实际 %s", node.Transport.TLSType)
			}
		}
	}
}

func TestParseV2rayN(t *testing.T) {
	nodes, err := ParseV2rayN(fixture(t, "v2rayn.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 4 || nodes[0].Protocol != "shadowsocks" || nodes[1].Setting["password"] != "secret" || nodes[2].Transport.Protocol != "ws" || nodes[3].Protocol != "hysteria2" {
		t.Fatalf("unexpected V2rayN nodes: %#v", nodes)
	}
}

func TestParseV2rayNBase64(t *testing.T) {
	raw := fixture(t, "v2rayn.txt")
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	nodes, err := ParseV2rayN(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 4 {
		t.Fatalf("expected four decoded nodes, got %d", len(nodes))
	}
}

func TestParseDetectsFormats(t *testing.T) {
	cases := []struct{ name, file, format string }{{"uif", "uif.txt", "uif"}, {"sing", "singbox.json", "sing-box"}, {"clash", "clash.yaml", "clash"}, {"v2rayn", "v2rayn.txt", "v2rayn"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse(fixture(t, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if result.Format != tc.format || len(result.Nodes) == 0 {
				t.Fatalf("unexpected result: %#v", result)
			}
		})
	}
}

func TestParseReportsSkippedCandidates(t *testing.T) {
	input := "proxies:\n  - name: supported\n    type: trojan\n    server: example.com\n    port: 443\n    password: secret\n  - name: unsupported\n    type: wireguard\n    server: example.com\n    port: 51820\n"
	result, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Format != "clash" || len(result.Nodes) != 1 || result.Skipped != 1 {
		t.Fatalf("unexpected parse summary: %#v", result)
	}
}

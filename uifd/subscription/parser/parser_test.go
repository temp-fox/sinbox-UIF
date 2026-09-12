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

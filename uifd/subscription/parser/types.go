package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Node is the normalized subscription node shared by all supported formats.
// Setting and transport settings intentionally use interface values so fields
// unknown to this package remain serializable and available to callers.
type Node struct {
	Protocol    string                 `json:"protocol"`
	Tag         string                 `json:"tag,omitempty"`
	Transport   Transport              `json:"transport"`
	Setting     map[string]interface{} `json:"setting,omitempty"`
	Fingerprint string                 `json:"fingerprint,omitempty"`
}

// Transport describes the server endpoint and its stream/TLS settings.
type Transport struct {
	Address   string                 `json:"address,omitempty"`
	Port      int                    `json:"port,omitempty"`
	Protocol  string                 `json:"protocol"`
	TLSType   string                 `json:"tls_type"`
	TLS       map[string]interface{} `json:"tls,omitempty"`
	Setting   map[string]interface{} `json:"setting,omitempty"`
	Multiplex map[string]interface{} `json:"multiplex,omitempty"`
}

// ParseResult is useful to callers that want to retain the detected input
// format alongside the normalized nodes.
type ParseResult struct {
	Format  string `json:"format"`
	Nodes   []Node `json:"nodes"`
	Skipped int    `json:"skipped"`
}

// Fingerprint returns a stable identity for a node. Human labels (Tag) and
// this field itself are intentionally excluded, so a renamed node still has
// the same identity. The same defaults used by the parser are applied here so
// a hand-built Node and a parsed Node get the same identity.
func Fingerprint(node Node) string {
	node = normalizeNode(node)
	identity := struct {
		Protocol  string                 `json:"protocol"`
		Transport Transport              `json:"transport"`
		Setting   map[string]interface{} `json:"setting,omitempty"`
	}{node.Protocol, node.Transport, node.Setting}
	data, _ := json.Marshal(identity)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeNode(node Node) Node {
	if node.Transport.Protocol == "" {
		node.Transport.Protocol = "tcp"
	}
	if node.Transport.TLSType == "" {
		node.Transport.TLSType = "none"
	}
	if node.Transport.Setting == nil {
		node.Transport.Setting = map[string]interface{}{}
	}
	if node.Setting == nil {
		node.Setting = map[string]interface{}{}
	}
	if node.Transport.TLS == nil {
		node.Transport.TLS = map[string]interface{}{}
	}
	if node.Transport.Multiplex == nil {
		node.Transport.Multiplex = map[string]interface{}{}
	}
	return node
}

func withFingerprint(nodes []Node) []Node {
	for i := range nodes {
		nodes[i] = normalizeNode(nodes[i])
		nodes[i].Fingerprint = Fingerprint(nodes[i])
	}
	return nodes
}

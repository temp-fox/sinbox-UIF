package subscription

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestSingBoxProbeExecutorRejectsMissingCoreClearly(t *testing.T) {
	result, err := NewSingBoxProbeExecutor(t.TempDir()+"/missing-sing-box").Probe(context.Background(), ProbeTarget{})
	if err == nil || result.Success || !strings.Contains(result.Error, "core is unavailable") {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestSingBoxProbeExecutorRunsIsolatedCoreAndDelay(t *testing.T) {
	executor := NewSingBoxProbeExecutor(os.Args[0])
	executor.ReadinessTimeout = time.Second
	executor.CommandFactory = func(ctx context.Context, configPath string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSingBoxProbeHelper", "--")
		cmd.Env = append(os.Environ(), "UIF_SINGBOX_PROBE_HELPER=1", "UIF_SINGBOX_PROBE_CONFIG="+configPath)
		return cmd
	}
	node := SnapshotNode{}
	node.Protocol = "vmess"
	node.Transport.Address = "node.example"
	node.Transport.Port = 443
	node.Transport.Protocol = "tcp"
	node.Transport.TLSType = "tls"
	node.Transport.TLS = map[string]interface{}{"server_name": "node.example"}
	node.Setting = map[string]interface{}{"uuid": "test-id"}
	result, err := executor.Probe(context.Background(), ProbeTarget{URL: "https://example.com/health", Node: node})
	if err != nil || !result.Success || result.DelayMs <= 0 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

// TestSingBoxProbeHelper is a fake executable core. It only runs in the child
// process created by TestSingBoxProbeExecutorRunsIsolatedCoreAndDelay.
func TestSingBoxProbeHelper(t *testing.T) {
	if os.Getenv("UIF_SINGBOX_PROBE_HELPER") != "1" {
		return
	}
	configPath := os.Getenv("UIF_SINGBOX_PROBE_CONFIG")
	data, err := os.ReadFile(configPath)
	if err != nil {
		os.Exit(2)
	}
	var config struct {
		Experimental struct {
			ClashAPI struct {
				ExternalController string `json:"external_controller"`
			} `json:"clash_api"`
		} `json:"experimental"`
	}
	if json.Unmarshal(data, &config) != nil || config.Experimental.ClashAPI.ExternalController == "" {
		os.Exit(3)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"fake"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/delay") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"delay":7}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})}
	listener, err := net.Listen("tcp", config.Experimental.ClashAPI.ExternalController)
	if err != nil {
		os.Exit(4)
	}
	_ = server.Serve(listener)
}

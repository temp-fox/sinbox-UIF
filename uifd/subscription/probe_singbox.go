package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SingBoxProbeExecutor starts an isolated sing-box process for each node and
// uses its Clash API delay endpoint. The process is never shared with UIF's
// long-running core, so probes cannot alter WT/TUN/nft state.
type SingBoxProbeExecutor struct {
	CorePath         string
	Secret           string
	ReadinessTimeout time.Duration
	HTTPClient       *http.Client
	CommandFactory   func(context.Context, string) *exec.Cmd
	OutboundTag      string
}

func NewSingBoxProbeExecutor(corePath string) *SingBoxProbeExecutor {
	return &SingBoxProbeExecutor{CorePath: corePath, ReadinessTimeout: 5 * time.Second, OutboundTag: "uif-probe-node"}
}

func (e *SingBoxProbeExecutor) Probe(ctx context.Context, target ProbeTarget) (ProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil {
		return failedProbe("sing-box probe executor is nil")
	}
	if strings.TrimSpace(e.CorePath) == "" {
		return failedProbe("sing-box probe core path is empty")
	}
	info, err := os.Stat(e.CorePath)
	if err != nil {
		return failedProbe(fmt.Sprintf("sing-box probe core is unavailable: %v", err))
	}
	if !info.Mode().IsRegular() {
		return failedProbe("sing-box probe core path is not a regular file")
	}
	if err := ctx.Err(); err != nil {
		return failedProbe(err.Error())
	}
	probeURL := strings.TrimSpace(target.URL)
	if probeURL == "" {
		probeURL = DefaultProbeURL
	}
	parsedURL, err := url.Parse(probeURL)
	if err != nil || parsedURL.Host == "" || parsedURL.User != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return failedProbe("sing-box probe target URL must be an http(s) URL without credentials")
	}
	outboundTag := e.OutboundTag
	if outboundTag == "" {
		outboundTag = "uif-probe-node"
	}
	outbound, err := singBoxProbeOutbound(target.Node, outboundTag)
	if err != nil {
		return failedProbe(err.Error())
	}
	apiAddr, err := freeLoopbackAddress()
	if err != nil {
		return failedProbe("allocate sing-box probe API port: " + err.Error())
	}
	config := map[string]interface{}{
		"log": map[string]interface{}{"level": "error"},
		"experimental": map[string]interface{}{"clash_api": map[string]interface{}{
			"external_controller": apiAddr,
			"secret":              e.Secret,
		}},
		"outbounds": []interface{}{outbound},
		"route":     map[string]interface{}{"final": outboundTag},
	}
	configFile, err := os.CreateTemp("", "uif-singbox-probe-*.json")
	if err != nil {
		return failedProbe("create sing-box probe config: " + err.Error())
	}
	configPath := configFile.Name()
	defer os.Remove(configPath)
	if err := configFile.Chmod(0600); err != nil {
		configFile.Close()
		return failedProbe("protect sing-box probe config: " + err.Error())
	}
	encoder := json.NewEncoder(configFile)
	if err := encoder.Encode(config); err != nil {
		configFile.Close()
		return failedProbe("write sing-box probe config: " + err.Error())
	}
	if err := configFile.Close(); err != nil {
		return failedProbe("close sing-box probe config: " + err.Error())
	}

	cmd := e.command(ctx, configPath)
	if cmd == nil {
		return failedProbe("sing-box probe command is nil")
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return failedProbe("start sing-box probe core: " + err.Error())
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	cleanup := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitDone:
		case <-time.After(2 * time.Second):
		}
	}
	defer cleanup()

	baseURL := "http://" + apiAddr
	readiness := e.ReadinessTimeout
	if readiness <= 0 {
		readiness = 5 * time.Second
	}
	readyCtx, cancel := context.WithTimeout(ctx, readiness)
	defer cancel()
	if err := waitClashAPI(readyCtx, e.client(), baseURL, e.Secret, waitDone); err != nil {
		if ctx.Err() != nil {
			return failedProbe(ctx.Err().Error())
		}
		message := err.Error()
		if output := strings.TrimSpace(stderr.String()); output != "" {
			message += ": " + output
		}
		return failedProbe(message)
	}

	start := time.Now()
	delayEndpoint := baseURL + "/proxies/" + url.PathEscape(outboundTag) + "/delay?url=" + url.QueryEscape(parsedURL.String()) + "&timeout=" + strconv.FormatInt(maxProbeTimeout(ctx).Milliseconds(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, delayEndpoint, nil)
	if err != nil {
		return failedProbe("create sing-box delay request: " + err.Error())
	}
	if strings.TrimSpace(e.Secret) != "" {
		req.Header.Set("Authorization", "Bearer "+e.Secret)
	}
	resp, err := e.client().Do(req)
	if err != nil {
		return failedProbe("sing-box delay request failed: " + err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return failedProbe(fmt.Sprintf("sing-box delay API returned %s: %s", resp.Status, strings.TrimSpace(string(body))))
	}
	var payload struct {
		Delay int `json:"delay"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Delay <= 0 {
		return failedProbe("sing-box delay API returned no positive delay")
	}
	if elapsed := int(time.Since(start) / time.Millisecond); elapsed > payload.Delay {
		payload.Delay = elapsed
	}
	return ProbeResult{Success: true, DelayMs: payload.Delay}, nil
}

func failedProbe(message string) (ProbeResult, error) {
	return ProbeResult{Success: false, Error: message}, errors.New(message)
}

func (e *SingBoxProbeExecutor) command(ctx context.Context, configPath string) *exec.Cmd {
	if e.CommandFactory != nil {
		return e.CommandFactory(ctx, configPath)
	}
	return exec.Command(e.CorePath, "run", "-c", configPath)
}

func (e *SingBoxProbeExecutor) client() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func maxProbeTimeout(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 {
			return remaining
		}
	}
	return 10 * time.Second
}

func freeLoopbackAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	address := listener.Addr().String()
	return address, listener.Close()
}

func waitClashAPI(ctx context.Context, client *http.Client, baseURL, secret string, processDone <-chan error) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/version", nil)
		if err == nil {
			if strings.TrimSpace(secret) != "" {
				req.Header.Set("Authorization", "Bearer "+secret)
			}
			resp, requestErr := client.Do(req)
			if requestErr == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					return nil
				}
			}
		}
		select {
		case err := <-processDone:
			if err == nil {
				return errors.New("sing-box probe core exited before Clash API readiness")
			}
			return fmt.Errorf("sing-box probe core exited before Clash API readiness: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func singBoxProbeOutbound(node SnapshotNode, tag string) (map[string]interface{}, error) {
	if strings.TrimSpace(node.Protocol) == "" {
		return nil, errors.New("sing-box probe node protocol is empty")
	}
	if strings.TrimSpace(node.Transport.Address) == "" || node.Transport.Port <= 0 || node.Transport.Port > 65535 {
		return nil, errors.New("sing-box probe node transport address/port is missing or invalid")
	}
	outbound := make(map[string]interface{}, len(node.Setting)+8)
	for key, value := range node.Setting {
		outbound[key] = value
	}
	outbound["type"] = node.Protocol
	outbound["tag"] = tag
	outbound["server"] = node.Transport.Address
	outbound["server_port"] = node.Transport.Port
	if node.Transport.Protocol != "" && node.Transport.Protocol != "tcp" {
		transport := map[string]interface{}{"type": node.Transport.Protocol}
		for key, value := range node.Transport.Setting {
			transport[key] = value
		}
		outbound["transport"] = transport
	}
	if node.Transport.TLSType != "" && node.Transport.TLSType != "none" {
		tls := map[string]interface{}{"enabled": true}
		for key, value := range node.Transport.TLS {
			tls[key] = value
		}
		outbound["tls"] = tls
	}
	if len(node.Transport.Multiplex) > 0 {
		outbound["multiplex"] = node.Transport.Multiplex
	}
	return outbound, nil
}

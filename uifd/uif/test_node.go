package uif

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type TestNodeRes struct {
	Status int    `json:"status"`
	Msg    string `json:"msg,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Delay  int    `json:"delay"`
}

type TestNodeReq struct {
	Config    string   `json:"config,omitempty"`
	Tags      []string `json:"tags"` // or domain list
	IsIpInfo  bool     `json:"is_ip_info"`
	TimeoutMs int      `json:"timeout_ms,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func delayTimeoutMs(requested int) int {
	if requested <= 0 {
		requested = 10000
	}
	if requested < 3000 {
		requested = 3000
	}
	if requested > 120000 {
		requested = 120000
	}
	return requested
}

func getDelayDirect(apiAddress string, tag string, timeoutMs int) (string, error) {
	delayURL := fmt.Sprintf("%s/proxies/%s/delay?timeout=%d", apiAddress, url.PathEscape(tag), timeoutMs)
	client := &http.Client{Timeout: time.Duration(timeoutMs+5000) * time.Millisecond}
	resp, err := client.Get(delayURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, readErr := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	if readErr != nil {
		return body, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if body != "" {
			return body, fmt.Errorf("delay HTTP status: %s: %s", resp.Status, body)
		}
		return body, fmt.Errorf("delay HTTP status: %s", resp.Status)
	}
	return body, nil
}

func waitTestCoreReady(apiAddress string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 800 * time.Millisecond}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(apiAddress + "/proxies")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("clash api status: %s", resp.Status)
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("test core clash api not ready: %w", lastErr)
	}
	return fmt.Errorf("test core clash api not ready")
}

func TestMultipleNode(conn *websocket.Conn) error {
	msgJson := &TestNodeReq{}
	err := conn.ReadJSON(msgJson)
	if err != nil {
		WriteLog(err.Error())
		return err
	}

	// init config
	portInt, err := GetUnusedPort()
	if err != nil {
		WriteLog(err.Error())
		conn.WriteJSON(&TestNodeRes{Status: 1, Msg: "failed to GetUnusedPort()"})
		return err
	}
	apiPort := strconv.Itoa(portInt)
	msgJson.Config = strings.ReplaceAll(msgJson.Config, "111111", apiPort)
	configPath := GetWorkSpace() + "/" + apiPort + "_" + randString(16) + ".json"
	os.WriteFile(configPath, []byte(msgJson.Config), 0644) // Create new if it is not exist
	defer os.Remove(configPath)
	apiAddress := "http://127.0.0.1:" + apiPort

	// run it
	testProcess, err := RunTestCore2(configPath, apiPort)
	if err != nil {
		WriteLog(err.Error())
		conn.WriteJSON(&TestNodeRes{Status: 1, Msg: err.Error()})
		return err
	}
	defer func() {
		testProcess.Process.Kill()
		testProcess.Wait()
	}()
	if err := waitTestCoreReady(apiAddress, 20*time.Second); err != nil {
		WriteLog(err.Error())
		conn.WriteJSON(&TestNodeRes{Status: 1, Msg: err.Error()})
		return err
	}

	// test it
	timeoutMs := delayTimeoutMs(msgJson.TimeoutMs)
	WriteLog(fmt.Sprintf("delay test start: tags=%d timeout_ms=%d api=%s", len(msgJson.Tags), timeoutMs, apiAddress))
	var writeMux sync.Mutex
	var wg sync.WaitGroup
	for _, v := range msgJson.Tags {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			res := &TestNodeRes{Tag: tag, Delay: 0, Status: 0, Msg: ""}
			start := time.Now()
			var body string
			var err error
			if msgJson.IsIpInfo {
				body, _, err = HTTPWithProxyPort(tag, apiAddress, "", "", "")
			} else {
				body, err = getDelayDirect(apiAddress, tag, timeoutMs)
			}
			if err != nil {
				res.Msg = err.Error()
				res.Status = 2
			} else {
				res.Msg = body
			}
			if body != "" {
				json.Unmarshal([]byte(body), &res)
			}
			WriteLog(fmt.Sprintf("delay test result: tag=%s status=%d delay=%d elapsed_ms=%d msg=%s", tag, res.Status, res.Delay, time.Since(start).Milliseconds(), res.Msg))
			writeMux.Lock()
			defer writeMux.Unlock()
			conn.WriteJSON(res)
		}(v)
	}
	wg.Wait()
	WriteLog(fmt.Sprintf("delay test done: tags=%d timeout_ms=%d", len(msgJson.Tags), timeoutMs))
	return nil
}

func RunTestCore2(path string, port string) (*exec.Cmd, error) {
	testProcess := exec.Command(GetCorePath(), "run", "-c", path)
	testProcess.Dir = GetWorkSpace()
	ProcessSet(testProcess)

	pipe, err := testProcess.StderrPipe()
	if err != nil {
		return nil, err
	}
	go SaveLog(pipe)

	err = testProcess.Start()
	if err != nil {
		return nil, err
	}

	return testProcess, nil
}

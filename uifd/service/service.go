package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/elevate"
	"github.com/gorilla/websocket"
	"github.com/uif/uifd/subscription"
	"github.com/uif/uifd/subscription/parser"
	"github.com/uif/uifd/uif"
)

var serviceMutext sync.Mutex
var subscriptionJobs = subscription.NewManager()
var subscriptionScheduler = subscription.NewScheduler(nil, subscription.WithJobManager(subscriptionJobs))

var APIServer http.Server
var WebServer http.Server

type ConnectInfo struct {
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,string,omitempty"`
	StartTime string `json:"startTime,string,omitempty"`
}

func BuildAllowedDomain(r *http.Request) string {
	if uif.IsNeedKey() {
		return "*"
	}
	domain := r.Header.Get("Origin")
	if domain == "" {
		domain = r.Header.Get("Referer")
		if domain == "" {
			return ""
		}
	}
	url, err := url.Parse(domain)
	if err != nil {
		return ""
	}
	trustedDomain := []string{"uiforfreedom.github.io", "127.0.0.1", "localhost", "ui4freedom.org", "192.168.0.230"}
	for _, v := range trustedDomain {
		if strings.HasSuffix(url.Hostname(), v) {
			return "*"
		}
	}
	return ""
}

func CheckPassword(w http.ResponseWriter, r *http.Request) bool {
	isPass := false
	allowedDomain := BuildAllowedDomain(r)
	// Protection.
	defer func() {
		w.Header().Set("content-type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", allowedDomain)
		w.Header().Set("Access-Control-Allow-Methods", "POST,OPTIONS,GET")
		if !isPass {
			return
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "accept,x-requested-with,Content-Type,Extra-Info")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Extra-Info")
	}()

	token := r.URL.Query().Get("key")
	if token == "" {
		err := r.ParseForm()
		if err == nil {
			token = r.FormValue("key")
		}
	}

	if uif.IsNeedKey() {
		if token != uif.GetKey() {
			if token != "" {
				time.Sleep(3 * time.Second) // security.
			}
			fmt.Fprint(w, "{\"status\": -1}") // empty means no
		} else {
			isPass = true
		}
	} else {
		// WT 下游模式由 WT/Droidspaces 网络边界限制访问，UIF 不要求密码。
		isPass = true
	}
	return isPass
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestNode(w http.ResponseWriter, r *http.Request) {
	serviceMutext.Lock()
	if !CheckPassword(w, r) {
		serviceMutext.Unlock()
		return
	}
	serviceMutext.Unlock()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		uif.WriteLog(err.Error())
		return
	}
	defer conn.Close()
	uif.TestMultipleNode(conn)
}

func TryOpenPort(i string) {
	var inboudPorts []string
	json.Unmarshal([]byte(i), &inboudPorts)
	for _, v := range inboudPorts {
		uif.AllowPort(v, "tcp")
		uif.AllowPort(v, "udp")
	}
}

func loadSubscriptionSpec(id, dst string) (subscription.SubscriptionSpec, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(dst) == "" {
		return subscription.SubscriptionSpec{}, fmt.Errorf("subscription id and url are required")
	}
	root := uif.GetWorkSpace()
	if root == "" {
		return subscription.SubscriptionSpec{}, fmt.Errorf("uif workspace is empty")
	}
	safe := strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(id)
	return subscription.SubscriptionSpec{ID: id, URL: dst, SnapshotPath: filepath.Join(root, "subscriptions", safe+".snapshot.json"), Policy: subscription.SchedulePolicy{UpdateMode: "merge", MissingGraceRuns: 3, MinKeep: 2}}, nil
}

func refreshSubscriptionOnce(ctx context.Context, spec subscription.SubscriptionSpec) (string, error) {
	result, err := subscription.Refresh(ctx, spec, func(fetchCtx context.Context, source string) (string, string, error) {
		return uif.HTTPGetDirectContext(fetchCtx, source)
	})
	if err != nil {
		return "", err
	}
	return subscription.MarshalResult(result), nil
}

func validateSubscriptionResult(result string) (string, error) {
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("subscription response body is empty")
	}
	return result, nil
}

func subscriptionSnapshotPath(requested, subscriptionID string) (string, error) {
	root, err := filepath.Abs(uif.GetWorkSpace())
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(requested) == "" {
		name := subscriptionID
		if name == "" {
			name = "default"
		}
		name = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, name)
		return filepath.Join(root, "subscriptions", name+".snapshot.json"), nil
	}
	candidate := requested
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	candidate, err = filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("snapshot path must stay under workspace")
	}
	return candidate, nil
}

func subscriptionSource(ctx context.Context, source, raw string) (string, string, error) {
	if strings.TrimSpace(raw) != "" {
		return raw, "", nil
	}
	if strings.TrimSpace(source) == "" {
		return "", "", fmt.Errorf("subscription source is empty")
	}
	result, extraInfo, err := uif.HTTPGetDirectContext(ctx, source)
	if err != nil {
		return "", "", err
	}
	return result, extraInfo, nil
}

func parseAndSaveSubscriptionSnapshot(result, extraInfo, snapshotPath string) (string, error) {
	result, err := validateSubscriptionResult(result)
	if err != nil {
		return "", err
	}
	parsed, err := parser.Parse(result)
	if err != nil {
		return "", fmt.Errorf("parse subscription: %w", err)
	}
	if len(parsed.Nodes) == 0 {
		return "", fmt.Errorf("subscription contains no nodes")
	}
	old, err := subscription.Load(snapshotPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("load subscription snapshot: %w", err)
	}
	options := subscription.DefaultMergeOptions()
	merged, err := subscription.ApplyParse(old, parsed, nil, options)
	if err != nil {
		return "", err
	}
	summary := subscription.SummarizeMerge(old.Nodes, parsed.Nodes, merged.Nodes)
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0700); err != nil {
		return "", fmt.Errorf("create snapshot directory: %w", err)
	}
	if err := subscription.SaveAtomicJSON(snapshotPath, merged); err != nil {
		return "", fmt.Errorf("save subscription snapshot: %w", err)
	}
	envelope, _ := json.Marshal(map[string]interface{}{
		"body": result, "extra_info": extraInfo,
		"parse_summary":    map[string]interface{}{"format": parsed.Format, "nodes": len(parsed.Nodes), "skipped": parsed.Skipped},
		"snapshot_summary": summary,
	})
	return string(envelope), nil
}

// refreshSubscription executes one complete fetch/parse/merge/publish cycle.
// It is shared by the legacy /subscriptions/job endpoint and the scheduler so
// both paths have identical failure and snapshot safety semantics.
func refreshSubscriptionResult(ctx context.Context, spec subscription.SubscriptionSpec) (string, error) {
	if strings.TrimSpace(spec.SnapshotPath) == "" {
		path, err := subscriptionSnapshotPath("", spec.ID)
		if err != nil {
			return "", err
		}
		spec.SnapshotPath = path
	}
	result, err := subscription.Refresh(ctx, spec, func(fetchCtx context.Context, source string) (string, string, error) {
		return subscriptionSource(fetchCtx, source, "")
	})
	if err != nil {
		return "", err
	}
	return subscription.MarshalResult(result), nil
}

func refreshSubscription(ctx context.Context, spec subscription.SubscriptionSpec) error {
	_, err := refreshSubscriptionResult(ctx, spec)
	return err
}

func subscriptionSchedulerSpecs() ([]subscription.SubscriptionSpec, error) {
	config, err := uif.ReadUIFConfigJson()
	if err != nil {
		return nil, err
	}
	raw, ok := config["subscribe"]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var items []struct {
		ID           string                      `json:"id"`
		URL          string                      `json:"url"`
		Source       string                      `json:"source"`
		Data         string                      `json:"data"`
		SnapshotPath string                      `json:"snapshot_path"`
		Policy       subscription.SchedulePolicy `json:"policy"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse subscription config: %w", err)
	}
	specs := make([]subscription.SubscriptionSpec, 0, len(items))
	for index, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = fmt.Sprintf("legacy-subscription-%d", index)
		}
		source := item.URL
		if source == "" {
			source = item.Source
		}
		if source == "" {
			source = item.Data
		}
		snapshotPath, err := subscriptionSnapshotPath(item.SnapshotPath, id)
		if err != nil {
			return nil, err
		}
		specs = append(specs, subscription.SubscriptionSpec{ID: id, URL: source, Source: source, SnapshotPath: snapshotPath, Policy: item.Policy})
	}
	return specs, nil
}

func startSubscriptionScheduler() {
	subscriptionScheduler = subscription.NewScheduler(refreshSubscription, subscription.WithJobManager(subscriptionJobs))
	if specs, err := subscriptionSchedulerSpecs(); err == nil {
		if err := subscriptionScheduler.Reload(specs); err == nil {
			if err := subscriptionScheduler.Start(); err != nil {
				uif.WriteLog("subscription scheduler start failed: " + err.Error())
			}
		} else {
			uif.WriteLog("subscription scheduler config failed: " + err.Error())
		}
	} else {
		uif.WriteLog("subscription scheduler load failed: " + err.Error())
	}
}

func parseSubscriptionResult(result, extraInfo string) (string, error) {
	result, err := validateSubscriptionResult(result)
	if err != nil {
		return "", err
	}
	parsed, err := parser.Parse(result)
	if err != nil {
		return "", fmt.Errorf("parse subscription: %w", err)
	}
	if len(parsed.Nodes) == 0 {
		return "", fmt.Errorf("subscription contains no nodes")
	}
	envelope, _ := json.Marshal(map[string]interface{}{
		"body": result, "extra_info": extraInfo,
		"parse_summary": map[string]interface{}{"format": parsed.Format, "nodes": len(parsed.Nodes), "skipped": parsed.Skipped},
	})
	return string(envelope), nil
}

func Service(w http.ResponseWriter, r *http.Request) {
	// {{{
	serviceMutext.Lock()
	if !CheckPassword(w, r) {
		serviceMutext.Unlock()
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := r.URL.Path
	res := "{}"

	if path == "/get_uif_config" {
		res = uif.ReadUIFConfig()
	} else if path == "/save_uif_config" {
		config := r.FormValue("config")
		shareConfig := r.FormValue("shareConfig")
		uif.SaveUIFConfig(config)
		uif.SaveShareConfig(shareConfig)
		uif.SetCoreAutoRestartTicker()
	} else if path == "/run_core" {
		TryOpenPort(r.FormValue("inboudPorts"))
		config := r.FormValue("config")
		uif.SaveCoreConfig(config)
		if uif.IsMacos() && uif.IsUseTun() {
			uif.SetOsDNS(false, "")
		}
		uif.RunCore()
	} else if path == "/close_core" {
		uif.CloseCore()
	} else if path == "/auto_startup" {
		enable := r.FormValue("isInstall")
		if enable == "true" {
			res = uif.SetAutoStartup(true)
		} else {
			res = uif.SetAutoStartup(false)
		}
	} else if path == "/connect" {
		res = uif.GetInfo()
	} else if path == "/close_uif" {
		go CleanAndQuit()
	} else if path == "/proxy_get" || path == "/http_mutiple" {
		serviceMutext.Unlock()
		dst := r.FormValue("dst")
		proxyFirst := r.FormValue("proxy_first") == "true"
		ExtraInfo := ""
		res, ExtraInfo = uif.ProxyGet(dst, proxyFirst, path == "/http_mutiple")
		w.Header().Set("Extra-Info", ExtraInfo)
		fmt.Fprint(w, res)
		return
	} else if path == "/proxy_get2" || path == "/http_with_port" {
		serviceMutext.Unlock()
		ExtraInfo := ""
		res, ExtraInfo, _ = uif.HTTPWithProxyPort(r.FormValue("dst"),
			r.FormValue("http_proxy_port"),
			r.FormValue("authorization"),
			r.FormValue("method"),
			r.FormValue("data"))
		w.Header().Set("Extra-Info", ExtraInfo)
		fmt.Fprint(w, res)
		return
	} else if path == "/ping" {
		serviceMutext.Unlock()
		address := r.FormValue("address")
		res = uif.Ping(address)
		fmt.Fprint(w, res)
		return
	} else if path == "/share" {
		res = uif.ReadUIFShareConfig()
	} else if path == "/update_uif" {
		res = uif.Update()
	} else if path == "/check_update" {
		res = uif.CheckUpdateReq()
	} else if path == "/subscriptions/job" {
		id := r.FormValue("subscription_id")
		kind := r.FormValue("kind")
		source := r.FormValue("source")
		raw := r.FormValue("raw")
		if source == "" && raw == "" {
			source = r.FormValue("dst")
		}
		snapshotPath, pathErr := subscriptionSnapshotPath(r.FormValue("snapshot_path"), id)
		job := subscriptionJobs.StartResult(nil, id, kind, func(ctx context.Context) (string, error) {
			if pathErr != nil {
				return "", pathErr
			}
			if raw != "" {
				return parseAndSaveSubscriptionSnapshot(raw, "", snapshotPath)
			}
			return refreshSubscriptionResult(ctx, subscription.SubscriptionSpec{ID: id, URL: source, Source: source, SnapshotPath: snapshotPath})
		})
		payload, _ := json.Marshal(job)
		res = string(payload)
	} else if path == "/subscriptions/job/status" {
		job := subscriptionJobs.Get(r.FormValue("job_id"))
		if job == nil {
			res = `{"status":-1,"error":"job not found"}`
		} else {
			payload, _ := json.Marshal(job)
			res = string(payload)
		}
		serviceMutext.Unlock()
		fmt.Fprint(w, res)
		return
	}

	fmt.Fprint(w, res)
	serviceMutext.Unlock()
	// }}}
}

func CheckPort() error {
	webPort, err := uif.GetWebAddressPort()
	if err != nil {
		return err
	}
	_, err = uif.TCPPortCheck(webPort)
	if err != nil {
		return err
	}

	apiPort, err := uif.GetAPIAddressPort()
	if err != nil {
		return err
	}
	_, err = uif.TCPPortCheck(apiPort)
	if err != nil {
		return err
	}

	// open port if it is public
	if uif.IsNeedKey() {
		// uif.AllowPort(apiPort, "tcp")
	}
	return nil
}

func RunServer() error {
	web := http.FileServer(http.Dir(uif.GetWebPath()))
	WebServer = http.Server{
		Addr:    uif.GetWebAddress(),
		Handler: web,
	}
	go WebServer.ListenAndServe()

	api := http.NewServeMux()
	api.HandleFunc("/delay", TestNode)
	api.HandleFunc("/", Service)

	APIServer = http.Server{
		Addr:    uif.GetAPIAddress(),
		Handler: api,
	}
	go APIServer.ListenAndServe()
	startSubscriptionScheduler()
	return nil
}

func CloseServer() {
	err := WebServer.Close()
	if err != nil {
		panic(err)
	}
	err = APIServer.Close()
	if err != nil {
		panic(err)
	}
}

func StartupCore() {
	if !uif.IsFirstTime() && !uif.HasFlutter() {
		err := uif.RunCore()
		if err != nil {
			uif.WriteLog(err.Error())
		}
	} else if !uif.HasFlutter() {
		SetQuicLink()
	}

	if uif.IsAutoUpdateUIF() {
		time.Sleep(10 * time.Second) // let core to be ready
		uif.Update()
	}
}

func CheckAndInit() error {
	uif.WriteLog("Password: " + uif.GetKey()) // init
	if err := CheckPort(); err != nil {
		return err
	}
	uif.ParseApiPort()
	return nil
}

func main() {
	err := Entry()
	if err != nil {
		uif.WriteLog(err.Error())
		fmt.Fprintf(os.Stderr, "UIF Error: %v\n", err)
		os.Exit(1)
	}
	if uif.HasFlutter() {
		uif.WaitQuitSnignal()
	} else {
		TrayInit()
	}
	CleanAndQuit()
}

func CheckAction() bool {
	action := ""
	if uif.IsActionExists() {
		rawURL := uif.ReadFile(uif.GetActionPath())
		uif.DeleteFile(uif.GetActionPath())
		if rawURL != "" {
			action = uif.ParseURL(rawURL)
		}
		uif.OpenBrowser("http://" + uif.GetWebAddress() + action)
		return true
	}
	return false
}

func CleanAndQuit() { // close all
	os.WriteFile(uif.GetWorkSpace()+"/version/abc.txt", []byte("1"), 0644)
	uif.CloseCore()
	CloseServer()
	uif.WriteLog("UIF Service Closed.")
	os.Exit(0)
}

func Elevate() {
	if !uif.IsWindows() {
		uif.UnixChmod()
		return
	}
	isElevated := flag.Bool("is_elevated", false, "not for user.")
	flag.Parse()

	if !uif.IsNeedAdmin() || *isElevated {
		return
	}
	path, _ := os.Executable()
	cmd := elevate.Command(path, "--is_elevated")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

func Entry() error {
	Elevate()
	if err := CheckAndInit(); err != nil {
		if !CheckAction() {
			uif.NormalOpenWeb()
		}
		uif.WriteLog(err.Error())
		return err
	}
	PrintPortInfo()
	RunServer()
	go StartupCore()
	uif.WriteLog("<<<< UIF running >>>>")

	if CheckAction() {
		return nil
	}
	uif.ListenOSQuit()
	if uif.IsOpenBrowser() {
		err := uif.NormalOpenWeb()
		if err != nil {
			uif.WriteLog("Can not open broswer. " + err.Error())
		}
	} else {
		uif.WriteLog("Setting not to open web.")
	}
	return nil // will not block
}

func printPortInfo(address string, t string) {
	_, port1, err := net.SplitHostPort(address)
	if err != nil {
		uif.WriteLog("Wrong " + t + " Address. " + err.Error())
		return
	}
	uif.WriteLog(t + " Address: " + uif.GetOutboundIP() + ":" + port1)
}

func PrintPortInfo() {
	printPortInfo(uif.GetAPIAddress(), "API")
	printPortInfo(uif.GetWebAddress(), "Web")

	outboundIP := uif.GetOutboundIP()
	_, webPort, _ := net.SplitHostPort(uif.GetWebAddress())
	_, apiPort, _ := net.SplitHostPort(uif.GetAPIAddress())
	a := url.QueryEscape(fmt.Sprintf(`http://%s:%s`, outboundIP, apiPort))
	p := url.QueryEscape(uif.GetKey())
	quicLink := fmt.Sprintf(`Quick Open Link:  http://%s:%s?a=%s&p=%s`, outboundIP, webPort, a, p)
	uif.WriteLog(quicLink)
}

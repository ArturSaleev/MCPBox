package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
)

const tailscaleSnapshotTTL = 5 * time.Second

type tailscaleStatusPayload struct {
	BackendState string `json:"BackendState"`
	Self         struct {
		DNSName string `json:"DNSName"`
		Online  bool   `json:"Online"`
	} `json:"Self"`
}

type tailscaleServeTCP struct {
	HTTP  bool `json:"HTTP"`
	HTTPS bool `json:"HTTPS"`
}

type tailscaleServeHandler struct {
	Proxy string `json:"Proxy"`
}

type tailscaleServeWeb struct {
	Handlers map[string]tailscaleServeHandler `json:"Handlers"`
}

type tailscaleServeConfig struct {
	TCP      map[string]tailscaleServeTCP    `json:"TCP"`
	Web      map[string]tailscaleServeWeb    `json:"Web"`
	Services map[string]tailscaleServeConfig `json:"Services"`
}

type tailscaleSnapshot struct {
	online     bool
	dnsName    string
	serve      tailscaleServeConfig
	discovered time.Time
}

var tailscaleCache struct {
	sync.Mutex
	snapshot tailscaleSnapshot
}

func tailscaleConnectionURLs(connectPort, requestPath string) []string {
	snapshot := cachedTailscaleSnapshot()
	if !snapshot.online {
		return nil
	}

	urls := serveConnectionURLs(snapshot.serve, connectPort, requestPath)
	if snapshot.dnsName != "" && connectPort != "" {
		urls = append(urls, "http://"+net.JoinHostPort(snapshot.dnsName, connectPort)+requestPath)
	}

	return dedupeNonEmptyStrings(urls)
}

func cachedTailscaleSnapshot() tailscaleSnapshot {
	tailscaleCache.Lock()
	defer tailscaleCache.Unlock()

	if time.Since(tailscaleCache.snapshot.discovered) < tailscaleSnapshotTTL {
		return tailscaleCache.snapshot
	}

	tailscaleCache.snapshot = discoverTailscaleSnapshot()
	return tailscaleCache.snapshot
}

func discoverTailscaleSnapshot() tailscaleSnapshot {
	snapshot := tailscaleSnapshot{discovered: time.Now()}
	executable := tailscaleExecutable()
	if executable == "" {
		return snapshot
	}

	var status tailscaleStatusPayload
	if !runTailscaleJSON(executable, &status, "status", "--json") {
		return snapshot
	}
	if !strings.EqualFold(strings.TrimSpace(status.BackendState), "running") || !status.Self.Online {
		return snapshot
	}

	snapshot.online = true
	snapshot.dnsName = strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
	_ = runTailscaleJSON(executable, &snapshot.serve, "serve", "status", "--json")
	return snapshot
}

func tailscaleExecutable() string {
	if executable, err := exec.LookPath("tailscale"); err == nil {
		return executable
	}

	const macOSAppExecutable = "/Applications/Tailscale.app/Contents/MacOS/Tailscale"
	if info, err := os.Stat(macOSAppExecutable); err == nil && !info.IsDir() {
		return macOSAppExecutable
	}
	return ""
}

func runTailscaleJSON(executable string, target any, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, executable, args...).Output()
	if err != nil {
		return false
	}
	return json.Unmarshal(output, target) == nil
}

func serveConnectionURLs(config tailscaleServeConfig, connectPort, requestPath string) []string {
	urls := serveConfigConnectionURLs(config, connectPort, requestPath)
	for _, service := range config.Services {
		urls = append(urls, serveConfigConnectionURLs(service, connectPort, requestPath)...)
	}
	urls = dedupeNonEmptyStrings(urls)
	slices.Sort(urls)
	return urls
}

func serveConfigConnectionURLs(config tailscaleServeConfig, connectPort, requestPath string) []string {
	urls := make([]string, 0, len(config.Web))
	for listener, web := range config.Web {
		if !serveWebTargetsPort(web, connectPort, requestPath) {
			continue
		}

		host, listenerPort := splitHostPort(listener)
		if host == "" {
			continue
		}

		scheme := "http"
		if tcp, ok := config.TCP[listenerPort]; ok && tcp.HTTPS {
			scheme = "https"
		} else if listenerPort == "443" {
			scheme = "https"
		}

		authority := host
		if listenerPort != "" && !isDefaultURLPort(scheme, listenerPort) {
			authority = net.JoinHostPort(host, listenerPort)
		}
		urls = append(urls, scheme+"://"+authority+requestPath)
	}
	return urls
}

func serveWebTargetsPort(web tailscaleServeWeb, connectPort, requestPath string) bool {
	for mountPath, handler := range web.Handlers {
		mountPath = strings.TrimSpace(mountPath)
		if mountPath == "" {
			mountPath = "/"
		}
		if mountPath != "/" && requestPath != mountPath && !strings.HasPrefix(requestPath, strings.TrimSuffix(mountPath, "/")+"/") {
			continue
		}
		if proxyTargetsPort(handler.Proxy, connectPort) {
			return true
		}
	}
	return false
}

func proxyTargetsPort(rawProxy, connectPort string) bool {
	rawProxy = strings.TrimSpace(rawProxy)
	if rawProxy == "" || connectPort == "" {
		return false
	}
	if !strings.Contains(rawProxy, "://") {
		rawProxy = "http://" + rawProxy
	}

	parsed, err := url.Parse(rawProxy)
	if err != nil {
		return false
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return port == connectPort
}

func isDefaultURLPort(scheme, port string) bool {
	return (scheme == "https" && port == "443") || (scheme == "http" && port == "80")
}

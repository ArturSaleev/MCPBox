package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/httpapi"
	"github.com/ArturSaleev/MCPBox/internal/installer"
	"github.com/ArturSaleev/MCPBox/internal/llamacpphost"
	"github.com/ArturSaleev/MCPBox/internal/mcphostbridge"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/ollamahost"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/rag"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

const DefaultPort = 38180
const ragAutoReindexInterval = 10 * time.Minute

type listenConfig struct {
	adminHost string
	adminPort int
	mcpHost   string
	mcpPort   int
}

type Options struct {
	Edition  Edition
	StoreDSN string
}

type OpenedRuntime struct {
	Runtime *RuntimeContext
	store   *storage.Store
	closeFn func() error
}

func Run(options Options) error {
	if len(os.Args) > 1 && os.Args[1] == "ollama-chat" {
		return ollamahost.Run(context.Background(), os.Args[2:])
	}
	if len(os.Args) > 1 && os.Args[1] == "llamacpp-chat" {
		return llamacpphost.Run(context.Background(), os.Args[2:])
	}
	if len(os.Args) > 1 && os.Args[1] == "project-http-stdio" {
		return mcphostbridge.Run(context.Background(), os.Args[2:])
	}

	options = normalizeOptions(options)
	listen := resolveListenConfig(options.Edition)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opened, err := OpenRuntime(options)
	if err != nil {
		return fmt.Errorf("init runtime: %w", err)
	}
	defer func() {
		if closeErr := opened.Close(); closeErr != nil {
			log.Printf("store close error: %v", closeErr)
		}
	}()
	runtimeContext := opened.Runtime
	store := opened.store

	registry := orchestrator.NewRegistry(rootCtx)
	packageInstaller := installer.NewService(store, "package_store")
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := registry.Shutdown(shutdownCtx); err != nil {
			log.Printf("runner shutdown error: %v", err)
		}
	}()

	projects, err := store.ListProjects(rootCtx)
	if err != nil {
		return fmt.Errorf("load projects for auto-start: %w", err)
	}

	for _, server := range startupServersForProjects(projects) {
		if err := registry.StartServer(rootCtx, server); err != nil {
			log.Printf("auto-start server %d failed: %v", server.ID, err)
		}
	}

	startRAGAutoReindexLoop(rootCtx, store)

	for _, hook := range options.Edition.StartupHooks {
		if hook == nil {
			continue
		}
		if err := hook(rootCtx, runtimeContext); err != nil {
			return fmt.Errorf("edition startup hook failed: %w", err)
		}
	}

	api := httpapi.NewServerWithInstaller(store, registry, packageInstaller, httpapi.Options{
		EditionID:           options.Edition.ID,
		EditionName:         options.Edition.Name,
		EditionCapabilities: slices.Clone(options.Edition.Capabilities),
		ConnectHost:         listen.mcpHost,
		ConnectPort:         listen.mcpPort,
		UIFS:                options.Edition.UIFS,
		HTTPRegistrars:      toHTTPRegistrars(runtimeContext, options.Edition.HTTPRegistrars),
		ProjectAuthorizer:   options.Edition.ConnectAuthorizer,
	})

	adminAddr := joinListenAddress(listen.adminHost, listen.adminPort)
	mcpAddr := joinListenAddress(listen.mcpHost, listen.mcpPort)

	adminServer := &http.Server{
		Addr:              adminAddr,
		Handler:           api.AdminHandler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	mcpHandler := api.ConnectHandler()
	if adminAddr == mcpAddr {
		mcpHandler = api.Handler()
	}

	mcpServer := &http.Server{
		Addr:              mcpAddr,
		Handler:           mcpHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	servers := make([]*http.Server, 0, 2)
	listeners := make([]net.Listener, 0, 2)

	adminListener, err := net.Listen("tcp", adminServer.Addr)
	if err != nil {
		return fmt.Errorf("admin listen failed: %w", err)
	}
	listeners = append(listeners, adminListener)
	servers = append(servers, adminServer)

	if adminAddr == mcpAddr {
		mcpServer = adminServer
	} else {
		mcpListener, err := net.Listen("tcp", mcpServer.Addr)
		if err != nil {
			_ = adminListener.Close()
			return fmt.Errorf("mcp listen failed: %w", err)
		}
		listeners = append(listeners, mcpListener)
		servers = append(servers, mcpServer)
	}

	go func() {
		<-rootCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, server := range servers {
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("http shutdown error for %s: %v", server.Addr, err)
			}
		}
	}()

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d/", listen.adminPort)
		if err := openBrowser(url); err != nil {
			log.Printf("open browser error: %v", err)
		}
	}()

	for _, address := range listenAddresses(listen.adminHost, listen.adminPort, true) {
		log.Printf("%s admin UI: %s", options.Edition.Name, address)
	}
	for _, address := range listenAddresses(listen.mcpHost, listen.mcpPort, false) {
		log.Printf("%s MCP endpoint: %s", options.Edition.Name, address)
	}

	serveErrCh := make(chan error, len(listeners))
	for idx, listener := range listeners {
		server := servers[idx]
		go func(server *http.Server, listener net.Listener) {
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				serveErrCh <- fmt.Errorf("%s serve failed: %w", listener.Addr().String(), err)
				return
			}
			serveErrCh <- nil
		}(server, listener)
	}

	for range listeners {
		if err := <-serveErrCh; err != nil {
			return err
		}
	}

	return nil
}

func OpenRuntime(options Options) (*OpenedRuntime, error) {
	options = normalizeOptions(options)

	store, err := storage.NewStore(options.StoreDSN)
	if err != nil {
		return nil, err
	}

	opened := &OpenedRuntime{
		Runtime: &RuntimeContext{
			Edition: options.Edition,
			DB:      store.DB(),
			LogAudit: func(ctx context.Context, entry AuditEntry) error {
				return store.CreateAuditLog(ctx, &models.AuditLog{
					ProjectID: entry.ProjectID,
					ServerID:  entry.ServerID,
					Action:    strings.TrimSpace(entry.Action),
					Actor:     strings.TrimSpace(entry.Actor),
					Detail:    entry.Detail,
				})
			},
		},
		store:   store,
		closeFn: store.Close,
	}

	return opened, nil
}

func (o *OpenedRuntime) Close() error {
	if o == nil || o.closeFn == nil {
		return nil
	}
	return o.closeFn()
}

func normalizeOptions(options Options) Options {
	if strings.TrimSpace(options.Edition.ID) == "" {
		options.Edition = FreeEdition()
	}
	if strings.TrimSpace(options.StoreDSN) == "" {
		options.StoreDSN = "mcpbox.db"
	}
	return options
}

func toHTTPRegistrars(runtimeContext *RuntimeContext, registrars []HTTPRegistrar) []func(*http.ServeMux) {
	if len(registrars) == 0 {
		return nil
	}

	adapted := make([]func(*http.ServeMux), 0, len(registrars))
	for _, registrar := range registrars {
		if registrar == nil {
			continue
		}
		adapted = append(adapted, func(mux *http.ServeMux) {
			registrar(runtimeContext, mux)
		})
	}

	return adapted
}

func startRAGAutoReindexLoop(ctx context.Context, store *storage.Store) {
	go func() {
		ticker := time.NewTicker(ragAutoReindexInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runRAGAutoReindexPass(ctx, store)
			}
		}
	}()
}

func runRAGAutoReindexPass(ctx context.Context, store *storage.Store) {
	collections, err := store.ListAutoReindexRAGCollections(ctx)
	if err != nil {
		log.Printf("rag auto reindex list failed: %v", err)
		return
	}

	for _, collection := range collections {
		index, err := rag.NewCollection(collection.CollectionID, collection.Name, collection.IndexPath)
		if err != nil {
			log.Printf("rag auto reindex open failed for %s: %v", collection.CollectionID, err)
			writeSystemAuditLog(store, "rag_collection_auto_index_failed", fmt.Sprintf("%s: %v", collection.CollectionID, err))
			continue
		}

		roots := make([]string, 0, 2)
		if sourcePath := strings.TrimSpace(collection.SourcePath); sourcePath != "" {
			roots = append(roots, sourcePath)
		}
		managedPath := rag.ResolveManagedSourcePath(store.DataRoot(), collection.CollectionID)
		if info, statErr := os.Stat(managedPath); statErr == nil && info.IsDir() {
			roots = append(roots, managedPath)
		}

		reindexErr := index.IndexFolders(roots)
		_ = index.Close()
		if reindexErr != nil {
			log.Printf("rag auto reindex failed for %s: %v", collection.CollectionID, reindexErr)
			writeSystemAuditLog(store, "rag_collection_auto_index_failed", fmt.Sprintf("%s: %v", collection.CollectionID, reindexErr))
			continue
		}

		writeSystemAuditLog(store, "rag_collection_auto_indexed", collection.CollectionID)
	}
}

func writeSystemAuditLog(store *storage.Store, action, detail string) {
	if err := store.CreateAuditLog(context.Background(), &models.AuditLog{
		Action: action,
		Actor:  "system",
		Detail: detail,
	}); err != nil {
		log.Printf("write audit log failed for %s: %v", action, err)
	}
}

func resolveListenConfig(edition Edition) listenConfig {
	hostFlag := flag.String("host", "", "host for the MCPBox admin UI and API listener")
	portFlag := flag.Int("port", 0, "TCP port for the MCPBox admin UI and API listener")
	mcpHostFlag := flag.String("mcp-host", "", "host for the MCPBox MCP listener")
	mcpPortFlag := flag.Int("mcp-port", 0, "TCP port for the MCPBox MCP listener")
	flag.Parse()

	config := listenConfig{
		adminHost: firstNonEmpty(strings.TrimSpace(*hostFlag), strings.TrimSpace(os.Getenv("MCPBOX_HOST")), strings.TrimSpace(edition.AdminHost), "127.0.0.1"),
		adminPort: firstValidPort(*portFlag, os.Getenv("MCPBOX_PORT"), edition.AdminPort, DefaultPort, "MCPBOX_PORT"),
		mcpHost:   firstNonEmpty(strings.TrimSpace(*mcpHostFlag), strings.TrimSpace(os.Getenv("MCPBOX_MCP_HOST")), strings.TrimSpace(edition.MCPHost), "0.0.0.0"),
		mcpPort:   firstValidPort(*mcpPortFlag, os.Getenv("MCPBOX_MCP_PORT"), edition.MCPPort, 0, "MCPBOX_MCP_PORT"),
	}

	if config.mcpPort == 0 {
		config.mcpPort = config.adminPort
	}

	return config
}

func mustBeValidPort(port int, source string) int {
	if port <= 0 || port > 65535 {
		log.Fatalf("invalid port from %s: %d", source, port)
	}

	return port
}

func openBrowser(target string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}

	return cmd.Start()
}

func listenAddresses(host string, port int, localOnly bool) []string {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	addresses := []string{}
	seen := map[string]struct{}{}

	appendAddress := func(candidateHost string) {
		candidateHost = strings.Trim(strings.TrimSpace(candidateHost), "[]")
		if candidateHost == "" {
			return
		}

		address := fmt.Sprintf("http://%s:%d/", candidateHost, port)
		if strings.Contains(candidateHost, ":") {
			address = fmt.Sprintf("http://[%s]:%d/", candidateHost, port)
		}
		if _, ok := seen[address]; ok {
			return
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}

	switch {
	case host == "", host == "0.0.0.0", host == "::":
		appendAddress("127.0.0.1")
	default:
		appendAddress(host)
	}

	if localOnly {
		slices.Sort(addresses)
		return addresses
	}

	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return addresses
	}

	for _, iface := range ifaces {
		ipNet, ok := iface.(*net.IPNet)
		if !ok || ipNet == nil {
			continue
		}

		ip := ipNet.IP
		if ip == nil || ip.IsLoopback() {
			continue
		}

		if ipv4 := ip.To4(); ipv4 != nil {
			appendAddress(ipv4.String())
			continue
		}

		appendAddress(ip.String())
	}

	slices.Sort(addresses)
	return addresses
}

func joinListenAddress(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Sprintf(":%d", port)
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstValidPort(flagValue int, envValue string, editionValue int, fallback int, envName string) int {
	if flagValue > 0 {
		return mustBeValidPort(flagValue, "flag")
	}

	if raw := strings.TrimSpace(envValue); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("invalid %s value %q", envName, raw)
		}
		return mustBeValidPort(port, envName)
	}

	if editionValue > 0 {
		return mustBeValidPort(editionValue, "edition")
	}

	if fallback > 0 {
		return mustBeValidPort(fallback, "default")
	}

	return 0
}

func startupServersForProjects(projects []models.Project) []models.MCPServer {
	startupServers := make([]models.MCPServer, 0)

	for _, project := range projects {
		if project.IsPaused {
			continue
		}

		hasAutoStart := slices.ContainsFunc(project.Servers, func(server models.MCPServer) bool {
			return server.Transport == models.ServerTransportSTDIO && server.AutoStart && server.IsEnabled
		})
		if !hasAutoStart {
			continue
		}

		for _, server := range project.Servers {
			if server.Transport != models.ServerTransportSTDIO || !server.IsEnabled {
				continue
			}
			startupServers = append(startupServers, server)
		}
	}

	return startupServers
}

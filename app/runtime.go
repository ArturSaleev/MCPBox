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
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/ollamahost"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/rag"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

const DefaultPort = 38180
const ragAutoReindexInterval = 10 * time.Minute

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

	options = normalizeOptions(options)
	port := resolvePort()
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
		HTTPRegistrars:      toHTTPRegistrars(runtimeContext, options.Edition.HTTPRegistrars),
	})
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-rootCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown error: %v", err)
		}
	}()

	listener, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return fmt.Errorf("http listen failed: %w", err)
	}

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d/", port)
		if err := openBrowser(url); err != nil {
			log.Printf("open browser error: %v", err)
		}
	}()

	for _, address := range listenAddresses(port) {
		log.Printf("%s UI: %s", options.Edition.Name, address)
	}

	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
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

		reindexErr := index.IndexFolder(collection.SourcePath)
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

func resolvePort() int {
	portFlag := flag.Int("port", 0, "TCP port for the MCPBox HTTP server")
	flag.Parse()

	if *portFlag > 0 {
		return mustBeValidPort(*portFlag, "flag")
	}

	if raw := os.Getenv("MCPBOX_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatalf("invalid MCPBOX_PORT value %q", raw)
		}

		return mustBeValidPort(port, "MCPBOX_PORT")
	}

	return DefaultPort
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

func listenAddresses(port int) []string {
	addresses := []string{
		fmt.Sprintf("http://127.0.0.1:%d/", port),
	}

	ifaces, err := net.InterfaceAddrs()
	if err != nil {
		return addresses
	}

	seen := map[string]struct{}{
		addresses[0]: {},
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

		ip = ip.To4()
		if ip == nil {
			continue
		}

		address := fmt.Sprintf("http://%s:%d/", ip.String(), port)
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}

	slices.Sort(addresses)
	return addresses
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

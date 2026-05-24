package main

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

	"MCPBox/internal/httpapi"
	"MCPBox/internal/installer"
	"MCPBox/internal/models"
	"MCPBox/internal/ollamahost"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/rag"
	"MCPBox/internal/storage"
)

const defaultPort = 38180
const ragAutoReindexInterval = 10 * time.Minute

func main() {
	if len(os.Args) > 1 && os.Args[1] == "ollama-chat" {
		if err := ollamahost.Run(context.Background(), os.Args[2:]); err != nil {
			log.Fatalf("ollama chat failed: %v", err)
		}
		return
	}

	port := resolvePort()
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.NewStore("mcpbox.db")
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

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
		log.Fatalf("load projects for auto-start: %v", err)
	}

	for _, server := range startupServersForProjects(projects) {
		if err := registry.StartServer(rootCtx, server); err != nil {
			log.Printf("auto-start server %d failed: %v", server.ID, err)
		}
	}

	startRAGAutoReindexLoop(rootCtx, store)

	api := httpapi.NewServerWithInstaller(store, registry, packageInstaller)
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
		log.Fatalf("http listen failed: %v", err)
	}

	go func() {
		url := fmt.Sprintf("http://127.0.0.1:%d/", port)
		if err := openBrowser(url); err != nil {
			log.Printf("open browser error: %v", err)
		}
	}()

	for _, address := range listenAddresses(port) {
		log.Printf("MCPBox UI: %s", address)
	}

	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
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

	return defaultPort
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

	slices.SortFunc(addresses[1:], func(a, b string) int {
		return strings.Compare(a, b)
	})

	return addresses
}

func startupServersForProjects(projects []models.Project) []models.MCPServer {
	servers := make([]models.MCPServer, 0)
	for _, project := range projects {
		if project.IsPaused {
			continue
		}

		projectShouldStart := false
		for _, server := range project.Servers {
			if server.Transport != models.ServerTransportSTDIO {
				continue
			}
			if !server.IsEnabled {
				continue
			}
			if server.AutoStart {
				projectShouldStart = true
				break
			}
		}

		if !projectShouldStart {
			continue
		}

		for _, server := range project.Servers {
			if server.Transport != models.ServerTransportSTDIO {
				continue
			}
			if !server.IsEnabled {
				continue
			}
			servers = append(servers, server)
		}
	}

	return servers
}

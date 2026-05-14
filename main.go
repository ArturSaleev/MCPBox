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
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

const defaultPort = 38180

func main() {
	port := resolvePort()
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.NewStore("mcpbox.db")
	if err != nil {
		log.Fatalf("init storage: %v", err)
	}

	registry := orchestrator.NewRegistry(rootCtx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := registry.Shutdown(shutdownCtx); err != nil {
			log.Printf("runner shutdown error: %v", err)
		}
	}()

	autoStartServers, err := store.ListAutoStartServers(rootCtx)
	if err != nil {
		log.Fatalf("load auto-start servers: %v", err)
	}

	for _, server := range autoStartServers {
		if err := registry.StartServer(rootCtx, server); err != nil {
			log.Printf("auto-start server %d failed: %v", server.ID, err)
		}
	}

	api := httpapi.NewServer(store, registry)
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

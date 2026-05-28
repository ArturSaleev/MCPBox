package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/mcpboxpro/internal/proauth"
	"github.com/ArturSaleev/mcpboxpro/internal/prohttp"
)

func main() {
	authService := &proauth.Service{}

	proEdition := app.Edition{
		ID:         "pro",
		Name:       "MCPBox Pro",
		BinaryName: "MCPBoxPro",
		Capabilities: []string{
			"projects",
			"market",
			"rag",
			"ollama",
			"lmstudio",
			"audit_logs",
			"performance_metrics",
			"sso",
			"rbac",
			"advanced_rag",
			"ocr",
			"team_control_plane",
			"guardrails",
			"agent_memory",
		},
		StartupHooks: []app.StartupHook{
			func(ctx context.Context, runtime *app.RuntimeContext) error {
				service, err := proauth.NewServiceFromRuntime(runtime)
				if err != nil {
					return err
				}
				authService = service
				return authService.AutoMigrate(ctx)
			},
		},
		HTTPRegistrars: []app.HTTPRegistrar{
			func(runtime *app.RuntimeContext, mux *http.ServeMux) {
				prohttp.RegisterRoutes(runtime, mux, authService)
			},
		},
	}

	if err := app.Run(app.Options{
		Edition: proEdition,
	}); err != nil {
		log.Fatalf("mcpbox pro failed: %v", err)
	}
}

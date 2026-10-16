package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/prohttp"
	"github.com/ArturSaleev/MCPBox/internal/propolicy"
	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"github.com/ArturSaleev/MCPBox/internal/prosso"
	"github.com/ArturSaleev/MCPBox/internal/prostorage"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "project-http-stdio" || os.Args[1] == "ollama-chat" || os.Args[1] == "llamacpp-chat") {
		if err := app.Run(app.Options{}); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "admin" {
		if err := runAdminCommand(os.Args[2:]); err != nil {
			log.Fatalf("mcpbox admin command failed: %v", err)
		}
		return
	}

	if err := ensureInitialLocalAdmin(); err != nil {
		log.Fatalf("mcpbox bootstrap failed: %v", err)
	}

	authService := &proauth.Service{}
	policyService := &propolicy.Service{}
	ssoService := &prosso.Service{}
	ragBoxService := &proragbox.Service{}
	ragBoxBindingService := &proragbox.BindingService{}
	storageService := &prostorage.Service{}
	storageManager := &prostorage.Manager{}
	projectShadowService := &proprojects.Service{}

	capabilities := []string{
		"projects",
		"cli",
		"market",
		"rag",
		"ollama",
		"lmstudio",
		"llamacpp",
		"audit_logs",
		"performance_metrics",
		"auth",
		"local_auth",
		"agent_tokens",
	}
	if prosso.IsConfiguredFromEnv() {
		capabilities = append(capabilities, "sso")
	}

	proEdition := app.Edition{
		ID:           "mcpbox",
		Name:         "MCPBox",
		BinaryName:   "MCPBox",
		Capabilities: capabilities,
		AdminHost:    "127.0.0.1",
		AdminPort:    38181,
		MCPHost:      "0.0.0.0",
		MCPPort:      38182,
		StartupHooks: []app.StartupHook{
			func(ctx context.Context, runtime *app.RuntimeContext) error {
				service, err := proauth.NewServiceFromRuntime(runtime)
				if err != nil {
					return err
				}
				authService = service
				if err := authService.AutoMigrate(ctx); err != nil {
					return err
				}
				storageConfig, err := prostorage.NewServiceFromRuntime(runtime)
				if err != nil {
					return err
				}
				storageService = storageConfig
				if err := storageService.AutoMigrate(ctx); err != nil {
					return err
				}
				manager, err := prostorage.NewManager(runtime, storageService)
				if err != nil {
					return err
				}
				storageManager = manager
				if _, err := storageManager.Reload(ctx); err != nil {
					return err
				}
				projectShadows, err := proprojects.NewService(storageManager.OperationalDB(), runtime.DB)
				if err != nil {
					return err
				}
				projectShadowService = projectShadows
				if err := projectShadowService.AutoMigrate(ctx); err != nil {
					return err
				}
				if _, err := projectShadowService.SyncAll(ctx); err != nil {
					return err
				}
				policy, err := propolicy.NewService(storageManager.OperationalDB(), runtime.DB)
				if err != nil {
					return err
				}
				policyService = policy
				if err := policyService.AutoMigrate(ctx); err != nil {
					return err
				}
				sso, err := prosso.NewServiceFromRuntime(runtime, authService)
				if err != nil {
					return err
				}
				ssoService = sso
				if err := ssoService.AutoMigrate(ctx); err != nil {
					return err
				}
				ragbox, err := proragbox.NewService(storageManager.OperationalDB())
				if err != nil {
					return err
				}
				ragBoxService = ragbox
				if err := ragBoxService.AutoMigrate(ctx); err != nil {
					return err
				}
				bindings, err := proragbox.NewBindingService(storageManager.OperationalDB(), runtime.DB, filepath.Join(runtime.DataRoot, ".mcpboxpro", "rag-shadow"))
				if err != nil {
					return err
				}
				ragBoxBindingService = bindings
				return ragBoxBindingService.AutoMigrate(ctx)
			},
		},
		HTTPRegistrars: []app.HTTPRegistrar{
			func(runtime *app.RuntimeContext, mux *http.ServeMux) {
				prohttp.RegisterRoutes(runtime, mux, authService, ssoService, ragBoxService, ragBoxBindingService, policyService, storageService, storageManager, projectShadowService)
			},
		},
		AdminAPIAuthorizer: func(requiredScope string, next http.Handler) http.Handler {
			return proauth.Middleware(authService, requiredScope, func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		},
		ConnectAuthorizer: prohttp.ProjectAuthorizer(policyService),
	}

	if err := app.Run(app.Options{
		Edition: proEdition,
	}); err != nil {
		log.Fatalf("mcpbox failed: %v", err)
	}
}

func ensureInitialLocalAdmin() error {
	opened, err := app.OpenRuntime(app.Options{
		Edition: app.Edition{
			ID:   "mcpbox",
			Name: "MCPBox",
		},
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = opened.Close()
	}()

	authService, err := proauth.NewServiceFromRuntime(opened.Runtime)
	if err != nil {
		return err
	}
	if err := authService.AutoMigrate(context.Background()); err != nil {
		return err
	}

	hasLocalUsers, err := authService.HasLocalPasswordUsers(context.Background())
	if err != nil {
		return err
	}
	if hasLocalUsers {
		return nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		log.Printf("No local admin user found. Run `go run . admin create --email ... --password ...` or start the app in an interactive terminal to create the first user.")
		return nil
	}

	return promptInitialLocalAdmin(authService)
}

func promptInitialLocalAdmin(authService *proauth.Service) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("No local admin user found.")
	fmt.Println("Create the first admin account to initialize MCPBox.")

	email, err := promptRequiredLine(reader, "Email")
	if err != nil {
		return err
	}

	displayName, err := promptOptionalLine(reader, "Display name", email)
	if err != nil {
		return err
	}

	password, err := promptPasswordWithConfirmation()
	if err != nil {
		return err
	}

	record, created, err := authService.CreateOrUpdateLocalPasswordUser(context.Background(), proauth.CreateLocalUserInput{
		Email:       email,
		DisplayName: displayName,
		Password:    password,
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	})
	if err != nil {
		return err
	}

	if created {
		fmt.Printf("Created local admin user: %s\n", record.Email)
	} else {
		fmt.Printf("Updated local admin user: %s\n", record.Email)
	}
	return nil
}

func promptRequiredLine(reader *bufio.Reader, label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, nil
		}
		fmt.Printf("%s is required.\n", label)
	}
}

func promptOptionalLine(reader *bufio.Reader, label, fallback string) (string, error) {
	fmt.Printf("%s [%s]: ", label, fallback)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return value, nil
}

func promptPasswordWithConfirmation() (string, error) {
	for {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}

		fmt.Print("Confirm password: ")
		confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}

		password := strings.TrimSpace(string(passwordBytes))
		confirm := strings.TrimSpace(string(confirmBytes))

		if len(password) < 8 {
			fmt.Println("Password must be at least 8 characters.")
			continue
		}
		if password != confirm {
			fmt.Println("Passwords do not match. Please try again.")
			continue
		}

		return password, nil
	}
}

func runAdminCommand(args []string) error {
	opened, err := app.OpenRuntime(app.Options{
		Edition: app.Edition{
			ID:   "mcpbox",
			Name: "MCPBox",
		},
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = opened.Close()
	}()

	switch {
	case len(args) > 0 && args[0] == "create":
		return runAdminCreate(opened.Runtime, args[1:])
	case len(args) > 0 && args[0] == "policybox":
		return runAdminPolicyBox(opened.Runtime, args[1:])
	default:
		return fmt.Errorf("supported commands: admin create, admin policybox bind-project, admin policybox list-bindings")
	}
}

func runAdminCreate(runtime *app.RuntimeContext, args []string) error {
	fs := flag.NewFlagSet("admin create", flag.ContinueOnError)
	email := fs.String("email", "", "admin email")
	password := fs.String("password", "", "admin password")
	displayName := fs.String("name", "", "admin display name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *email == "" || *password == "" {
		return fmt.Errorf("both --email and --password are required")
	}

	authService, err := proauth.NewServiceFromRuntime(runtime)
	if err != nil {
		return err
	}
	if err := authService.AutoMigrate(context.Background()); err != nil {
		return err
	}

	record, created, err := authService.CreateOrUpdateLocalPasswordUser(context.Background(), proauth.CreateLocalUserInput{
		Email:       *email,
		DisplayName: *displayName,
		Password:    *password,
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	})
	if err != nil {
		return err
	}

	verb := "updated"
	if created {
		verb = "created"
	}
	fmt.Printf("Local admin %s: %s (user #%d)\n", verb, record.Email, record.ID)
	return nil
}

func runAdminPolicyBox(runtime *app.RuntimeContext, args []string) error {
	service, err := propolicy.NewServiceFromRuntime(runtime)
	if err != nil {
		return err
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("supported commands: admin policybox bind-project, admin policybox list-bindings")
	}

	switch args[0] {
	case "bind-project":
		fs := flag.NewFlagSet("admin policybox bind-project", flag.ContinueOnError)
		mcpProjectID := fs.Uint("mcp-project-id", 0, "MCPBox project id")
		policyProjectID := fs.String("policy-project-id", "", "PolicyBox project id")
		policyProjectSlug := fs.String("policy-project-slug", "", "PolicyBox project slug")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if err := service.UpsertProjectBinding(context.Background(), propolicy.RuntimeProjectBinding{
			MCPBoxProjectID:   uint(*mcpProjectID),
			PolicyProjectID:   *policyProjectID,
			PolicyProjectSlug: *policyProjectSlug,
		}); err != nil {
			return err
		}
		fmt.Printf("PolicyBox binding saved: mcp_project_id=%d policy_project_id=%s policy_project_slug=%s\n", *mcpProjectID, *policyProjectID, *policyProjectSlug)
		return nil
	case "list-bindings":
		items, err := service.ListProjectBindings(context.Background())
		if err != nil {
			return err
		}
		for _, item := range items {
			fmt.Printf("mcp_project_id=%d policy_project_id=%s policy_project_slug=%s\n", item.MCPBoxProjectID, item.PolicyProjectID, item.PolicyProjectSlug)
		}
		return nil
	default:
		return fmt.Errorf("unsupported admin policybox command: %s", args[0])
	}
}

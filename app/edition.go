package app

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/ArturSaleev/MCPBox/connectruntime"
	"gorm.io/gorm"
)

// Edition describes one distributable MCPBox build configuration.
type Edition struct {
	ID                string
	Name              string
	BinaryName        string
	Capabilities      []string
	UIFS              fs.FS
	AdminHost         string
	AdminPort         int
	MCPHost           string
	MCPPort           int
	StartupHooks      []StartupHook
	HTTPRegistrars    []HTTPRegistrar
	ConnectAuthorizer connectruntime.ProjectAuthorizer
}

// RuntimeContext exposes safe shared runtime handles for edition extensions.
type RuntimeContext struct {
	Edition  Edition
	DB       *gorm.DB
	LogAudit func(ctx context.Context, entry AuditEntry) error
}

type AuditEntry struct {
	ProjectID *uint
	ServerID  *uint
	Action    string
	Actor     string
	Detail    string
}

// StartupHook lets an edition perform migrations or service initialization.
type StartupHook func(ctx context.Context, runtime *RuntimeContext) error

// HTTPRegistrar lets a caller extend the shared HTTP server without forking it.
type HTTPRegistrar func(runtime *RuntimeContext, mux *http.ServeMux)

// FreeEdition returns the default MCPBox build metadata.
func FreeEdition() Edition {
	return Edition{
		ID:         "free",
		Name:       "MCPBox",
		BinaryName: "MCPBox",
		Capabilities: []string{
			"projects",
			"market",
			"rag",
			"ollama",
			"llamacpp",
			"lmstudio",
			"audit_logs",
			"performance_metrics",
		},
	}
}

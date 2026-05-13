package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"MCPBox/internal/models"
)

type Registry struct {
	mu      sync.RWMutex
	runners map[uint]*ServerRunner
}

func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[uint]*ServerRunner),
	}
}

func (r *Registry) StartServer(ctx context.Context, server models.MCPServer) error {
	runner := r.getOrCreateRunner(server)
	return runner.Start(ctx)
}

func (r *Registry) StopServer(ctx context.Context, serverID uint) error {
	r.mu.RLock()
	runner := r.runners[serverID]
	r.mu.RUnlock()

	if runner == nil {
		return nil
	}

	return runner.Stop(ctx)
}

func (r *Registry) Status(serverID uint) string {
	r.mu.RLock()
	runner := r.runners[serverID]
	r.mu.RUnlock()

	if runner != nil && runner.Running() {
		return "Running"
	}

	return "Stopped"
}

func (r *Registry) RunnerForProject(ctx context.Context, project models.Project) (*ServerRunner, *models.MCPServer, error) {
	if len(project.Servers) == 0 {
		return nil, nil, errors.New("project has no configured MCP servers")
	}

	primary := project.Servers[0]
	runner := r.getOrCreateRunner(primary)
	if !runner.Running() {
		if err := runner.Start(ctx); err != nil {
			return nil, nil, fmt.Errorf("start project server: %w", err)
		}
	}

	return runner, &primary, nil
}

func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.RLock()
	runners := make([]*ServerRunner, 0, len(r.runners))
	for _, runner := range r.runners {
		runners = append(runners, runner)
	}
	r.mu.RUnlock()

	var firstErr error
	for _, runner := range runners {
		if err := runner.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (r *Registry) getOrCreateRunner(server models.MCPServer) *ServerRunner {
	r.mu.Lock()
	defer r.mu.Unlock()

	if runner, ok := r.runners[server.ID]; ok {
		return runner
	}

	runner := NewServerRunner(server)
	r.runners[server.ID] = runner
	return runner
}

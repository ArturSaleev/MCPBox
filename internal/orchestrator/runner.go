package orchestrator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

const scannerBufferSize = 1024 * 1024

type ServerRunner struct {
	server models.MCPServer

	mu          sync.RWMutex
	writeMu     sync.Mutex
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	cancel      context.CancelFunc
	done        chan error
	running     bool
	subscribers map[int]chan []byte
	pending     map[string]chan []byte
	nextSubID   int
}

func NewServerRunner(server models.MCPServer) *ServerRunner {
	return &ServerRunner{
		server:      server,
		subscribers: make(map[int]chan []byte),
		pending:     make(map[string]chan []byte),
	}
}

func (r *ServerRunner) Server() models.MCPServer {
	return r.server
}

func (r *ServerRunner) Running() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

func (r *ServerRunner) Start(parent context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}

	if r.server.Transport == models.ServerTransportHTTPStream {
		return errors.New("http streaming servers are managed externally and cannot be started")
	}

	executable, args, err := commandSpec(r.server)
	if err != nil {
		return err
	}

	childCtx, cancel := context.WithCancel(parent)
	cmd := exec.CommandContext(childCtx, executable, args...)
	if r.server.WorkingDir != "" {
		cmd.Dir = r.server.WorkingDir
	}
	env, err := commandEnv(r.server)
	if err != nil {
		cancel()
		return err
	}
	if len(env) > 0 {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		_ = stdin.Close()
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		_ = stdin.Close()
		return fmt.Errorf("start process: %w", err)
	}

	r.cmd = cmd
	r.stdin = stdin
	r.cancel = cancel
	r.done = make(chan error, 1)
	r.running = true

	go r.consumeStdout(stdout)
	go r.consumeStderr(stderr)
	go r.waitProcess()

	return nil
}

func (r *ServerRunner) Send(ctx context.Context, payload []byte) error {
	return r.send(ctx, payload)
}

func (r *ServerRunner) SendAndWait(ctx context.Context, payload []byte) ([]byte, error) {
	requestID, ok, err := jsonRPCID(payload)
	if err != nil {
		return nil, err
	}
	if !ok {
		if err := r.send(ctx, payload); err != nil {
			return nil, err
		}
		return nil, errors.New("json-rpc request id is required for synchronous transport")
	}

	responseCh := make(chan []byte, 1)

	r.mu.Lock()
	if _, exists := r.pending[requestID]; exists {
		r.mu.Unlock()
		return nil, fmt.Errorf("request with id %s is already in flight", requestID)
	}
	r.pending[requestID] = responseCh
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.pending, requestID)
		r.mu.Unlock()
	}()

	if err := r.send(ctx, payload); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response, ok := <-responseCh:
		if !ok {
			return nil, errors.New("server stopped before responding")
		}
		return response, nil
	}
}

func (r *ServerRunner) send(ctx context.Context, payload []byte) error {
	r.mu.RLock()
	stdin := r.stdin
	running := r.running
	r.mu.RUnlock()

	if !running || stdin == nil {
		return errors.New("server is not running")
	}

	r.writeMu.Lock()
	defer r.writeMu.Unlock()

	// MCP servers speak JSON-RPC over stdio line by line, so each request must
	// be terminated by '\n' to allow the child process scanner to emit frames.
	message := bytes.TrimSpace(payload)
	if len(message) == 0 {
		return errors.New("empty payload")
	}

	writeErrCh := make(chan error, 1)
	go func() {
		_, err := stdin.Write(append(message, '\n'))
		writeErrCh <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-writeErrCh:
		if err != nil {
			return fmt.Errorf("write request to stdin: %w", err)
		}
	}

	return nil
}

func (r *ServerRunner) Subscribe() (<-chan []byte, func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextSubID
	r.nextSubID++

	ch := make(chan []byte, 32)
	r.subscribers[id] = ch

	unsubscribe := func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if existing, ok := r.subscribers[id]; ok {
			delete(r.subscribers, id)
			close(existing)
		}
	}

	return ch, unsubscribe
}

func (r *ServerRunner) Stop(ctx context.Context) error {
	r.mu.RLock()
	if !r.running || r.cmd == nil {
		r.mu.RUnlock()
		return nil
	}

	cmd := r.cmd
	cancel := r.cancel
	stdin := r.stdin
	done := r.done
	r.mu.RUnlock()

	if stdin != nil {
		_ = stdin.Close()
	}

	if cancel != nil {
		cancel()
	}

	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}

		select {
		case err := <-done:
			return err
		default:
			return ctx.Err()
		}
	}
}

func (r *ServerRunner) waitProcess() {
	err := normalizeProcessExitError(r.cmd.Wait())

	r.mu.Lock()
	r.running = false
	r.stdin = nil
	r.cancel = nil
	r.cmd = nil
	done := r.done
	r.done = nil
	r.mu.Unlock()

	if done != nil {
		done <- err
		close(done)
	}

	r.closeSubscribers()
}

func (r *ServerRunner) consumeStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferSize)

	// Stdout is treated as a framed JSON-RPC stream. Each scanned line is copied
	// before fan-out so SSE clients never race with the scanner's internal buffer.
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		r.routePending(line)
		r.broadcast(line)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("stdout scanner error for server %d: %v", r.server.ID, err)
	}
}

func (r *ServerRunner) consumeStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 32*1024), scannerBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		if isBenignServerStderr(line) {
			log.Printf("server %d info: %s", r.server.ID, line)
			continue
		}
		log.Printf("server %d stderr: %s", r.server.ID, line)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		log.Printf("stderr scanner error for server %d: %v", r.server.ID, err)
	}
}

func (r *ServerRunner) broadcast(line []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ch := range r.subscribers {
		select {
		case ch <- line:
		default:
			log.Printf("dropping stdout frame for slow subscriber on server %d", r.server.ID)
		}
	}
}

func (r *ServerRunner) closeSubscribers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, ch := range r.subscribers {
		delete(r.subscribers, id)
		close(ch)
	}

	for id, ch := range r.pending {
		delete(r.pending, id)
		close(ch)
	}
}

func (r *ServerRunner) routePending(line []byte) {
	responseID, ok, err := jsonRPCID(line)
	if err != nil || !ok {
		return
	}

	r.mu.RLock()
	responseCh := r.pending[responseID]
	r.mu.RUnlock()
	if responseCh == nil {
		return
	}

	select {
	case responseCh <- line:
	default:
	}
}

func splitCommandLine(raw string) (string, []string, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return "", nil, errors.New("launch command is empty")
	}

	// Stage 1 keeps parsing intentionally simple. The command should be stored
	// without shell metacharacters so it can be executed directly by os/exec.
	return parts[0], parts[1:], nil
}

func commandSpec(server models.MCPServer) (string, []string, error) {
	if strings.TrimSpace(server.Command) != "" {
		var args []string
		if strings.TrimSpace(server.ArgsJSON) != "" {
			if err := json.Unmarshal([]byte(server.ArgsJSON), &args); err != nil {
				return "", nil, fmt.Errorf("decode args: %w", err)
			}
		}

		return server.Command, args, nil
	}

	return splitCommandLine(server.LaunchCommand)
}

func commandEnv(server models.MCPServer) ([]string, error) {
	if strings.TrimSpace(server.EnvJSON) == "" {
		return nil, nil
	}

	var pairs []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(server.EnvJSON), &pairs); err != nil {
		return nil, fmt.Errorf("decode env vars: %w", err)
	}

	baseEnv := os.Environ()
	envMap := make(map[string]string, len(baseEnv)+len(pairs))
	for _, item := range baseEnv {
		key, value, found := strings.Cut(item, "=")
		if found {
			envMap[key] = value
		}
	}

	for _, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key == "" {
			continue
		}

		envMap[key] = pair.Value
	}

	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}

	return env, nil
}

func isBenignServerStderr(line string) bool {
	normalized := strings.TrimSpace(line)
	if normalized == "" {
		return true
	}

	benignPrefixes := []string{
		"Secure MCP Filesystem Server running on stdio",
		"Client does not support MCP Roots, using allowed directories set from server args:",
	}

	for _, prefix := range benignPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}

	return false
}

func normalizeProcessExitError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}

	if runtime.GOOS != "windows" {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				switch status.Signal() {
				case os.Interrupt, syscall.SIGTERM, syscall.SIGKILL:
					return nil
				}
			}
		}
	}

	return err
}

func jsonRPCID(payload []byte) (string, bool, error) {
	var envelope struct {
		ID json.RawMessage `json:"id"`
	}

	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", false, fmt.Errorf("decode json-rpc envelope: %w", err)
	}

	id := bytes.TrimSpace(envelope.ID)
	if len(id) == 0 || bytes.Equal(id, []byte("null")) {
		return "", false, nil
	}

	return string(id), true, nil
}

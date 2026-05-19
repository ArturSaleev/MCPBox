package ollamahost

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mark3labs/mcphost/sdk"
)

type Options struct {
	Model      string
	ConfigFile string
	Input      io.Reader
	Output     io.Writer
}

func Run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("ollama-chat", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	opts := Options{
		Input:  os.Stdin,
		Output: os.Stdout,
	}

	flagSet.StringVar(&opts.Model, "model", "", "Ollama model name")
	flagSet.StringVar(&opts.ConfigFile, "config", "", "mcphost-compatible MCP config file")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(opts.Model) == "" {
		return errors.New("ollama model is required")
	}
	if strings.TrimSpace(opts.ConfigFile) == "" {
		return errors.New("config file is required")
	}

	return runInteractiveSession(ctx, opts)
}

func runInteractiveSession(ctx context.Context, opts Options) error {
	host, err := sdk.New(ctx, &sdk.Options{
		Model:      "ollama:" + strings.TrimSpace(opts.Model),
		ConfigFile: strings.TrimSpace(opts.ConfigFile),
		Streaming:  false,
		Quiet:      true,
	})
	if err != nil {
		return fmt.Errorf("create embedded mcphost session: %w", err)
	}
	defer host.Close()

	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	in := opts.Input
	if in == nil {
		in = os.Stdin
	}

	fmt.Fprintf(out, "MCPBox Ollama chat started with model %s\n", strings.TrimSpace(opts.Model))
	fmt.Fprintln(out, "Type your prompt and press Enter. Use /exit to quit.")

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Fprint(out, "\n> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		prompt := strings.TrimSpace(scanner.Text())
		switch prompt {
		case "":
			continue
		case "/exit", "exit", "quit":
			return nil
		}

		response, err := host.Prompt(ctx, prompt)
		if err != nil {
			fmt.Fprintf(out, "\n[error] %v\n", err)
			continue
		}

		fmt.Fprintf(out, "\n%s\n", strings.TrimSpace(response))
	}
}

package llamacpphost

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcphost/sdk"
)

const (
	ansiReset     = "\033[0m"
	ansiDim       = "\033[2m"
	ansiBold      = "\033[1m"
	ansiCyan      = "\033[36m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiRed       = "\033[31m"
	ansiBlueBG    = "\033[44m"
	ansiWhiteText = "\033[97m"
)

type Options struct {
	Model            string
	ConfigFile       string
	ProviderURL      string
	ProviderAPIKey   string
	SystemPromptFile string
	Input            io.Reader
	Output           io.Writer
}

func Run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("llamacpp-chat", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	opts := Options{
		Input:          os.Stdin,
		Output:         os.Stdout,
		ProviderAPIKey: "llama.cpp",
	}

	flagSet.StringVar(&opts.Model, "model", "", "OpenAI-compatible model name")
	flagSet.StringVar(&opts.ConfigFile, "config", "", "mcphost-compatible MCP config file")
	flagSet.StringVar(&opts.ProviderURL, "provider-url", "", "OpenAI-compatible base URL")
	flagSet.StringVar(&opts.ProviderAPIKey, "provider-api-key", "llama.cpp", "provider API key")
	flagSet.StringVar(&opts.SystemPromptFile, "system-prompt-file", "", "path to a text file with system prompt")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(opts.Model) == "" {
		return errors.New("llama.cpp model is required")
	}
	if strings.TrimSpace(opts.ConfigFile) == "" {
		return errors.New("config file is required")
	}
	if strings.TrimSpace(opts.ProviderURL) == "" {
		return errors.New("provider URL is required")
	}

	return runInteractiveSession(ctx, opts)
}

func runInteractiveSession(ctx context.Context, opts Options) error {
	runtimeConfigPath, err := writeRuntimeConfig(opts)
	if err != nil {
		return err
	}

	host, err := sdk.New(ctx, &sdk.Options{
		Model:      "openai:" + strings.TrimSpace(opts.Model),
		ConfigFile: runtimeConfigPath,
		Streaming:  true,
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

	fmt.Fprintln(out, renderHeader("MCPBox llama.cpp Chat"))
	fmt.Fprintf(out, "%sModel:%s %s\n", styleDim(), styleReset(), strings.TrimSpace(opts.Model))
	fmt.Fprintf(out, "%sEndpoint:%s %s\n", styleDim(), styleReset(), strings.TrimSpace(opts.ProviderURL))
	fmt.Fprintln(out, styleDim()+"Type your prompt and press Enter. Use /exit to quit."+styleReset())

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Fprint(out, "\n"+renderPrompt())
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

		fmt.Fprintf(out, "\n%s\n", renderUserMessage(prompt))

		var streamed bool
		var assistantHeaderPrinted bool
		response, err := host.PromptWithCallbacks(
			ctx,
			prompt,
			func(name, args string) {
				if !assistantHeaderPrinted {
					fmt.Fprintf(out, "\n%s\n", renderAssistantHeader())
					assistantHeaderPrinted = true
				}
				fmt.Fprintf(out, "\n%s %s\n", renderToolLabel(), strings.TrimSpace(name))
				if trimmedArgs := strings.TrimSpace(args); trimmedArgs != "" {
					fmt.Fprintf(out, "%s%s%s\n", styleDim(), trimmedArgs, styleReset())
				}
			},
			func(name, _ string, _ string, isError bool) {
				label := renderToolResultLabel(isError)
				fmt.Fprintf(out, "%s %s\n", label, strings.TrimSpace(name))
			},
			func(chunk string) {
				if !assistantHeaderPrinted {
					fmt.Fprintf(out, "\n%s\n", renderAssistantHeader())
					assistantHeaderPrinted = true
				}
				streamed = true
				fmt.Fprint(out, chunk)
			},
		)
		if err != nil {
			fmt.Fprintf(out, "\n%s %v\n", renderErrorLabel(), err)
			continue
		}

		trimmedResponse := strings.TrimSpace(response)
		if !streamed && trimmedResponse != "" {
			fmt.Fprintf(out, "\n%s\n%s", renderAssistantHeader(), trimmedResponse)
		}
		if assistantHeaderPrinted || streamed || trimmedResponse != "" {
			fmt.Fprintln(out)
		}
	}
}

func writeRuntimeConfig(opts Options) (string, error) {
	baseConfigPath := strings.TrimSpace(opts.ConfigFile)
	baseConfig, err := os.ReadFile(baseConfigPath)
	if err != nil {
		return "", fmt.Errorf("read base mcphost config: %w", err)
	}

	configDir := filepath.Join(os.TempDir(), "mcpbox-llamacpp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("create llama.cpp runtime config directory: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("model: %q\n", "openai:"+strings.TrimSpace(opts.Model)))
	builder.WriteString(fmt.Sprintf("provider-url: %q\n", strings.TrimSpace(opts.ProviderURL)))
	builder.WriteString(fmt.Sprintf("provider-api-key: %q\n", strings.TrimSpace(opts.ProviderAPIKey)))
	if promptFile := strings.TrimSpace(opts.SystemPromptFile); promptFile != "" {
		builder.WriteString(fmt.Sprintf("system-prompt: %q\n", promptFile))
	}
	builder.WriteString(string(baseConfig))

	configPath := filepath.Join(configDir, fmt.Sprintf("runtime-%s.yml", sanitizeFileComponent(strings.TrimSpace(opts.Model))))
	if err := os.WriteFile(configPath, []byte(builder.String()), 0o600); err != nil {
		return "", fmt.Errorf("write llama.cpp runtime config: %w", err)
	}
	return configPath, nil
}

func sanitizeFileComponent(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	sanitized := replacer.Replace(value)
	if sanitized == "" {
		return "model"
	}
	return sanitized
}

func renderHeader(title string) string {
	return styleBold() + title + styleReset()
}

func renderPrompt() string {
	return styleUserBadge() + " "
}

func renderUserMessage(message string) string {
	return styleUserBadge() + " " + message
}

func renderAssistantHeader() string {
	return styleAssistantLabel()
}

func renderToolLabel() string {
	return styleToolLabel()
}

func renderToolResultLabel(isError bool) string {
	if isError {
		return styleErrorLabel()
	}
	return styleSuccessLabel()
}

func renderErrorLabel() string {
	return styleErrorLabel()
}

func styleUserBadge() string {
	return ansiBlueBG + ansiWhiteText + ansiBold + " YOU " + ansiReset + ansiCyan + ">" + ansiReset
}

func styleAssistantLabel() string {
	return ansiGreen + ansiBold + "Assistant:" + ansiReset
}

func styleToolLabel() string {
	return ansiYellow + ansiBold + "[tool]" + ansiReset
}

func styleSuccessLabel() string {
	return ansiGreen + ansiBold + "[tool-ok]" + ansiReset
}

func styleErrorLabel() string {
	return ansiRed + ansiBold + "[error]" + ansiReset
}

func styleBold() string {
	return ansiBold
}

func styleDim() string {
	return ansiDim
}

func styleReset() string {
	return ansiReset
}

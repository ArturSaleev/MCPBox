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

	fmt.Fprintln(out, renderHeader("MCPBox Ollama Chat"))
	fmt.Fprintf(out, "%sModel:%s %s\n", styleDim(), styleReset(), strings.TrimSpace(opts.Model))
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

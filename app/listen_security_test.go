package app

import (
	"flag"
	"os"
	"testing"
)

func TestMCPListenerDefaultsAndExplicitRemoteAccess(t *testing.T) {
	for _, tc := range []struct {
		name, env, edition, want string
		args                     []string
	}{
		{name: "default", want: "127.0.0.1"},
		{name: "edition remote override", edition: "0.0.0.0", want: "0.0.0.0"},
		{name: "environment remote override", env: "0.0.0.0", want: "0.0.0.0"},
		{name: "flag overrides environment", env: "0.0.0.0", args: []string{"--mcp-host", "127.0.0.1"}, want: "127.0.0.1"},
		{name: "explicit remote flag", args: []string{"--mcp-host", "0.0.0.0"}, want: "0.0.0.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previousFlags, previousArgs := flag.CommandLine, os.Args
			t.Cleanup(func() { flag.CommandLine = previousFlags; os.Args = previousArgs })
			flag.CommandLine = flag.NewFlagSet("listen-test", flag.ContinueOnError)
			os.Args = append([]string{"listen-test"}, tc.args...)
			t.Setenv("MCPBOX_MCP_HOST", tc.env)
			t.Setenv("MCPBOX_HOST", "")
			t.Setenv("MCPBOX_PORT", "")
			t.Setenv("MCPBOX_MCP_PORT", "")
			got := resolveListenConfig(Edition{MCPHost: tc.edition, AdminPort: 38181, MCPPort: 38182})
			if got.mcpHost != tc.want || got.mcpPort != 38182 || got.adminHost != "127.0.0.1" {
				t.Fatalf("unexpected listeners: %+v", got)
			}
		})
	}
}

func TestListenerAdvertisesOnlyExplicitBindAddress(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "192.0.2.10"} {
		got := listenAddresses(host, 38182, false)
		want := joinListenAddress(host, 38182)
		if len(got) != 1 || got[0] != "http://"+want+"/" {
			t.Fatalf("host %s advertises unreachable endpoints: %v", host, got)
		}
	}
}

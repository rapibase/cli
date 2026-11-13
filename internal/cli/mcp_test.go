package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rapibase/cli/internal/agents"
	"github.com/rapibase/cli/internal/cerr"
)

// The instance's own MCP next to the CLI's gives the agent every tool
// twice: without a yes the new one is not added; with --yes it replaces it.
func TestAddMCPReplacesTheOlderOneOnlyWithAYes(t *testing.T) {
	for _, yes := range []bool{false, true} {
		a, root, home := skillApp(t, yes, "")
		a.host = "https://rapibase.com"
		t.Setenv("CLAUDE_CONFIG_DIR", home)
		t.Setenv("PATH", "")
		mcp := filepath.Join(root, ".mcp.json")
		write := func(s string) {
			if err := os.WriteFile(mcp, []byte(s), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		write(`{"mcpServers":{"rapibase-shop-1a2b3c":{"type":"http","url":"https://shop.rapibase.com/mcp","headers":{"Authorization":"Bearer k"}}}}`)

		res, err := a.addMCP([]agents.Agent{agents.Claude}, agents.ScopeProject, "shop-1a2b3c", "shop")
		b, _ := os.ReadFile(mcp)
		if !yes {
			if err == nil || cerr.As(err).Code != cerr.MissingFlag {
				t.Fatalf("without --yes: %v", err)
			}
			if len(res) != 1 || len(res[0].Legacy) != 1 || res[0].Registered {
				t.Fatalf("result: %+v", res)
			}
			if strings.Contains(string(b), `"rapibase"`) || !strings.Contains(string(b), "rapibase-shop-1a2b3c") {
				t.Fatalf("nothing changes without --yes:\n%s", b)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 || !res[0].Registered || len(res[0].Replaced) != 1 || res[0].Replaced[0] != "rapibase-shop-1a2b3c" {
			t.Fatalf("result: %+v", res)
		}
		if strings.Contains(string(b), "rapibase-shop-1a2b3c") || !strings.Contains(string(b), `"https://rapibase.com/mcp?project=shop-1a2b3c"`) {
			t.Fatalf("with --yes:\n%s", b)
		}
	}
}

package main

import (
	"fmt"
	"strings"

	"github.com/ivalx1s/skill-agent-facing-api/agentquery"
	"github.com/ivalx1s/skill-agent-facing-api/agentquery/cobraext"
	"github.com/ivalx1s/skill-confluence-manager/internal/query"
	"github.com/spf13/cobra"
)

// newQueryCommand builds the "q" command using agentquery.Schema + cobraext.
// The schema is constructed lazily (on first RunE) because the Confluence client
// requires auth config that may not be available at init() time.
func newQueryCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "q '<dsl-query>'",
		Short: "Execute DSL query (agent-facing reads)",
		Long: `Execute one or more DSL queries against Confluence.

Operations:
  get(PAGE_ID)                      — Get page by ID
  get(space=KEY, title="Title")     — Get page by space+title
  list(space=KEY)                   — List pages in space
  list(space=KEY, label=NAME)       — List pages with label (CQL)
  search("CQL query")              — CQL search
  children(PAGE_ID)                 — Direct children
  ancestors(PAGE_ID)                — Breadcrumb chain
  tree(PAGE_ID)                     — Recursive children (default depth=3)
  tree(PAGE_ID, depth=5)            — Recursive children with depth
  spaces()                          — List all spaces
  schema()                          — Show available operations, fields, presets

Field presets: minimal, default, overview, full

Examples:
  confluence-mgmt q 'spaces(){minimal}' --format json
  confluence-mgmt q 'get(12345){full}' --format json
  confluence-mgmt q 'list(space=DEV){default}' --format json
  confluence-mgmt q 'search("type=page AND space=DEV AND text~\"API\""){default}' --format json
  confluence-mgmt q 'children(12345){minimal}; ancestors(12345){minimal}' --format json
  confluence-mgmt q 'schema()' --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := buildConfluenceClientFromConfig()
			if err != nil {
				return err
			}

			schema := query.NewSchema(client)

			mode, err := parseOutputMode(format)
			if err != nil {
				return err
			}

			data, err := schema.QueryJSONWithMode(args[0], mode)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", `Output format: "json" or "compact"/"llm"`)
	return cmd
}

// parseOutputMode converts a string flag value to an agentquery.OutputMode.
func parseOutputMode(s string) (agentquery.OutputMode, error) {
	switch strings.ToLower(s) {
	case "compact", "llm":
		return agentquery.LLMReadable, nil
	case "json":
		return agentquery.HumanReadable, nil
	default:
		return 0, fmt.Errorf("unknown format %q: use \"json\", \"compact\", or \"llm\"", s)
	}
}

// init registers the query command.
// We intentionally don't use cobraext.QueryCommand directly because:
// 1. The Confluence client must be constructed lazily (needs auth config).
// 2. cobraext.QueryCommand marks --format as required; we want a default of "json".
func init() {
	rootCmd.AddCommand(newQueryCommand())
}

// Ensure cobraext is available (compile-time check).
var _ = cobraext.QueryCommand[any]

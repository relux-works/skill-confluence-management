package main

import (
	"encoding/json"
	"fmt"

	"github.com/ivalx1s/skill-confluence-manager/internal/query"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
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

Field presets: minimal, default, overview, full

Examples:
  confluence-mgmt q 'spaces(){minimal}'
  confluence-mgmt q 'get(12345){full}'
  confluence-mgmt q 'list(space=DEV){default}'
  confluence-mgmt q 'search("type=page AND space=DEV AND text~\"API\""){default}'
  confluence-mgmt q 'children(12345){minimal}; ancestors(12345){minimal}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parsed, err := query.ParseQuery(args[0])
		if err != nil {
			return fmt.Errorf("parse error: %w", err)
		}

		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		executor := query.NewExecutor(client)
		results, err := executor.Execute(parsed)
		if err != nil {
			return err
		}

		// Single result — unwrap array.
		var output interface{}
		if len(results) == 1 {
			output = results[0]
		} else {
			output = results
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
}

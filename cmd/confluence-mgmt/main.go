package main

import (
	"fmt"
	"os"

	"github.com/ivalx1s/skill-confluence-manager/internal/config"
	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global flags.
var (
	flagSpace  string
	flagFormat string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "confluence-mgmt",
	Short: "Confluence management CLI for AI agents",
	Long:  "Agent-facing CLI for Confluence Cloud and Server/DC: DSL queries, scoped grep, and write commands.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: persistentPreRun,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "confluence-mgmt %s\n  commit: %s\n  built:  %s\n", version, commit, date)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagSpace, "space", "", "Confluence space key (overrides config)")
	rootCmd.PersistentFlags().StringVar(&flagFormat, "format", "json", "Output format: json, compact, or text")

	rootCmd.AddCommand(versionCmd)
}

func persistentPreRun(cmd *cobra.Command, args []string) error {
	loadConfigDefaults(cmd)

	// Skip first-run check for commands that don't need auth.
	name := cmd.Name()
	if name == "version" || name == "auth" || name == "config" || name == "set" || name == "show" || name == "help" || name == "confluence-mgmt" {
		return nil
	}

	return checkFirstRun()
}

// loadConfigDefaults fills global flags from config when not set explicitly via CLI.
func loadConfigDefaults(cmd *cobra.Command) {
	mgr, err := config.NewConfigManager()
	if err != nil {
		return
	}

	cfg, err := mgr.GetConfig()
	if err != nil {
		return
	}

	if !cmd.Flags().Changed("space") && cfg.ActiveSpace != "" {
		flagSpace = cfg.ActiveSpace
	}
}

// checkFirstRun verifies that authentication is configured.
func checkFirstRun() error {
	cfgMgr, err := config.NewConfigManager()
	if err != nil {
		return nil
	}

	cfg, err := cfgMgr.GetConfig()
	if err != nil {
		return nil
	}

	if cfg.InstanceURL == "" {
		return fmt.Errorf("confluence-mgmt is not configured\nRun 'confluence-mgmt auth' to set up authentication")
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/relux-works/skill-confluence-management/internal/config"
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
	flagSpace    string
	flagFormat   string
	flagInsecure bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:               "confluence-mgmt",
	Short:             "Confluence management CLI for AI agents",
	Long:              "Agent-facing CLI for Confluence Cloud and Server/DC: DSL queries, scoped grep, and write commands.",
	SilenceUsage:      true,
	SilenceErrors:     true,
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
	rootCmd.PersistentFlags().BoolVar(&flagInsecure, "insecure", false, "Skip TLS certificate verification (for corporate CAs)")

	rootCmd.AddCommand(versionCmd)
}

func persistentPreRun(cmd *cobra.Command, args []string) error {
	loadConfigDefaults(cmd)

	if shouldSkipAuthCheck(cmd) {
		return nil
	}

	return checkFirstRun()
}

func shouldSkipAuthCheck(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Name() == "confluence-mgmt" {
		return true
	}

	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "auth", "config", "help", "completion", "version":
			return true
		}
	}

	return false
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

	resolver := getCredentialResolver()
	instanceURL := resolver.ResolveInstanceURL(cfg.InstanceURL)
	if _, err := resolver.Resolve(config.SourceAuto, instanceURL); err == nil {
		return nil
	} else if errors.Is(err, config.ErrCredentialsNotFound) || errors.Is(err, config.ErrInstanceURLRequired) {
		return fmt.Errorf("confluence-mgmt is not configured\nRun 'confluence-mgmt auth set-access' to set up authentication")
	} else {
		return err
	}
}

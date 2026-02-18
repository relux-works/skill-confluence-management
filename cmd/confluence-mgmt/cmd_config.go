package main

import (
	"fmt"

	"github.com/relux-works/skill-confluence-management/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage confluence-mgmt configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Keys:
  space            — active Confluence space key (e.g. DEV)
  tls_skip_verify  — skip TLS cert verification: true/false (for corporate CAs)

Examples:
  confluence-mgmt config set space DEV
  confluence-mgmt config set tls_skip_verify true`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfgMgr, err := config.NewConfigManager()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		switch key {
		case "space":
			if err := cfgMgr.SetActiveSpace(value); err != nil {
				return err
			}
			fmt.Fprintf(out, "Active space set to %s\n", value)

		case "tls_skip_verify":
			skip := value == "true" || value == "1" || value == "yes"
			if err := cfgMgr.SetTLSSkipVerify(skip); err != nil {
				return err
			}
			fmt.Fprintf(out, "TLS skip verify set to %v\n", skip)

		default:
			return fmt.Errorf("unknown config key %q (supported: space, tls_skip_verify)", key)
		}

		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgMgr, err := config.NewConfigManager()
		if err != nil {
			return err
		}

		cfg, err := cfgMgr.GetConfig()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Configuration")
		fmt.Fprintln(out, "=============")
		fmt.Fprintf(out, "  config file:    %s\n", cfgMgr.ConfigPath())
		fmt.Fprintf(out, "  instance:       %s\n", valueOrNone(cfg.InstanceURL))
		fmt.Fprintf(out, "  instance type:  %s\n", valueOrNone(cfg.InstanceType))
		fmt.Fprintf(out, "  auth type:      %s\n", valueOrNone(cfg.AuthType))
		fmt.Fprintf(out, "  active space:   %s\n", valueOrNone(cfg.ActiveSpace))
		fmt.Fprintf(out, "  tls skip verify: %v\n", cfg.TLSSkipVerify)

		return nil
	},
}

func valueOrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

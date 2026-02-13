package main

import (
	"fmt"

	"github.com/ivalx1s/skill-confluence-manager/internal/config"
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
  space  — active Confluence space key (e.g. DEV)

Examples:
  confluence-mgmt config set space DEV`,
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

		default:
			return fmt.Errorf("unknown config key %q (supported: space)", key)
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

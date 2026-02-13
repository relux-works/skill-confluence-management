package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Label operations (add, remove)",
}

var labelAddLabels string

var labelAddCmd = &cobra.Command{
	Use:   "add <page-id>",
	Short: "Add labels to a page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		labels := strings.Split(labelAddLabels, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}

		if err := client.AddLabels(args[0], labels); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Labels added to page %s: %s\n", args[0], labelAddLabels)
		return nil
	},
}

var labelRemoveLabels string

var labelRemoveCmd = &cobra.Command{
	Use:   "remove <page-id>",
	Short: "Remove labels from a page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		labels := strings.Split(labelRemoveLabels, ",")
		for _, l := range labels {
			l = strings.TrimSpace(l)
			if err := client.RemoveLabel(args[0], l); err != nil {
				return fmt.Errorf("removing label %q: %w", l, err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Labels removed from page %s: %s\n", args[0], labelRemoveLabels)
		return nil
	},
}

func init() {
	labelAddCmd.Flags().StringVar(&labelAddLabels, "labels", "", "Comma-separated labels to add")
	labelRemoveCmd.Flags().StringVar(&labelRemoveLabels, "labels", "", "Comma-separated labels to remove")

	labelCmd.AddCommand(labelAddCmd)
	labelCmd.AddCommand(labelRemoveCmd)
	rootCmd.AddCommand(labelCmd)
}

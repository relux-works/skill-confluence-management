package main

import (
	"github.com/spf13/cobra"
)

var spaceCmd = &cobra.Command{
	Use:   "space",
	Short: "Space operations",
}

var spaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible spaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		spaces, err := client.ListSpaces(0)
		if err != nil {
			return err
		}

		return outputResult(cmd, spaces)
	},
}

func init() {
	spaceCmd.AddCommand(spaceListCmd)
	rootCmd.AddCommand(spaceCmd)
}

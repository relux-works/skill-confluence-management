package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Page operations (create, update, delete)",
}

// --- page create ---

var (
	pageCreateSpace    string
	pageCreateTitle    string
	pageCreateBody     string
	pageCreateBodyFile string
	pageCreateParent   string
)

var pageCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new page",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		space := pageCreateSpace
		if space == "" {
			space = flagSpace
		}
		if space == "" {
			return fmt.Errorf("space is required (use --space flag or 'config set space')")
		}

		body := pageCreateBody
		if pageCreateBodyFile != "" {
			data, err := os.ReadFile(pageCreateBodyFile)
			if err != nil {
				return fmt.Errorf("reading body file: %w", err)
			}
			body = string(data)
		}

		page, err := client.CreatePage(space, pageCreateTitle, body, pageCreateParent)
		if err != nil {
			return err
		}

		return outputResult(cmd, page)
	},
}

// --- page update ---

var (
	pageUpdateTitle   string
	pageUpdateBody    string
	pageUpdateBodyFile string
	pageUpdateMessage string
)

var pageUpdateCmd = &cobra.Command{
	Use:   "update <page-id>",
	Short: "Update an existing page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		body := pageUpdateBody
		if pageUpdateBodyFile != "" {
			data, err := os.ReadFile(pageUpdateBodyFile)
			if err != nil {
				return fmt.Errorf("reading body file: %w", err)
			}
			body = string(data)
		}

		page, err := client.UpdatePage(args[0], pageUpdateTitle, body, pageUpdateMessage)
		if err != nil {
			return err
		}

		return outputResult(cmd, page)
	},
}

// --- page delete ---

var pageDeleteCmd = &cobra.Command{
	Use:   "delete <page-id>",
	Short: "Delete (trash) a page",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		if err := client.DeletePage(args[0]); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Page %s deleted\n", args[0])
		return nil
	},
}

// --- page get ---

var pageGetBody bool

var pageGetCmd = &cobra.Command{
	Use:   "get <page-id>",
	Short: "Get a page by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := buildConfluenceClientFromConfig()
		if err != nil {
			return err
		}

		page, err := client.GetPage(args[0], pageGetBody)
		if err != nil {
			return err
		}

		return outputResult(cmd, page)
	},
}

func outputResult(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func init() {
	pageCreateCmd.Flags().StringVar(&pageCreateSpace, "space", "", "Space key")
	pageCreateCmd.Flags().StringVar(&pageCreateTitle, "title", "", "Page title")
	pageCreateCmd.Flags().StringVar(&pageCreateBody, "body", "", "Page body (storage format)")
	pageCreateCmd.Flags().StringVar(&pageCreateBodyFile, "body-file", "", "Read body from file")
	pageCreateCmd.Flags().StringVar(&pageCreateParent, "parent", "", "Parent page ID")

	pageUpdateCmd.Flags().StringVar(&pageUpdateTitle, "title", "", "New title")
	pageUpdateCmd.Flags().StringVar(&pageUpdateBody, "body", "", "New body (storage format)")
	pageUpdateCmd.Flags().StringVar(&pageUpdateBodyFile, "body-file", "", "Read body from file")
	pageUpdateCmd.Flags().StringVar(&pageUpdateMessage, "message", "", "Version message")

	pageGetCmd.Flags().BoolVar(&pageGetBody, "body", false, "Include page body in response")

	pageCmd.AddCommand(pageCreateCmd)
	pageCmd.AddCommand(pageUpdateCmd)
	pageCmd.AddCommand(pageDeleteCmd)
	pageCmd.AddCommand(pageGetCmd)
	rootCmd.AddCommand(pageCmd)
}

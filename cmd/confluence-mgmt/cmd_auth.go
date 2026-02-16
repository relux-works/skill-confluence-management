package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/relux-works/skill-confluence-management/internal/config"
	"github.com/spf13/cobra"
)

var (
	authFlagInstance string
	authFlagEmail    string
	authFlagToken    string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure Confluence authentication",
	Long: `Setup Confluence authentication (Cloud or Server/DC).

Cloud (Basic auth — email + API token, same token as Jira):
  confluence-mgmt auth --instance https://mycompany.atlassian.net/wiki --email user@company.com --token API_TOKEN

Server/DC (Bearer auth — Personal Access Token):
  confluence-mgmt auth --instance https://confluence.company.com --token PAT_TOKEN

Interactive (prompts for input):
  confluence-mgmt auth`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		instanceURL := authFlagInstance
		email := authFlagEmail
		apiToken := authFlagToken

		// If flags not provided, fall back to interactive prompts
		if instanceURL == "" || apiToken == "" {
			reader := bufio.NewReader(os.Stdin)

			fmt.Fprintln(out, "Confluence Authentication Setup")
			fmt.Fprintln(out, "===============================")
			fmt.Fprintln(out)

			if instanceURL == "" {
				fmt.Fprint(out, "Instance URL (e.g. https://mycompany.atlassian.net/wiki): ")
				line, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				instanceURL = strings.TrimSpace(line)
			}

			if email == "" {
				fmt.Fprint(out, "Email (leave empty for Server/DC PAT auth): ")
				line, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				email = strings.TrimSpace(line)
			}

			if apiToken == "" {
				fmt.Fprint(out, "API Token / PAT: ")
				line, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				apiToken = strings.TrimSpace(line)
			}
		}

		// Determine auth type from presence of email.
		authType := "basic"
		if email == "" {
			authType = "bearer"
		}

		creds := config.Credentials{
			InstanceURL: instanceURL,
			Email:       email,
			APIToken:    apiToken,
			AuthType:    authType,
		}

		if err := creds.Validate(); err != nil {
			return fmt.Errorf("invalid credentials: %w", err)
		}

		// Detect instance type from URL.
		instanceType := "server"
		if strings.Contains(instanceURL, ".atlassian.net") {
			instanceType = "cloud"
		}

		fmt.Fprintf(out, "Auth method: %s\n", authType)
		fmt.Fprintf(out, "Instance type: %s (detected from URL)\n", instanceType)

		// Save credentials
		store := getCredentialStore()
		if err := store.Save(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}

		// Save instance URL and type to config
		cfgMgr, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("config manager: %w", err)
		}
		_ = cfgMgr.SetInstanceURL(instanceURL)
		_ = cfgMgr.SetInstanceType(instanceType)
		_ = cfgMgr.SetAuthType(authType)

		fmt.Fprintln(out, "Authentication configured successfully.")
		fmt.Fprintln(out, "Credentials stored in OS keychain (service: atlassian-mgmt)")
		return nil
	},
}

func init() {
	authCmd.Flags().StringVar(&authFlagInstance, "instance", "", "Confluence instance URL (e.g. https://mycompany.atlassian.net/wiki)")
	authCmd.Flags().StringVar(&authFlagEmail, "email", "", "Atlassian account email (Cloud only)")
	authCmd.Flags().StringVar(&authFlagToken, "token", "", "API token (Cloud) or PAT (Server/DC)")
	rootCmd.AddCommand(authCmd)
}

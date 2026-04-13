package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/relux-works/skill-confluence-management/internal/config"
	"github.com/relux-works/skill-confluence-management/internal/confluence"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type authSetAccessFlags struct {
	Instance string
	Email    string
	Token    string
	Source   string
	Check    bool
}

type authLookupFlags struct {
	Instance string
	Source   string
}

type authWhoamiFlags struct {
	Instance string
	Source   string
	Check    bool
}

var (
	authCompatOptions    authSetAccessFlags
	authSetAccessOptions authSetAccessFlags
	authResolveOptions   authLookupFlags
	authCleanOptions     authLookupFlags
	authWhoamiOptions    authWhoamiFlags
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Confluence authentication",
	Long: `Manage Confluence authentication for Cloud or Server/DC.

Primary flow:
  confluence-mgmt auth set-access --instance https://company.atlassian.net/wiki --email user@company.com --token API_TOKEN
  confluence-mgmt auth whoami
  confluence-mgmt auth resolve
  confluence-mgmt auth clean

Compatibility:
  confluence-mgmt auth --instance URL --email EMAIL --token TOKEN

Cloud uses Basic auth (email + API token).
Server/DC PAT uses Bearer auth (token without email).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthSetAccess(cmd, authCompatOptions)
	},
}

var authSetAccessCmd = &cobra.Command{
	Use:   "set-access",
	Short: "Store Confluence credentials in the platform default secret backend",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthSetAccess(cmd, authSetAccessOptions)
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show resolved Confluence auth state and optionally validate it",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthWhoami(cmd, authWhoamiOptions)
	},
}

var authResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Show where Confluence credentials would resolve from",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthResolve(cmd, authResolveOptions)
	},
}

var authCleanCmd = &cobra.Command{
	Use:     "clean",
	Aliases: []string{"clear-access"},
	Short:   "Remove stored Confluence credentials for the selected instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuthClean(cmd, authCleanOptions)
	},
}

var authConfigPathCmd = &cobra.Command{
	Use:   "config-path",
	Short: "Print the global auth.json path",
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := getCredentialResolver()
		path, err := resolver.AuthConfigPath()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}

func bindAuthSetAccessFlags(fs *pflag.FlagSet, target *authSetAccessFlags) {
	fs.StringVar(&target.Instance, "instance", "", "Confluence instance URL (e.g. https://company.atlassian.net/wiki)")
	fs.StringVar(&target.Email, "email", "", "Confluence account email (omit for Server/DC PAT auth)")
	fs.StringVar(&target.Token, "token", "", "Confluence API token or PAT")
	fs.StringVar(&target.Source, "source", string(config.SourceAuto), "Credential source: auto, keychain, env_or_file")
	fs.BoolVar(&target.Check, "check", false, "Validate credentials immediately with a live Confluence read")
}

func bindAuthLookupFlags(fs *pflag.FlagSet, target *authLookupFlags) {
	fs.StringVar(&target.Instance, "instance", "", "Confluence instance URL (defaults to configured value or env override)")
	fs.StringVar(&target.Source, "source", string(config.SourceAuto), "Credential source: auto, keychain, env_or_file")
}

func bindAuthWhoamiFlags(fs *pflag.FlagSet, target *authWhoamiFlags) {
	fs.StringVar(&target.Instance, "instance", "", "Confluence instance URL (defaults to configured value or env override)")
	fs.StringVar(&target.Source, "source", string(config.SourceAuto), "Credential source: auto, keychain, env_or_file")
	fs.BoolVar(&target.Check, "check", true, "Run a live Confluence auth probe")
}

func runAuthSetAccess(cmd *cobra.Command, opts authSetAccessFlags) error {
	out := cmd.OutOrStdout()

	instanceURL, email, apiToken, err := promptForCredentials(out, opts.Instance, opts.Email, opts.Token)
	if err != nil {
		return err
	}

	creds := config.Credentials{
		InstanceURL: instanceURL,
		Email:       email,
		APIToken:    apiToken,
	}
	if err := creds.Validate(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	resolver := getCredentialResolver()
	result, err := resolver.SetAccess(config.Source(opts.Source), creds)
	if err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	cfgMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("config manager: %w", err)
	}
	instanceType := inferInstanceType(result.Credentials.InstanceURL)
	_ = cfgMgr.SetInstanceURL(result.Credentials.InstanceURL)
	_ = cfgMgr.SetAuthType(result.Credentials.AuthType)
	_ = cfgMgr.SetInstanceType(string(instanceType))

	fmt.Fprintln(out, "Credentials stored.")
	fmt.Fprintf(out, "  instance: %s\n", result.Credentials.InstanceURL)
	fmt.Fprintf(out, "  auth type: %s\n", result.Credentials.AuthType)
	fmt.Fprintf(out, "  instance type: %s\n", instanceType)
	fmt.Fprintf(out, "  source: %s\n", result.Source)
	switch result.Source {
	case config.SourceKeychain:
		fmt.Fprintf(out, "  keychain service: %s\n", result.KeychainService)
		fmt.Fprintf(out, "  keychain account: %s\n", result.KeychainAccount)
	case config.SourceEnvOrFile:
		fmt.Fprintf(out, "  auth file: %s\n", result.ConfigPath)
		fmt.Fprintf(out, "  profile key: %s\n", result.ProfileKey)
	}

	if opts.Check {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Validating credentials...")
		instanceType, err := validateCredentials(result.Credentials, flagInsecure)
		if err != nil {
			return err
		}
		_ = cfgMgr.SetInstanceType(string(instanceType))
		fmt.Fprintf(out, "  auth probe: ok\n")
		fmt.Fprintf(out, "  instance type: %s\n", instanceType)
	} else {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Run 'confluence-mgmt auth whoami' to validate stored credentials.")
	}

	return nil
}

func runAuthResolve(cmd *cobra.Command, opts authLookupFlags) error {
	out := cmd.OutOrStdout()

	resolver := getCredentialResolver()
	instanceURL, _ := configuredInstanceURL(resolver, opts.Instance)
	resolved, err := resolver.Resolve(config.Source(opts.Source), instanceURL)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	printResolvedCredentials(out, resolved)
	return nil
}

func runAuthWhoami(cmd *cobra.Command, opts authWhoamiFlags) error {
	out := cmd.OutOrStdout()

	resolver := getCredentialResolver()
	instanceURL, _ := configuredInstanceURL(resolver, opts.Instance)
	resolved, err := resolver.Resolve(config.Source(opts.Source), instanceURL)
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	printResolvedCredentials(out, resolved)

	if !opts.Check {
		fmt.Fprintln(out, "  auth probe: skipped (--check=false)")
		return nil
	}

	instanceType, err := validateCredentials(resolved.Credentials, flagInsecure)
	if err != nil {
		return err
	}

	cfgMgr, err := config.NewConfigManager()
	if err == nil && resolved.ResolvedFrom != "env" {
		_ = cfgMgr.SetInstanceURL(resolved.Credentials.InstanceURL)
		_ = cfgMgr.SetAuthType(resolved.Credentials.AuthType)
		_ = cfgMgr.SetInstanceType(string(instanceType))
	}

	fmt.Fprintf(out, "  auth probe: ok\n")
	fmt.Fprintf(out, "  instance type: %s\n", instanceType)
	return nil
}

func runAuthClean(cmd *cobra.Command, opts authLookupFlags) error {
	out := cmd.OutOrStdout()

	resolver := getCredentialResolver()
	instanceURL, _ := configuredInstanceURL(resolver, opts.Instance)
	result, err := resolver.Clear(config.Source(opts.Source), instanceURL)
	if err != nil {
		return fmt.Errorf("clearing credentials: %w", err)
	}

	if !result.Removed {
		fmt.Fprintln(out, "No stored credentials were removed.")
		return nil
	}

	fmt.Fprintln(out, "Removed stored credentials.")
	if len(result.RemovedFrom) > 0 {
		fmt.Fprintf(out, "  removed from: %s\n", joinSources(result.RemovedFrom))
	}
	if result.KeychainAccount != "" {
		fmt.Fprintf(out, "  keychain account: %s\n", result.KeychainAccount)
	}
	if result.ProfileKey != "" {
		fmt.Fprintf(out, "  profile key: %s\n", result.ProfileKey)
	}
	if result.ConfigPath != "" {
		fmt.Fprintf(out, "  auth file: %s\n", result.ConfigPath)
	}
	return nil
}

func promptForCredentials(out io.Writer, instanceURL, email, apiToken string) (string, string, string, error) {
	if instanceURL != "" && apiToken != "" {
		return strings.TrimSpace(instanceURL), strings.TrimSpace(email), strings.TrimSpace(apiToken), nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintln(out, "Confluence Authentication Setup")
	fmt.Fprintln(out, "==============================")
	fmt.Fprintln(out)

	var err error
	if strings.TrimSpace(instanceURL) == "" {
		fmt.Fprint(out, "Instance URL (e.g. https://company.atlassian.net/wiki): ")
		instanceURL, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", fmt.Errorf("reading input: %w", err)
		}
	}

	if strings.TrimSpace(email) == "" {
		fmt.Fprint(out, "Email (leave empty for Server/DC PAT auth): ")
		email, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", fmt.Errorf("reading input: %w", err)
		}
	}

	if strings.TrimSpace(apiToken) == "" {
		fmt.Fprint(out, "API Token / PAT: ")
		apiToken, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", fmt.Errorf("reading input: %w", err)
		}
	}

	return strings.TrimSpace(instanceURL), strings.TrimSpace(email), strings.TrimSpace(apiToken), nil
}

func validateCredentials(creds config.Credentials, insecure bool) (confluence.InstanceType, error) {
	instanceType := inferInstanceType(creds.InstanceURL)
	client, err := confluence.NewClient(confluence.Config{
		BaseURL:            creds.InstanceURL,
		Email:              creds.Email,
		Token:              creds.APIToken,
		AuthType:           confluence.AuthType(creds.AuthType),
		InstanceType:       instanceType,
		InsecureSkipVerify: insecure,
	})
	if err != nil {
		return "", fmt.Errorf("creating client: %w", err)
	}

	if _, err := client.ListSpaces(1); err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	return instanceType, nil
}

func inferInstanceType(instanceURL string) confluence.InstanceType {
	if strings.Contains(strings.ToLower(strings.TrimSpace(instanceURL)), ".atlassian.net") {
		return confluence.InstanceCloud
	}
	return confluence.InstanceServer
}

func configuredInstanceURL(resolver *config.Resolver, explicit string) (string, config.Config) {
	cfgMgr, err := config.NewConfigManager()
	if err != nil {
		return resolver.ResolveInstanceURL(explicit), config.DefaultConfig()
	}
	cfg, err := cfgMgr.GetConfig()
	if err != nil {
		return resolver.ResolveInstanceURL(explicit), config.DefaultConfig()
	}
	if strings.TrimSpace(explicit) != "" {
		return resolver.ResolveInstanceURL(explicit), cfg
	}
	return resolver.ResolveInstanceURL(cfg.InstanceURL), cfg
}

func printResolvedCredentials(out io.Writer, resolved config.ResolvedCredentials) {
	fmt.Fprintln(out, "Resolved Credentials")
	fmt.Fprintln(out, "====================")
	fmt.Fprintf(out, "  instance: %s\n", valueOrNone(resolved.Credentials.InstanceURL))
	fmt.Fprintf(out, "  email: %s\n", valueOrNone(resolved.Credentials.Email))
	fmt.Fprintf(out, "  auth type: %s\n", valueOrNone(resolved.Credentials.AuthType))
	fmt.Fprintf(out, "  source: %s\n", resolved.Source)
	fmt.Fprintf(out, "  resolved from: %s\n", resolved.ResolvedFrom)
	fmt.Fprintf(out, "  inferred instance type: %s\n", inferInstanceType(resolved.Credentials.InstanceURL))
	if resolved.ConfigPath != "" {
		fmt.Fprintf(out, "  auth file: %s\n", resolved.ConfigPath)
	}
	if resolved.KeychainService != "" {
		fmt.Fprintf(out, "  keychain service: %s\n", resolved.KeychainService)
	}
	if resolved.KeychainAccount != "" {
		fmt.Fprintf(out, "  keychain account: %s\n", resolved.KeychainAccount)
	}
	if resolved.ProfileKey != "" {
		fmt.Fprintf(out, "  profile key: %s\n", resolved.ProfileKey)
	}
}

func joinSources(sources []config.Source) string {
	if len(sources) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		parts = append(parts, string(source))
	}
	return strings.Join(parts, ", ")
}

func init() {
	bindAuthSetAccessFlags(authCmd.Flags(), &authCompatOptions)
	bindAuthSetAccessFlags(authSetAccessCmd.Flags(), &authSetAccessOptions)
	bindAuthWhoamiFlags(authWhoamiCmd.Flags(), &authWhoamiOptions)
	bindAuthLookupFlags(authResolveCmd.Flags(), &authResolveOptions)
	bindAuthLookupFlags(authCleanCmd.Flags(), &authCleanOptions)

	authCmd.AddCommand(authSetAccessCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authResolveCmd)
	authCmd.AddCommand(authCleanCmd)
	authCmd.AddCommand(authConfigPathCmd)
	rootCmd.AddCommand(authCmd)
}

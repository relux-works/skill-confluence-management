package main

import (
	"fmt"

	"github.com/relux-works/skill-confluence-management/internal/config"
	"github.com/relux-works/skill-confluence-management/internal/confluence"
	"github.com/zalando/go-keyring"
)

// getCredentialResolver returns the platform-aware credential resolver.
// Tests can override it.
var getCredentialResolver = func() *config.Resolver {
	return config.NewResolver(
		config.Runtime{},
		config.NewKeychainStore(
			keyring.Set,
			keyring.Get,
			keyring.Delete,
		),
	)
}

// buildConfluenceClientFromConfig creates a Confluence client from stored config and credentials.
func buildConfluenceClientFromConfig() (*confluence.Client, error) {
	cfgMgr, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("config manager: %w", err)
	}

	cfg, err := cfgMgr.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	resolver := getCredentialResolver()
	instanceURL := resolver.ResolveInstanceURL(cfg.InstanceURL)
	if instanceURL == "" {
		return nil, fmt.Errorf("not configured: run 'confluence-mgmt auth set-access' first")
	}

	resolved, err := resolver.Resolve(config.SourceAuto, instanceURL)
	if err != nil {
		return nil, fmt.Errorf("loading credentials: %w (run 'confluence-mgmt auth set-access' to configure)", err)
	}

	instanceType := cfg.InstanceType
	if instanceType == "" {
		instanceType = string(inferInstanceType(resolved.Credentials.InstanceURL))
	}

	client, err := confluence.NewClient(confluence.Config{
		BaseURL:            resolved.Credentials.InstanceURL,
		Email:              resolved.Credentials.Email,
		Token:              resolved.Credentials.APIToken,
		InstanceType:       confluence.InstanceType(instanceType),
		AuthType:           confluence.AuthType(resolved.Credentials.AuthType),
		InsecureSkipVerify: flagInsecure || cfg.TLSSkipVerify,
	})
	if err != nil {
		return nil, err
	}

	if cfg.InstanceType == "" {
		_ = cfgMgr.SetInstanceType(instanceType)
	}
	if cfg.AuthType == "" && resolved.ResolvedFrom != "env" {
		_ = cfgMgr.SetAuthType(resolved.Credentials.AuthType)
	}

	return client, nil
}

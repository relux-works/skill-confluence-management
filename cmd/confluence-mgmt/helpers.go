package main

import (
	"fmt"

	"github.com/ivalx1s/skill-confluence-manager/internal/config"
	"github.com/ivalx1s/skill-confluence-manager/internal/confluence"
	"github.com/zalando/go-keyring"
)

// getCredentialStore returns the keychain-backed credential store.
var getCredentialStore = func() config.CredentialStore {
	return config.NewKeychainStore(
		keyring.Set,
		keyring.Get,
		keyring.Delete,
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

	if cfg.InstanceURL == "" {
		return nil, fmt.Errorf("not configured: run 'confluence-mgmt auth' first")
	}

	store := getCredentialStore()
	creds, err := store.Load(cfg.InstanceURL)
	if err != nil {
		return nil, fmt.Errorf("loading credentials: %w (run 'confluence-mgmt auth' to configure)", err)
	}

	return confluence.NewClient(confluence.Config{
		BaseURL:      creds.InstanceURL,
		Email:        creds.Email,
		Token:        creds.APIToken,
		InstanceType: confluence.InstanceType(cfg.InstanceType),
		AuthType:     confluence.AuthType(cfg.AuthType),
	})
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

// serviceName is the shared keychain service identifier for all Atlassian CLI credentials.
// Used by both confluence-mgmt and jira-mgmt.
const serviceName = "atlassian-mgmt"

// Credentials holds the authentication details for a Confluence instance.
type Credentials struct {
	InstanceURL string `json:"instance_url"`
	Email       string `json:"email,omitempty"`     // Required for Basic auth (Cloud), empty for Bearer (Server/DC PAT)
	APIToken    string `json:"api_token"`
	AuthType    string `json:"auth_type,omitempty"` // "basic" or "bearer"
}

// Validate checks that all required credential fields are populated.
func (c Credentials) Validate() error {
	if c.InstanceURL == "" {
		return errors.New("instance URL is required")
	}
	if c.APIToken == "" {
		return errors.New("API token is required")
	}
	// Email is only required for Basic auth (Cloud).
	if c.AuthType != "bearer" && c.Email == "" {
		return errors.New("email is required for basic auth (use --token without --email for PAT/bearer auth)")
	}
	return nil
}

// CredentialStore defines the interface for persisting and retrieving credentials.
type CredentialStore interface {
	// Save persists credentials. Overwrites existing credentials for the same instance URL.
	Save(creds Credentials) error
	// Load retrieves credentials for the given instance URL.
	Load(instanceURL string) (Credentials, error)
	// Delete removes credentials for the given instance URL.
	Delete(instanceURL string) error
}

// ErrCredentialsNotFound is returned when no credentials exist for a given instance URL.
var ErrCredentialsNotFound = errors.New("credentials not found")

// KeychainStore implements CredentialStore using the OS keychain via go-keyring.
type KeychainStore struct {
	keyringSet    func(service, user, password string) error
	keyringGet    func(service, user string) (string, error)
	keyringDelete func(service, user string) error
}

// NewKeychainStore creates a KeychainStore with the provided keyring functions.
// In production, pass go-keyring's Set, Get, Delete functions directly.
func NewKeychainStore(
	setFn func(service, user, password string) error,
	getFn func(service, user string) (string, error),
	deleteFn func(service, user string) error,
) *KeychainStore {
	return &KeychainStore{
		keyringSet:    setFn,
		keyringGet:    getFn,
		keyringDelete: deleteFn,
	}
}

// Save serializes credentials to JSON and stores them in the keychain.
func (k *KeychainStore) Save(creds Credentials) error {
	if err := creds.Validate(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	if err := k.keyringSet(serviceName, creds.InstanceURL, string(data)); err != nil {
		return fmt.Errorf("saving to keychain: %w", err)
	}

	return nil
}

// Load retrieves and deserializes credentials from the keychain for the given instance URL.
func (k *KeychainStore) Load(instanceURL string) (Credentials, error) {
	if instanceURL == "" {
		return Credentials{}, errors.New("instance URL is required")
	}

	data, err := k.keyringGet(serviceName, instanceURL)
	if err != nil {
		return Credentials{}, ErrCredentialsNotFound
	}

	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return Credentials{}, fmt.Errorf("unmarshaling credentials: %w", err)
	}

	return creds, nil
}

// Delete removes credentials from the keychain for the given instance URL.
func (k *KeychainStore) Delete(instanceURL string) error {
	if instanceURL == "" {
		return errors.New("instance URL is required")
	}

	if err := k.keyringDelete(serviceName, instanceURL); err != nil {
		return fmt.Errorf("deleting from keychain: %w", err)
	}

	return nil
}

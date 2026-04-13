package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	// serviceName is the shared keychain service identifier for all Atlassian CLI credentials.
	// Used by both confluence-mgmt and jira-mgmt.
	serviceName = "atlassian-mgmt"

	EnvInstanceURL = "CONFLUENCE_MGMT_INSTANCE_URL"
	EnvEmail       = "CONFLUENCE_MGMT_EMAIL"
	EnvAPIToken    = "CONFLUENCE_MGMT_API_TOKEN"
	EnvAuthType    = "CONFLUENCE_MGMT_AUTH_TYPE"
)

// Source identifies the credential backend.
type Source string

const (
	SourceAuto      Source = "auto"
	SourceKeychain  Source = "keychain"
	SourceEnvOrFile Source = "env_or_file"
)

// Credentials holds the authentication details for a Confluence instance.
type Credentials struct {
	InstanceURL string `json:"instance_url"`
	Email       string `json:"email,omitempty"` // Required for Basic auth (Cloud), empty for Bearer (Server/DC PAT)
	APIToken    string `json:"api_token"`
	AuthType    string `json:"auth_type,omitempty"` // "basic" or "bearer"
}

// Validate checks that all required credential fields are populated.
func (c Credentials) Validate() error {
	normalized := normalizeCredentials(c)
	if normalized.InstanceURL == "" {
		return errors.New("instance URL is required")
	}
	if normalized.APIToken == "" {
		return errors.New("API token is required")
	}
	if normalized.AuthType != "bearer" && normalized.Email == "" {
		return errors.New("email is required for basic auth (use --token without --email for PAT/bearer auth)")
	}
	return nil
}

// CredentialStore defines the interface for persisting and retrieving credentials.
type CredentialStore interface {
	Save(creds Credentials) error
	Load(instanceURL string) (Credentials, error)
	Delete(instanceURL string) error
}

// ErrCredentialsNotFound is returned when no credentials exist for a given instance URL.
var ErrCredentialsNotFound = errors.New("credentials not found")

// ErrInstanceURLRequired is returned when a credential operation needs an instance URL key.
var ErrInstanceURLRequired = errors.New("instance URL is required")

var errKeychainUnavailable = errors.New("keychain store is not configured")

// KeychainStore implements CredentialStore using the OS keychain via go-keyring.
type KeychainStore struct {
	keyringSet    func(service, user, password string) error
	keyringGet    func(service, user string) (string, error)
	keyringDelete func(service, user string) error
}

// NewKeychainStore creates a KeychainStore with the provided keyring functions.
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
	creds = normalizeCredentials(creds)
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
	instanceURL = normalizeInstanceURL(instanceURL)
	if instanceURL == "" {
		return Credentials{}, ErrInstanceURLRequired
	}

	data, err := k.keyringGet(serviceName, instanceURL)
	if err != nil {
		return Credentials{}, ErrCredentialsNotFound
	}

	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return Credentials{}, fmt.Errorf("unmarshaling credentials: %w", err)
	}

	return normalizeCredentials(creds), nil
}

// Delete removes credentials from the keychain for the given instance URL.
func (k *KeychainStore) Delete(instanceURL string) error {
	instanceURL = normalizeInstanceURL(instanceURL)
	if instanceURL == "" {
		return ErrInstanceURLRequired
	}

	if err := k.keyringDelete(serviceName, instanceURL); err != nil {
		return fmt.Errorf("deleting from keychain: %w", err)
	}

	return nil
}

type authFile struct {
	Profiles map[string]Credentials `json:"profiles,omitempty"`
}

// FileStore persists credentials under os.UserConfigDir()/confluence-mgmt/auth.json.
type FileStore struct {
	path string
}

// NewFileStore creates a file-backed credential store at the provided path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Save stores credentials under a profile keyed by normalized instance URL.
func (f *FileStore) Save(creds Credentials) error {
	creds = normalizeCredentials(creds)
	if err := creds.Validate(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	cfg, err := f.read()
	if err != nil && !errors.Is(err, ErrCredentialsNotFound) {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Credentials{}
	}
	cfg.Profiles[creds.InstanceURL] = creds
	return f.write(cfg)
}

// Load reads credentials for the given instance URL.
func (f *FileStore) Load(instanceURL string) (Credentials, error) {
	cfg, err := f.read()
	if err != nil {
		return Credentials{}, err
	}

	key, err := selectProfileKey(cfg.Profiles, instanceURL)
	if err != nil {
		return Credentials{}, err
	}

	creds, ok := cfg.Profiles[key]
	if !ok {
		return Credentials{}, ErrCredentialsNotFound
	}
	return normalizeCredentials(creds), nil
}

// Delete removes credentials for the given instance URL.
func (f *FileStore) Delete(instanceURL string) error {
	cfg, err := f.read()
	if err != nil {
		return err
	}

	key, err := selectProfileKey(cfg.Profiles, instanceURL)
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[key]; !ok {
		return ErrCredentialsNotFound
	}
	delete(cfg.Profiles, key)

	if len(cfg.Profiles) == 0 {
		if err := os.Remove(f.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove auth file: %w", err)
		}
		return nil
	}

	return f.write(cfg)
}

func (f *FileStore) read() (authFile, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return authFile{}, ErrCredentialsNotFound
		}
		return authFile{}, fmt.Errorf("reading auth file: %w", err)
	}

	var cfg authFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return authFile{}, fmt.Errorf("parsing auth file: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Credentials{}
	}
	return cfg, nil
}

func (f *FileStore) write(cfg authFile) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("creating auth directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding auth file: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(f.path, data, 0o600); err != nil {
		return fmt.Errorf("writing auth file: %w", err)
	}
	return nil
}

// Runtime allows auth resolution logic to be tested without touching process globals.
type Runtime struct {
	GOOS   string
	Getenv func(string) string
}

type ResolvedCredentials struct {
	Credentials     Credentials
	Source          Source
	ResolvedFrom    string
	ConfigPath      string
	ProfileKey      string
	KeychainService string
	KeychainAccount string
}

type SetAccessResult struct {
	Credentials     Credentials
	Source          Source
	StoredIn        string
	ConfigPath      string
	ProfileKey      string
	KeychainService string
	KeychainAccount string
}

type ClearAccessResult struct {
	Source          Source
	ConfigPath      string
	ProfileKey      string
	KeychainService string
	KeychainAccount string
	Removed         bool
	RemovedFrom     []Source
}

// Resolver implements cross-platform credential lookup policy.
type Resolver struct {
	runtime      Runtime
	keychain     CredentialStore
	authFilePath string
}

// NewResolver creates a credential resolver using the default auth file path.
func NewResolver(rt Runtime, keychain CredentialStore) *Resolver {
	return NewResolverWithAuthFilePath(rt, keychain, "")
}

// NewResolverWithAuthFilePath creates a credential resolver with an explicit auth file path.
func NewResolverWithAuthFilePath(rt Runtime, keychain CredentialStore, authFilePath string) *Resolver {
	if rt.GOOS == "" {
		rt.GOOS = runtime.GOOS
	}
	if rt.Getenv == nil {
		rt.Getenv = os.Getenv
	}
	return &Resolver{
		runtime:      rt,
		keychain:     keychain,
		authFilePath: authFilePath,
	}
}

// DefaultSourceForGOOS returns the preferred desktop credential store for the platform.
func DefaultSourceForGOOS(goos string) Source {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin", "windows":
		return SourceKeychain
	default:
		return SourceEnvOrFile
	}
}

// AuthConfigPath returns the path to the global auth.json file.
func (r *Resolver) AuthConfigPath() (string, error) {
	if r.authFilePath != "" {
		return r.authFilePath, nil
	}
	return AuthConfigPath()
}

// ResolveInstanceURL resolves the active instance URL, allowing an env override.
func (r *Resolver) ResolveInstanceURL(configured string) string {
	if envURL := normalizeInstanceURL(r.runtime.Getenv(EnvInstanceURL)); envURL != "" {
		return envURL
	}
	return normalizeInstanceURL(configured)
}

// SetAccess writes credentials to the selected backend. auto picks the platform default.
func (r *Resolver) SetAccess(source Source, creds Credentials) (SetAccessResult, error) {
	creds = normalizeCredentials(creds)
	if err := creds.Validate(); err != nil {
		return SetAccessResult{}, err
	}

	source = normalizeSource(source)
	if source == SourceAuto {
		source = DefaultSourceForGOOS(r.runtime.GOOS)
	}

	switch source {
	case SourceKeychain:
		if r.keychain == nil {
			return SetAccessResult{}, errKeychainUnavailable
		}
		if err := r.keychain.Save(creds); err != nil {
			return SetAccessResult{}, err
		}
		return SetAccessResult{
			Credentials:     creds,
			Source:          SourceKeychain,
			StoredIn:        "keychain",
			KeychainService: serviceName,
			KeychainAccount: creds.InstanceURL,
		}, nil

	case SourceEnvOrFile:
		path, err := r.AuthConfigPath()
		if err != nil {
			return SetAccessResult{}, err
		}
		store := NewFileStore(path)
		if err := store.Save(creds); err != nil {
			return SetAccessResult{}, err
		}
		return SetAccessResult{
			Credentials: creds,
			Source:      SourceEnvOrFile,
			StoredIn:    "file",
			ConfigPath:  path,
			ProfileKey:  creds.InstanceURL,
		}, nil

	default:
		return SetAccessResult{}, fmt.Errorf("unsupported credential source %q", source)
	}
}

// Resolve finds credentials from the selected backend. auto prefers the platform default
// and then falls back to the alternate desktop-friendly source.
func (r *Resolver) Resolve(source Source, instanceURL string) (ResolvedCredentials, error) {
	instanceURL = r.ResolveInstanceURL(instanceURL)
	source = normalizeSource(source)

	if source != SourceAuto {
		return r.resolveFromSource(source, instanceURL)
	}

	var lastErr error
	for _, candidate := range preferredResolveOrder(r.runtime.GOOS) {
		resolved, err := r.resolveFromSource(candidate, instanceURL)
		if err == nil {
			return resolved, nil
		}
		if isSoftResolveError(err) {
			lastErr = err
			continue
		}
		return ResolvedCredentials{}, err
	}

	if lastErr != nil {
		return ResolvedCredentials{}, lastErr
	}
	return ResolvedCredentials{}, ErrCredentialsNotFound
}

func (r *Resolver) resolveFromSource(source Source, instanceURL string) (ResolvedCredentials, error) {
	switch source {
	case SourceKeychain:
		if r.keychain == nil {
			return ResolvedCredentials{}, errKeychainUnavailable
		}
		if instanceURL == "" {
			return ResolvedCredentials{}, ErrInstanceURLRequired
		}
		creds, err := r.keychain.Load(instanceURL)
		if err != nil {
			return ResolvedCredentials{}, err
		}
		return ResolvedCredentials{
			Credentials:     creds,
			Source:          SourceKeychain,
			ResolvedFrom:    "keychain",
			KeychainService: serviceName,
			KeychainAccount: creds.InstanceURL,
		}, nil

	case SourceEnvOrFile:
		path, err := r.AuthConfigPath()
		if err != nil {
			return ResolvedCredentials{}, err
		}
		if envCreds := credentialsFromEnv(r.runtime.Getenv, instanceURL); envCreds.APIToken != "" {
			if err := envCreds.Validate(); err != nil {
				return ResolvedCredentials{}, err
			}
			return ResolvedCredentials{
				Credentials:  envCreds,
				Source:       SourceEnvOrFile,
				ResolvedFrom: "env",
				ConfigPath:   path,
				ProfileKey:   envCreds.InstanceURL,
			}, nil
		}

		store := NewFileStore(path)
		creds, err := store.Load(instanceURL)
		if err != nil {
			return ResolvedCredentials{}, err
		}
		return ResolvedCredentials{
			Credentials:  creds,
			Source:       SourceEnvOrFile,
			ResolvedFrom: "file",
			ConfigPath:   path,
			ProfileKey:   creds.InstanceURL,
		}, nil

	default:
		return ResolvedCredentials{}, fmt.Errorf("unsupported credential source %q", source)
	}
}

// Clear removes stored credentials. auto removes from both keychain and auth.json.
func (r *Resolver) Clear(source Source, instanceURL string) (ClearAccessResult, error) {
	instanceURL = r.ResolveInstanceURL(instanceURL)
	source = normalizeSource(source)

	if source == SourceAuto {
		return r.clearAuto(instanceURL)
	}
	return r.clearFromSource(source, instanceURL)
}

func (r *Resolver) clearAuto(instanceURL string) (ClearAccessResult, error) {
	result := ClearAccessResult{Source: SourceAuto}
	if path, err := r.AuthConfigPath(); err == nil {
		result.ConfigPath = path
	}

	keychainResult, keychainErr := r.clearFromSource(SourceKeychain, instanceURL)
	if keychainErr == nil && keychainResult.Removed {
		result.Removed = true
		result.RemovedFrom = append(result.RemovedFrom, SourceKeychain)
		result.KeychainService = keychainResult.KeychainService
		result.KeychainAccount = keychainResult.KeychainAccount
	}

	fileResult, fileErr := r.clearFromSource(SourceEnvOrFile, instanceURL)
	if fileErr == nil && fileResult.Removed {
		result.Removed = true
		result.RemovedFrom = append(result.RemovedFrom, SourceEnvOrFile)
		result.ProfileKey = fileResult.ProfileKey
		result.ConfigPath = fileResult.ConfigPath
	}

	if result.Removed {
		sort.Slice(result.RemovedFrom, func(i, j int) bool {
			return result.RemovedFrom[i] < result.RemovedFrom[j]
		})
		return result, nil
	}

	if keychainErr != nil && !isSoftResolveError(keychainErr) {
		return ClearAccessResult{}, keychainErr
	}
	if fileErr != nil && !errors.Is(fileErr, ErrCredentialsNotFound) && !errors.Is(fileErr, ErrInstanceURLRequired) {
		return ClearAccessResult{}, fileErr
	}
	return result, nil
}

func (r *Resolver) clearFromSource(source Source, instanceURL string) (ClearAccessResult, error) {
	switch source {
	case SourceKeychain:
		if r.keychain == nil {
			return ClearAccessResult{}, errKeychainUnavailable
		}
		if instanceURL == "" {
			return ClearAccessResult{}, ErrInstanceURLRequired
		}
		err := r.keychain.Delete(instanceURL)
		if err != nil {
			if errors.Is(err, ErrCredentialsNotFound) || isProbablyMissingSecret(err) {
				return ClearAccessResult{
					Source:          SourceKeychain,
					KeychainService: serviceName,
					KeychainAccount: instanceURL,
					Removed:         false,
				}, nil
			}
			return ClearAccessResult{}, err
		}
		return ClearAccessResult{
			Source:          SourceKeychain,
			KeychainService: serviceName,
			KeychainAccount: instanceURL,
			Removed:         true,
			RemovedFrom:     []Source{SourceKeychain},
		}, nil

	case SourceEnvOrFile:
		path, err := r.AuthConfigPath()
		if err != nil {
			return ClearAccessResult{}, err
		}
		store := NewFileStore(path)
		err = store.Delete(instanceURL)
		if err != nil {
			if errors.Is(err, ErrCredentialsNotFound) {
				return ClearAccessResult{
					Source:     SourceEnvOrFile,
					ConfigPath: path,
					ProfileKey: normalizeInstanceURL(instanceURL),
					Removed:    false,
				}, nil
			}
			return ClearAccessResult{}, err
		}
		return ClearAccessResult{
			Source:      SourceEnvOrFile,
			ConfigPath:  path,
			ProfileKey:  normalizeInstanceURL(instanceURL),
			Removed:     true,
			RemovedFrom: []Source{SourceEnvOrFile},
		}, nil

	default:
		return ClearAccessResult{}, fmt.Errorf("unsupported credential source %q", source)
	}
}

func preferredResolveOrder(goos string) []Source {
	if DefaultSourceForGOOS(goos) == SourceKeychain {
		return []Source{SourceKeychain, SourceEnvOrFile}
	}
	return []Source{SourceEnvOrFile, SourceKeychain}
}

func normalizeSource(source Source) Source {
	switch strings.ToLower(strings.TrimSpace(string(source))) {
	case "", string(SourceAuto):
		return SourceAuto
	case string(SourceKeychain):
		return SourceKeychain
	case string(SourceEnvOrFile):
		return SourceEnvOrFile
	default:
		return source
	}
}

func normalizeCredentials(creds Credentials) Credentials {
	creds.InstanceURL = normalizeInstanceURL(creds.InstanceURL)
	creds.Email = strings.TrimSpace(creds.Email)
	creds.APIToken = strings.TrimSpace(creds.APIToken)
	creds.AuthType = normalizeAuthType(strings.TrimSpace(creds.AuthType), creds.Email)
	return creds
}

func normalizeInstanceURL(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), "/")
}

func normalizeAuthType(authType, email string) string {
	if authType != "" {
		return authType
	}
	if strings.TrimSpace(email) == "" {
		return "bearer"
	}
	return "basic"
}

func credentialsFromEnv(getenv func(string) string, fallbackInstanceURL string) Credentials {
	return normalizeCredentials(Credentials{
		InstanceURL: firstNonEmpty(getenv(EnvInstanceURL), fallbackInstanceURL),
		Email:       getenv(EnvEmail),
		APIToken:    getenv(EnvAPIToken),
		AuthType:    getenv(EnvAuthType),
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func selectProfileKey(profiles map[string]Credentials, instanceURL string) (string, error) {
	if len(profiles) == 0 {
		return "", ErrCredentialsNotFound
	}

	if key := normalizeInstanceURL(instanceURL); key != "" {
		if _, ok := profiles[key]; ok {
			return key, nil
		}
		return "", ErrCredentialsNotFound
	}

	if len(profiles) == 1 {
		for key := range profiles {
			return key, nil
		}
	}

	return "", ErrInstanceURLRequired
}

func isProbablyMissingSecret(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not be found") ||
		strings.Contains(msg, "element not found")
}

func isSoftResolveError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCredentialsNotFound) ||
		errors.Is(err, errKeychainUnavailable) ||
		errors.Is(err, ErrInstanceURLRequired) ||
		strings.Contains(strings.ToLower(err.Error()), "unavailable")
}

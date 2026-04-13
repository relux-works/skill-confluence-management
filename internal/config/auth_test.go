package config

import (
	"errors"
	"path/filepath"
	"testing"
)

type mockKeyring struct {
	store map[string]map[string]string
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{store: make(map[string]map[string]string)}
}

func (m *mockKeyring) set(service, user, password string) error {
	if m.store[service] == nil {
		m.store[service] = make(map[string]string)
	}
	m.store[service][user] = password
	return nil
}

func (m *mockKeyring) get(service, user string) (string, error) {
	if users, ok := m.store[service]; ok {
		if pw, ok := users[user]; ok {
			return pw, nil
		}
	}
	return "", errors.New("not found")
}

func (m *mockKeyring) delete(service, user string) error {
	if users, ok := m.store[service]; ok {
		if _, ok := users[user]; ok {
			delete(users, user)
			return nil
		}
	}
	return errors.New("not found")
}

func newTestStore() (*KeychainStore, *mockKeyring) {
	mk := newMockKeyring()
	store := NewKeychainStore(mk.set, mk.get, mk.delete)
	return store, mk
}

func validCreds() Credentials {
	return Credentials{
		InstanceURL: "https://test.atlassian.net/wiki",
		Email:       "user@test.com",
		APIToken:    "test-token-123",
	}
}

func TestCredentials_Validate(t *testing.T) {
	tests := []struct {
		name    string
		creds   Credentials
		wantErr bool
	}{
		{"valid", validCreds(), false},
		{"missing URL", Credentials{Email: "a@b.com", APIToken: "tok"}, true},
		{"missing email for basic auth", Credentials{InstanceURL: "https://x.atlassian.net/wiki", APIToken: "tok", AuthType: "basic"}, true},
		{"missing token", Credentials{InstanceURL: "https://x.atlassian.net/wiki", Email: "a@b.com"}, true},
		{"bearer no email ok", Credentials{InstanceURL: "https://conf.co", APIToken: "pat", AuthType: "bearer"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeychainStore_SaveAndLoad(t *testing.T) {
	store, _ := newTestStore()
	creds := validCreds()

	if err := store.Save(creds); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(creds.InstanceURL)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Email != creds.Email || loaded.APIToken != creds.APIToken {
		t.Errorf("loaded = %+v, want %+v", loaded, creds)
	}
}

func TestKeychainStore_LoadNotFound(t *testing.T) {
	store, _ := newTestStore()
	_, err := store.Load("https://nonexistent.atlassian.net/wiki")
	if !errors.Is(err, ErrCredentialsNotFound) {
		t.Errorf("error = %v, want ErrCredentialsNotFound", err)
	}
}

func TestKeychainStore_Delete(t *testing.T) {
	store, _ := newTestStore()
	creds := validCreds()
	_ = store.Save(creds)

	if err := store.Delete(creds.InstanceURL); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := store.Load(creds.InstanceURL)
	if !errors.Is(err, ErrCredentialsNotFound) {
		t.Errorf("after delete: error = %v, want ErrCredentialsNotFound", err)
	}
}

func TestKeychainStore_UsesAtlassianMgmtServiceName(t *testing.T) {
	_, mk := newTestStore()
	store := NewKeychainStore(mk.set, mk.get, mk.delete)
	creds := validCreds()

	_ = store.Save(creds)

	if _, ok := mk.store["atlassian-mgmt"][creds.InstanceURL]; !ok {
		t.Error("credentials should be stored under 'atlassian-mgmt' service")
	}
}

func TestDefaultSourceForGOOS(t *testing.T) {
	tests := []struct {
		goos string
		want Source
	}{
		{goos: "darwin", want: SourceKeychain},
		{goos: "windows", want: SourceKeychain},
		{goos: "linux", want: SourceEnvOrFile},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			if got := DefaultSourceForGOOS(tt.goos); got != tt.want {
				t.Fatalf("DefaultSourceForGOOS(%q) = %q, want %q", tt.goos, got, tt.want)
			}
		})
	}
}

func TestFileStore_SaveLoadDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	store := NewFileStore(path)
	creds := validCreds()

	if err := store.Save(creds); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(creds.InstanceURL)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.InstanceURL != creds.InstanceURL {
		t.Fatalf("InstanceURL = %q, want %q", loaded.InstanceURL, creds.InstanceURL)
	}
	if loaded.Email != creds.Email {
		t.Fatalf("Email = %q, want %q", loaded.Email, creds.Email)
	}
	if loaded.APIToken != creds.APIToken {
		t.Fatalf("APIToken = %q, want %q", loaded.APIToken, creds.APIToken)
	}

	if err := store.Delete(creds.InstanceURL); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Load(creds.InstanceURL); !errors.Is(err, ErrCredentialsNotFound) {
		t.Fatalf("Load() after Delete() error = %v, want ErrCredentialsNotFound", err)
	}
}

func TestResolverResolveAutoFallsBackToFileOnWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	store := NewFileStore(path)
	creds := validCreds()
	if err := store.Save(creds); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	resolver := NewResolverWithAuthFilePath(Runtime{
		GOOS:   "windows",
		Getenv: func(string) string { return "" },
	}, nil, path)

	resolved, err := resolver.Resolve(SourceAuto, creds.InstanceURL)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source != SourceEnvOrFile {
		t.Fatalf("Source = %q, want %q", resolved.Source, SourceEnvOrFile)
	}
	if resolved.ResolvedFrom != "file" {
		t.Fatalf("ResolvedFrom = %q, want file", resolved.ResolvedFrom)
	}
}

func TestResolverResolveAutoFallsBackToEnvOnDarwin(t *testing.T) {
	creds := validCreds()
	env := map[string]string{
		EnvInstanceURL: creds.InstanceURL,
		EnvEmail:       creds.Email,
		EnvAPIToken:    creds.APIToken,
	}

	resolver := NewResolverWithAuthFilePath(Runtime{
		GOOS: "darwin",
		Getenv: func(key string) string {
			return env[key]
		},
	}, nil, filepath.Join(t.TempDir(), "auth.json"))

	resolved, err := resolver.Resolve(SourceAuto, "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source != SourceEnvOrFile {
		t.Fatalf("Source = %q, want %q", resolved.Source, SourceEnvOrFile)
	}
	if resolved.ResolvedFrom != "env" {
		t.Fatalf("ResolvedFrom = %q, want env", resolved.ResolvedFrom)
	}
}

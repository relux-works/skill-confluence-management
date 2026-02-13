package config

import (
	"errors"
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
		{"missing email", Credentials{InstanceURL: "https://x.atlassian.net/wiki", APIToken: "tok"}, true},
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
	_, err := store.Load("https://nonexistent.atlassian.net")
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

	// Verify it's stored under "atlassian-mgmt"
	if _, ok := mk.store["atlassian-mgmt"][creds.InstanceURL]; !ok {
		t.Error("credentials should be stored under 'atlassian-mgmt' service")
	}
}

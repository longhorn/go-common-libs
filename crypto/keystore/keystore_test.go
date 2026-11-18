package keystore

import (
	"context"
	"testing"

	lhtypes "github.com/longhorn/go-common-libs/types"
)

func TestDecideMatrix(t *testing.T) {
	cases := []struct {
		name    string
		state   DeviceState
		action  Action
		refused bool
	}{
		{"blank device with a stored key: format", DeviceState{HasKey: true}, ActionFormat, false},
		{"LUKS with its key: open", DeviceState{IsLUKS: true, HasKey: true}, ActionOpen, false},
		{"LUKS with its key, initialized: open", DeviceState{IsLUKS: true, HasKey: true, Initialized: true}, ActionOpen, false},
		{"LUKS without a key: refuse", DeviceState{IsLUKS: true}, "", true},
		{"blank with a key after initialization: refuse", DeviceState{HasKey: true, Initialized: true}, "", true},
		{"blank without a key: refuse", DeviceState{}, "", true},
		{"blank without a key, initialized: refuse", DeviceState{Initialized: true}, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			action, err := Decide(c.state)
			if c.refused {
				if err == nil {
					t.Fatalf("expected refusal, got action %v", action)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if action != c.action {
				t.Fatalf("expected %v, got %v", c.action, action)
			}
		})
	}
}

// vaultFields is the vault half of an encryption Secret.
func vaultFields(overrides map[string]string) map[string]string {
	m := map[string]string{
		lhtypes.CryptoKeyProvider: string(lhtypes.EncryptionKeyProviderVault),
		VaultAddr:                 "https://vault.example:8200",
		VaultAuthPath:             "kubernetes-e2e/",
		VaultRole:                 "longhorn-csi",
	}
	for k, v := range overrides {
		if v == "" {
			delete(m, k)
			continue
		}
		m[k] = v
	}
	return m
}

func TestParseVaultConfigDefaults(t *testing.T) {
	cfg, err := parseVaultConfig(vaultFields(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthPath != "auth/kubernetes-e2e" {
		t.Errorf("auth path should be normalised to auth/kubernetes-e2e, got %v", cfg.AuthPath)
	}
	if cfg.KVMount != defaultKVMount || cfg.KVPrefix != defaultKVPrefix {
		t.Errorf("unexpected KV defaults: mount %v prefix %v", cfg.KVMount, cfg.KVPrefix)
	}
	if !cfg.DestroyKeys {
		t.Errorf("destroy keys should default to true")
	}
	if got := cfg.dataPath(VolumeKey("pvc-1")); got != "v1/secret/data/longhorn/volumes/pvc-1" {
		t.Errorf("unexpected volume data path %v", got)
	}
	if got := cfg.dataPath(BackupKey("backup-1")); got != "v1/secret/data/longhorn/backups/backup-1" {
		t.Errorf("unexpected backup data path %v", got)
	}
	if got := cfg.metadataPath(VolumeKey("pvc-1")); got != "v1/secret/metadata/longhorn/volumes/pvc-1" {
		t.Errorf("unexpected metadata path %v", got)
	}

	cfg, err = parseVaultConfig(vaultFields(map[string]string{
		VaultKVMount: "kv", VaultKVPrefix: "clusters/a", VaultDestroyKeys: "false"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.dataPath(VolumeKey("pvc-1")); got != "v1/kv/data/clusters/a/volumes/pvc-1" {
		t.Errorf("unexpected data path %v", got)
	}
	if cfg.DestroyKeys {
		t.Errorf("destroy keys should honour false")
	}

	cfg, err = parseVaultConfig(vaultFields(map[string]string{VaultKVPrefix: "clusters/a/"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.dataPath(VolumeKey("pvc-1")); got != "v1/secret/data/clusters/a/volumes/pvc-1" {
		t.Errorf("a prefix ending in a slash should be kept as is, got %v", got)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		secrets map[string]string
		wantErr bool
	}{
		{"a secret volume needs its passphrase", map[string]string{lhtypes.CryptoKeyValue: "p"}, false},
		{"an empty provider is the secret provider", map[string]string{}, true},
		{"a vault secret with the required fields", vaultFields(nil), false},
		{"no address", vaultFields(map[string]string{VaultAddr: ""}), true},
		{"address not a URL", vaultFields(map[string]string{VaultAddr: "vault.example"}), true},
		{"no auth path", vaultFields(map[string]string{VaultAuthPath: ""}), true},
		{"no role", vaultFields(map[string]string{VaultRole: ""}), true},
		{"bad CA", vaultFields(map[string]string{VaultCACert: "not-a-pem"}), true},
		{"a passphrase beside the vault provider", vaultFields(map[string]string{lhtypes.CryptoKeyValue: "p"}), true},
		{"unknown provider", map[string]string{lhtypes.CryptoKeyProvider: "kms"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.secrets)
			if c.wantErr && err == nil {
				t.Fatalf("expected %v to be rejected", c.name)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSecretProvider(t *testing.T) {
	p, err := NewProvider(map[string]string{lhtypes.CryptoKeyValue: "passphrase"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != lhtypes.EncryptionKeyProviderSecret {
		t.Fatalf("expected the secret provider, got %v", p.Name())
	}
	got, version, err := p.Get(context.Background(), "any")
	if err != nil || string(got) != "passphrase" || version != 1 {
		t.Fatalf("unexpected Get result %q %v %v", got, version, err)
	}
	if _, err := p.Put(context.Background(), "any", []byte("x"), 0); err != nil {
		t.Fatalf("Put should be a no-op: %v", err)
	}
	if err := p.Delete(context.Background(), "any"); err != nil {
		t.Fatalf("Delete should be a no-op: %v", err)
	}
}

func TestGeneratePassphrase(t *testing.T) {
	a, err := GeneratePassphrase()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GeneratePassphrase()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 44 || string(a) == string(b) {
		t.Fatalf("passphrases should be 44 base64 characters and random: %q %q", a, b)
	}
	Zero(a)
	for _, c := range a {
		if c != 0 {
			t.Fatal("Zero left data behind")
		}
	}
}

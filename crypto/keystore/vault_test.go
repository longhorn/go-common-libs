package keystore

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	lhtypes "github.com/longhorn/go-common-libs/types"
)

// fakeVault is the subset of the Vault HTTP API the provider uses: a login
// mount and a KV v2 mount with versions and check-and-set.
type fakeVault struct {
	mu        sync.Mutex
	authPath  string
	role      string
	logins    int
	tokens    map[string]bool
	entries   map[string]*fakeEntry
	requests  []string
	failLogin bool
	down      bool
}

type fakeEntry struct {
	version   int
	data      map[string]string
	deleted   bool
	destroyed bool
	// history is every version ever written, keyed by version number, as KV v2 keeps them.
	history map[int]map[string]string
}

func newFakeVault() *fakeVault {
	return &fakeVault{authPath: "auth/kubernetes-e2e", role: "longhorn-csi", tokens: map[string]bool{}, entries: map[string]*fakeEntry{}}
}

func (f *fakeVault) write(w http.ResponseWriter, code int, body interface{}) {
	w.WriteHeader(code)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func (f *fakeVault) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	if f.down {
		f.write(w, http.StatusServiceUnavailable, map[string]interface{}{"errors": []string{"sealed"}})
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/v1/")

	if p == f.authPath+"/login" {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if f.failLogin || body["role"] != f.role || body["jwt"] != "sa-jwt" {
			f.write(w, http.StatusForbidden, map[string]interface{}{"errors": []string{"permission denied"}})
			return
		}
		f.logins++
		token := "tok" + string(rune('0'+f.logins))
		f.tokens[token] = true
		f.write(w, http.StatusOK, map[string]interface{}{"auth": map[string]interface{}{"client_token": token, "lease_duration": 3600}})
		return
	}

	if !f.tokens[r.Header.Get("X-Vault-Token")] {
		f.write(w, http.StatusForbidden, map[string]interface{}{"errors": []string{"permission denied"}})
		return
	}

	switch {
	case strings.HasPrefix(p, "secret/data/"):
		key := strings.TrimPrefix(p, "secret/data/")
		e := f.entries[key]
		switch r.Method {
		case http.MethodGet:
			if e == nil {
				f.write(w, http.StatusNotFound, map[string]interface{}{"errors": []string{}})
				return
			}
			version := e.version
			if q := r.URL.Query().Get("version"); q != "" {
				asked, err := strconv.Atoi(q)
				if err != nil || e.history[asked] == nil {
					f.write(w, http.StatusNotFound, map[string]interface{}{"errors": []string{}})
					return
				}
				version = asked
			}
			meta := map[string]interface{}{"version": version, "destroyed": e.destroyed, "deletion_time": ""}
			data := interface{}(e.history[version])
			if e.deleted && version == e.version {
				meta["deletion_time"] = "2026-01-01T00:00:00Z"
				data = nil
			}
			f.write(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{"data": data, "metadata": meta}})
		case http.MethodPost, http.MethodPut:
			var body struct {
				Options map[string]int    `json:"options"`
				Data    map[string]string `json:"data"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			current := 0
			if e != nil {
				current = e.version
			}
			if cas, ok := body.Options["cas"]; ok && cas != current {
				f.write(w, http.StatusBadRequest, map[string]interface{}{"errors": []string{"check-and-set parameter did not match the current version"}})
				return
			}
			history := map[int]map[string]string{}
			if e != nil {
				history = e.history
			}
			history[current+1] = body.Data
			f.entries[key] = &fakeEntry{version: current + 1, data: body.Data, history: history}
			f.write(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{"version": current + 1}})
		case http.MethodDelete:
			if e != nil {
				e.deleted = true
			}
			f.write(w, http.StatusNoContent, nil)
		}
	case strings.HasPrefix(p, "secret/metadata/"):
		key := strings.TrimPrefix(p, "secret/metadata/")
		if r.Method == http.MethodDelete {
			delete(f.entries, key)
			f.write(w, http.StatusNoContent, nil)
			return
		}
		f.write(w, http.StatusMethodNotAllowed, nil)
	default:
		f.write(w, http.StatusNotFound, map[string]interface{}{"errors": []string{"unsupported path"}})
	}
}

func (f *fakeVault) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.requests {
		if strings.HasPrefix(r, prefix) {
			n++
		}
	}
	return n
}

func resetTokenCache() {
	tokenCacheMu.Lock()
	tokenCache = map[string]*cachedToken{}
	tokenCacheMu.Unlock()
}

// stageToken points the provider at a token file a test can write.
func stageToken(t *testing.T, content string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "token")
	if content != "" {
		if err := os.WriteFile(file, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	previous := tokenFilePath
	tokenFilePath = file
	t.Cleanup(func() { tokenFilePath = previous })
}

// vaultSecret is an encryption Secret of the vault provider, as a caller reads
// it out of Kubernetes.
func vaultSecret(srv *httptest.Server, overrides map[string]string) map[string]string {
	s := map[string]string{
		lhtypes.CryptoKeyProvider: string(lhtypes.EncryptionKeyProviderVault),
		VaultAddr:                 srv.URL,
		VaultAuthPath:             "auth/kubernetes-e2e",
		VaultRole:                 "longhorn-csi",
	}
	for k, v := range overrides {
		if v == "" {
			delete(s, k)
			continue
		}
		s[k] = v
	}
	return s
}

func newTestProvider(t *testing.T, srv *httptest.Server, overrides map[string]string) *vaultProvider {
	t.Helper()
	stageToken(t, "sa-jwt\n")
	p, err := NewProvider(vaultSecret(srv, overrides))
	if err != nil {
		t.Fatal(err)
	}
	return p.(*vaultProvider)
}

func TestVaultLifecycle(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, nil)
	ctx := context.Background()

	if _, _, err := p.Get(ctx, VolumeKey("pvc-1")); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound for a new volume, got %v", err)
	}
	version, err := p.Put(ctx, VolumeKey("pvc-1"), []byte("k1"), 0)
	if err != nil || version != 1 {
		t.Fatalf("create-only Put should give version 1: %v %v", version, err)
	}
	if _, err := p.Put(ctx, VolumeKey("pvc-1"), []byte("k2"), 0); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("second create-only Put should conflict, got %v", err)
	}
	got, version, err := p.Get(ctx, VolumeKey("pvc-1"))
	if err != nil || string(got) != "k1" || version != 1 {
		t.Fatalf("Get should return the first key at version 1: %q %v %v", got, version, err)
	}
	version, err = p.Put(ctx, VolumeKey("pvc-1"), []byte("k2"), 1)
	if err != nil || version != 2 {
		t.Fatalf("Put with the current version should give version 2: %v %v", version, err)
	}
	if err := p.Delete(ctx, VolumeKey("pvc-1")); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if fake.count("DELETE /v1/secret/metadata/longhorn/volumes/pvc-1") != 1 {
		t.Fatalf("destroying keys should delete the metadata, requests: %v", fake.requests)
	}
	if _, _, err := p.Get(ctx, VolumeKey("pvc-1")); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound after delete, got %v", err)
	}
	if err := p.Delete(ctx, VolumeKey("pvc-1")); err != nil {
		t.Fatalf("deleting a missing key should succeed: %v", err)
	}
	if fake.logins != 1 {
		t.Fatalf("one login should serve every call, got %d", fake.logins)
	}
}

func TestVaultSoftDelete(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, map[string]string{VaultDestroyKeys: "false"})
	ctx := context.Background()
	if _, err := p.Put(ctx, VolumeKey("pvc-1"), []byte("k1"), 0); err != nil {
		t.Fatal(err)
	}
	if err := p.Delete(ctx, VolumeKey("pvc-1")); err != nil {
		t.Fatal(err)
	}
	if fake.count("DELETE /v1/secret/data/longhorn/volumes/pvc-1") != 1 || fake.count("DELETE /v1/secret/metadata/") != 0 {
		t.Fatalf("soft delete should hit the data path only, requests: %v", fake.requests)
	}
	if _, _, err := p.Get(ctx, VolumeKey("pvc-1")); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("a soft-deleted key reads as not found, got %v", err)
	}
}

func TestVaultTokenCacheAndRelogin(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()

	p1 := newTestProvider(t, srv, nil)
	if _, err := p1.Put(ctx, "pvc-1", []byte("k1"), 0); err != nil {
		t.Fatal(err)
	}
	// A second provider for the same target (another NodeStage) reuses the token.
	p2 := newTestProvider(t, srv, nil)
	if _, _, err := p2.Get(ctx, "pvc-1"); err != nil {
		t.Fatal(err)
	}
	if fake.logins != 1 {
		t.Fatalf("the token should be shared across providers, logins=%d", fake.logins)
	}

	// Vault forgets the token (revoked or expired): one re-login, then the call succeeds.
	fake.mu.Lock()
	fake.tokens = map[string]bool{}
	fake.mu.Unlock()
	if _, _, err := p2.Get(ctx, "pvc-1"); err != nil {
		t.Fatalf("Get after token revocation should re-login: %v", err)
	}
	if fake.logins != 2 {
		t.Fatalf("expected exactly one re-login, logins=%d", fake.logins)
	}

	// Login refused: the error names the mount and role and the call fails.
	fake.mu.Lock()
	fake.tokens = map[string]bool{}
	fake.failLogin = true
	fake.mu.Unlock()
	_, _, err := p2.Get(ctx, "pvc-1")
	if err == nil || !strings.Contains(err.Error(), "auth/kubernetes-e2e") || !strings.Contains(err.Error(), "longhorn-csi") {
		t.Fatalf("a refused login should fail naming the mount and role, got %v", err)
	}
}

func TestVaultLoginAtConfiguredPath(t *testing.T) {
	for _, authPath := range []string{"auth/kubernetes-prod-a", "kubernetes-prod-b"} {
		t.Run(authPath, func(t *testing.T) {
			resetTokenCache()
			fake := newFakeVault()
			fake.authPath = "auth/" + strings.TrimPrefix(authPath, "auth/")
			srv := httptest.NewServer(fake)
			defer srv.Close()
			p := newTestProvider(t, srv, map[string]string{VaultAuthPath: authPath})
			if _, err := p.Put(context.Background(), "pvc-1", []byte("k"), 0); err != nil {
				t.Fatal(err)
			}
			// An auth path given with or without the auth/ prefix reaches the same mount.
			if fake.count("POST /v1/"+fake.authPath+"/login") != 1 {
				t.Fatalf("expected one login at %v, requests: %v", fake.authPath, fake.requests)
			}
		})
	}
}

func TestVaultUnavailable(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, nil)
	ctx := context.Background()
	if _, err := p.Put(ctx, "pvc-1", []byte("k1"), 0); err != nil {
		t.Fatal(err)
	}

	fake.mu.Lock()
	fake.down = true
	fake.mu.Unlock()
	if _, _, err := p.Get(ctx, "pvc-1"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("a 5xx should be ErrUnavailable, got %v", err)
	}

	fake.mu.Lock()
	fake.down = false
	fake.mu.Unlock()
	if got, _, err := p.Get(ctx, "pvc-1"); err != nil || string(got) != "k1" {
		t.Fatalf("Get after recovery: %q %v", got, err)
	}

	srv.Close()
	if _, _, err := p.Get(ctx, "pvc-1"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("a connection error should be ErrUnavailable, got %v", err)
	}
}

func TestVaultMissingTokenFile(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, nil)
	stageToken(t, "") // the projected volume is not mounted

	_, _, err := p.Get(context.Background(), "pvc-1")
	if err == nil || !strings.Contains(err.Error(), "ServiceAccount token") {
		t.Fatalf("a missing token file should be reported explicitly, got %v", err)
	}
}

// TestVaultKeyNames covers the two namespaces of one profile: volumes keep the
// prefix, backups sit under it, and no volume name can reach a backup entry.
func TestVaultKeyNames(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, nil)
	ctx := context.Background()

	if _, err := p.Put(ctx, VolumeKey("pvc-1"), []byte("volume-key"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Put(ctx, BackupKey("backup-1"), []byte("backup-key"), 0); err != nil {
		t.Fatal(err)
	}
	if fake.count("POST /v1/secret/data/longhorn/volumes/pvc-1") != 1 {
		t.Fatalf("a volume key belongs under volumes/, requests: %v", fake.requests)
	}
	if fake.count("POST /v1/secret/data/longhorn/backups/backup-1") != 1 {
		t.Fatalf("a backup key belongs under backups/, requests: %v", fake.requests)
	}
	// Deleting the volume leaves the backup readable: that is why a backup owns an entry.
	if err := p.Delete(ctx, VolumeKey("pvc-1")); err != nil {
		t.Fatal(err)
	}
	got, _, err := p.Get(ctx, BackupKey("backup-1"))
	if err != nil || string(got) != "backup-key" {
		t.Fatalf("the backup key should survive the volume: %q %v", got, err)
	}
}

// TestVaultGetVersion covers the versioned read a derived volume uses.
func TestVaultGetVersion(t *testing.T) {
	resetTokenCache()
	fake := newFakeVault()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	p := newTestProvider(t, srv, nil)
	ctx := context.Background()

	if _, err := p.Put(ctx, "pvc-1", []byte("k1"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Put(ctx, "pvc-1", []byte("k2"), 1); err != nil {
		t.Fatal(err)
	}
	got, err := p.GetVersion(ctx, "pvc-1", 1)
	if err != nil || string(got) != "k1" {
		t.Fatalf("version 1 should still read the first key: %q %v", got, err)
	}
	if _, err := p.GetVersion(ctx, "pvc-1", 0); err == nil {
		t.Fatal("version 0 is not a stored version")
	}
	if _, err := p.GetVersion(ctx, "pvc-1", 9); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("a version that was never written should be ErrKeyNotFound, got %v", err)
	}
}

func TestVaultRefusesDowngradeRedirect(t *testing.T) {
	req := func(u string) *http.Request {
		r, _ := http.NewRequest(http.MethodGet, u, nil)
		return r
	}
	via := []*http.Request{req("https://vault-0:8200/v1/secret/data/x")}
	if err := refuseDowngrade(req("http://vault-1:8200/v1/secret/data/x"), via); err == nil {
		t.Error("an https to http redirect should be refused")
	}
	if err := refuseDowngrade(req("https://vault-1:8200/v1/secret/data/x"), via); err != nil {
		t.Errorf("an https to https redirect should be followed: %v", err)
	}
}

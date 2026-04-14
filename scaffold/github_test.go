package scaffold

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slouowzee/kapi/internal/testutil"
)

func redirectTo(t *testing.T, ts *httptest.Server) {
	t.Helper()
	orig := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = orig })
	inner := orig
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = ts.Listener.Addr().String()
		return inner.RoundTrip(req2)
	})
}

func setupTempHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestCreateGithubRepo_NoToken(t *testing.T) {
	setupTempHome(t)
	t.Setenv("GITHUB_TOKEN", "")

	_, err := createGithubRepo(context.Background(), "my-repo", false)
	if err == nil {
		t.Error("expected error when no token configured, got nil")
	}
}

func TestCreateGithubRepo_Created_ReturnsSSHURL(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	const wantSSH = "git@github.com:user/my-repo.git"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"ssh_url": wantSSH})
	}))
	defer ts.Close()
	redirectTo(t, ts)

	got, err := createGithubRepo(context.Background(), "my-repo", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantSSH {
		t.Errorf("sshURL = %q, want %q", got, wantSSH)
	}
}

func TestCreateGithubRepo_AlreadyExists(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("{}"))
	}))
	defer ts.Close()
	redirectTo(t, ts)

	_, err := createGithubRepo(context.Background(), "existing-repo", false)
	if err == nil {
		t.Error("expected error for already existing repo, got nil")
	}
	if !strings.Contains(err.Error(), "existing-repo") {
		t.Errorf("error %q should mention the repo name", err.Error())
	}
}

func TestCreateGithubRepo_HTTPError(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer ts.Close()
	redirectTo(t, ts)

	_, err := createGithubRepo(context.Background(), "my-repo", false)
	if err == nil {
		t.Error("expected error for HTTP 403, got nil")
	}
}

func TestCreateGithubRepo_InvalidJSON(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json at all {{{"))
	}))
	defer ts.Close()
	redirectTo(t, ts)

	_, err := createGithubRepo(context.Background(), "my-repo", false)
	if err == nil {
		t.Error("expected error for invalid JSON response, got nil")
	}
}

func TestCreateGithubRepo_MissingSSHURL(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"other_field": "value"})
	}))
	defer ts.Close()
	redirectTo(t, ts)

	_, err := createGithubRepo(context.Background(), "my-repo", false)
	if err == nil {
		t.Error("expected error when ssh_url is missing from response, got nil")
	}
}

func TestCreateGithubRepo_SendsBearerAuth(t *testing.T) {
	const tok = "my-secret-token"
	t.Setenv("GITHUB_TOKEN", tok)

	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"ssh_url": "git@github.com:user/repo.git"})
	}))
	defer ts.Close()
	redirectTo(t, ts)

	if _, err := createGithubRepo(context.Background(), "repo", false); err != nil {
		t.Fatal(err)
	}
	if want := "Bearer " + tok; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
}

func TestCreateGithubRepo_PrivateFlag(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"ssh_url": "git@github.com:user/repo.git"})
	}))
	defer ts.Close()
	redirectTo(t, ts)

	if _, err := createGithubRepo(context.Background(), "repo", true); err != nil {
		t.Fatal(err)
	}
	if gotBody["private"] != true {
		t.Errorf("private field in request body = %v, want true", gotBody["private"])
	}
}

func TestCreateGithubRepo_UsesEnvTokenOverConfigFile(t *testing.T) {
	setupTempHome(t)
	const envTok = "env-token"
	t.Setenv("GITHUB_TOKEN", envTok)

	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"ssh_url": "git@github.com:u/r.git"})
	}))
	defer ts.Close()
	redirectTo(t, ts)

	if _, err := createGithubRepo(context.Background(), "repo", false); err != nil {
		t.Fatal(err)
	}
	if want := "Bearer " + envTok; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
}

func TestSetupTempHome_IsolatesConfigFile(t *testing.T) {
	setupTempHome(t)
	t.Setenv("GITHUB_TOKEN", "")

	_, err := createGithubRepo(context.Background(), "repo", false)
	if err == nil {
		t.Error("should fail with no token in isolated home")
	}
}

package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

// The tests below exercise the client-go paths that link golang.org/x/oauth2.
// k8s.io/client-go/transport is the only importer of oauth2 in this module, and
// it reaches oauth2.Transport when a rest.Config carries a BearerTokenFile.
// The SEC-31 update raised oauth2 from v0.13.0 to v0.27.0, so these tests prove
// the new version still signs requests the same way at run time.

// Test_TransportSendsBearerTokenFromFile checks the oauth2 backed round tripper.
// The token comes from a file, so client-go builds a cached token source and
// wraps the transport with oauth2.Transport.
func Test_TransportSendsBearerTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("secret-token-value"), 0o600))

	var got string
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		got = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := rest.HTTPClientFor(&rest.Config{
		Host:            server.URL,
		BearerTokenFile: tokenPath,
	})
	require.NoError(t, err)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 1, calls, "the server must receive exactly 1 request")
	assert.Equal(t, "Bearer secret-token-value", got,
		"the oauth2 transport must add the bearer token from the file")
}

// Test_TransportUsesTheCachedTokenInsideTheCacheWindow checks the oauth2 token
// cache. client-go reads the token file with a period of 1 minute and a leeway
// of 10 seconds, so 2 requests in the same test share one read.
func Test_TransportUsesTheCachedTokenInsideTheCacheWindow(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("first-token"), 0o600))

	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := rest.HTTPClientFor(&rest.Config{
		Host:            server.URL,
		BearerTokenFile: tokenPath,
	})
	require.NoError(t, err)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	// The file changes, but the cache still holds the first token.
	require.NoError(t, os.WriteFile(tokenPath, []byte("second-token"), 0o600))

	resp, err = client.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, seen, 2)
	assert.Equal(t, "Bearer first-token", seen[0])
	assert.Equal(t, "Bearer first-token", seen[1],
		"the cached token must serve a second request inside the cache window")
}

// Test_TransportReadsTheRotatedTokenFile checks that the transport reads the
// token file again after a rotation. A new client holds an empty cache, so the
// test asserts the rotated value without a wait for the cache window.
func Test_TransportReadsTheRotatedTokenFile(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("first-token"), 0o600))

	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &rest.Config{Host: server.URL, BearerTokenFile: tokenPath}

	first, err := rest.HTTPClientFor(config)
	require.NoError(t, err)
	resp, err := first.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	require.NoError(t, os.WriteFile(tokenPath, []byte("second-token"), 0o600))

	second, err := rest.HTTPClientFor(config)
	require.NoError(t, err)
	resp, err = second.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	require.Len(t, seen, 2)
	assert.Equal(t, "Bearer first-token", seen[0])
	assert.Equal(t, "Bearer second-token", seen[1],
		"the transport must read the rotated token file")
}

// Test_TransportOmitsAuthorizationWithoutAToken checks that the transport adds
// no Authorization header when the rest.Config holds no credential. An oauth2
// change that injected an empty bearer token would break anonymous access.
func Test_TransportOmitsAuthorizationWithoutAToken(t *testing.T) {
	var got string
	var present bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, present = r.Header["Authorization"]
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := rest.HTTPClientFor(&rest.Config{Host: server.URL})
	require.NoError(t, err)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.False(t, present, "the transport must send no Authorization header, got %q", got)
}

// Test_TransportFailsOnAMissingTokenFile checks the error path. The token file
// does not exist, so client-go must fail while it builds the client. A lazy
// failure could let an unauthenticated request reach the API server.
func Test_TransportFailsOnAMissingTokenFile(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := rest.HTTPClientFor(&rest.Config{
		Host:            server.URL,
		BearerTokenFile: filepath.Join(t.TempDir(), "absent"),
	})
	require.Error(t, err, "an absent token file must fail while the client is built")
	assert.Contains(t, err.Error(), "failed to read token file")
	assert.Nil(t, client, "a failed build must return no client")
	assert.Zero(t, calls, "the client must send no request without a token")
}

// Test_TransportFailsOnAnEmptyTokenFile checks that an empty token file fails
// while client-go builds the client. The failure must be early, because the
// round tripper sends the header "Bearer " when the token is empty, and an API
// server can treat that request as anonymous.
func Test_TransportFailsOnAnEmptyTokenFile(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte(""), 0o600))

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := rest.HTTPClientFor(&rest.Config{
		Host:            server.URL,
		BearerTokenFile: tokenPath,
	})
	require.Error(t, err, "an empty token file must fail while the client is built")
	assert.Contains(t, err.Error(), "read empty token from file")
	assert.Nil(t, client, "a failed build must return no client")
	assert.Zero(t, calls, "the client must send no request without a token")
}

// Test_BearerRoundTripperSendsAnEmptyCredential documents the hazard that
// Test_TransportFailsOnAnEmptyTokenFile guards. The round tripper adds the
// header "Bearer " when the token is empty, so an empty token must never reach
// it through a rest.Config.
func Test_BearerRoundTripperSendsAnEmptyCredential(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: transport.NewBearerAuthRoundTripper("", http.DefaultTransport),
	}
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The round tripper sets the value "Bearer " with a trailing space, and
	// net/http trims that space, so the server reads "Bearer" with no credential.
	assert.Equal(t, "Bearer", got,
		"an empty bearer token gives an empty credential, which weakens authentication")
}

// Test_TransportRejectsConflictingCredentials checks that client-go still
// rejects a config that holds both a bearer token and a client certificate.
func Test_TransportRejectsConflictingCredentials(t *testing.T) {
	_, err := rest.HTTPClientFor(&rest.Config{
		Host:        "https://example.invalid",
		BearerToken: "a-token",
		Username:    "a-user",
		Password:    "a-password",
	})
	assert.Error(t, err, "a config with 2 credential kinds must fail")
}

// Test_GetRestConfigFallsBackToKubeconfig checks the admiral fallback path.
// The test runs outside a cluster, so rest.InClusterConfig fails and
// buildOutOfClusterConfig reads the file that KUBECONFIG names.
func Test_GetRestConfigFallsBackToKubeconfig(t *testing.T) {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		t.Skip("the test runs inside a cluster, so the fallback path is unreachable")
	}

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("kubeconfig-token"), 0o600))

	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://kube.example.invalid
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    tokenFile: ` + tokenPath + "\n"

	path := filepath.Join(dir, "config")
	require.NoError(t, os.WriteFile(path, []byte(kubeconfig), 0o600))
	t.Setenv("KUBECONFIG", path)

	config, err := GetRestConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "https://kube.example.invalid", config.Host)
	assert.Equal(t, tokenPath, config.BearerTokenFile,
		"the kubeconfig tokenFile must reach the oauth2 backed transport")

	// The config must build a working client, which links the oauth2 transport.
	client, err := GetClient()
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// Test_GetRestConfigFailsOnAnAbsentKubeconfig checks the error path when neither
// an in-cluster config nor a kubeconfig file exists.
func Test_GetRestConfigFailsOnAnAbsentKubeconfig(t *testing.T) {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		t.Skip("the test runs inside a cluster, so the fallback path is unreachable")
	}
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "absent"))

	_, err := GetRestConfig()
	assert.Error(t, err, "an absent kubeconfig must produce an error")

	_, err = GetClient()
	assert.Error(t, err, "GetClient must report the config error")
}

// Test_GetRestConfigUsesHomeWhenKubeconfigIsEmpty checks the HOME fallback that
// the README documents. An empty KUBECONFIG must send admiral to
// $HOME/.kube/config.
func Test_GetRestConfigUsesHomeWhenKubeconfigIsEmpty(t *testing.T) {
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		t.Skip("the test runs inside a cluster, so the fallback path is unreachable")
	}

	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".kube"), 0o755))
	kubeconfig := `apiVersion: v1
kind: Config
clusters:
- name: home
  cluster:
    server: https://home.example.invalid
contexts:
- name: home
  context:
    cluster: home
    user: home
current-context: home
users:
- name: home
  user:
    token: home-token
`
	require.NoError(t, os.WriteFile(filepath.Join(home, ".kube", "config"), []byte(kubeconfig), 0o600))

	t.Setenv("KUBECONFIG", "")
	t.Setenv("HOME", home)

	config, err := GetRestConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://home.example.invalid", config.Host)
	assert.Equal(t, "home-token", config.BearerToken)
}

// Test_ShrinkStringMap checks the map copy helper, including the empty and nil
// inputs that no existing test covers.
func Test_ShrinkStringMap(t *testing.T) {
	original := map[string]string{"a": "1", "b": "2"}
	copied := ShrinkStringMap(original)
	assert.Equal(t, original, copied)

	// The copy must be independent of the original.
	copied["a"] = "changed"
	assert.Equal(t, "1", original["a"], "the helper must not alias the input map")

	assert.Empty(t, ShrinkStringMap(map[string]string{}))
	assert.NotNil(t, ShrinkStringMap(nil), "a nil input must give an empty map, not nil")
	assert.Len(t, ShrinkStringMap(nil), 0)
}

// Test_GenerateUniqueContainerName checks the name format helper.
func Test_GenerateUniqueContainerName(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1"}}
	container := v1.Container{Name: "app"}
	assert.Equal(t, "ns-1.pod-1.app", GenerateUniqueContainerName(pod, container))

	// Empty fields must still give a stable 3 part name.
	empty := &v1.Pod{}
	assert.Equal(t, "..", GenerateUniqueContainerName(empty, v1.Container{}))
}

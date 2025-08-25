//go:build testing
// +build testing

package hub_test

import (
	"beszel/internal/tests"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestHub(t testing.TB) *tests.TestHub {
	hub, _ := tests.NewTestHub(t.TempDir())
	return hub
}

func TestMakeLink(t *testing.T) {
	hub := getTestHub(t)

	tests := []struct {
		name     string
		appURL   string
		parts    []string
		expected string
	}{
		{
			name:     "no parts, no trailing slash in AppURL",
			appURL:   "http://localhost:8090",
			parts:    []string{},
			expected: "http://localhost:8090",
		},
		{
			name:     "no parts, with trailing slash in AppURL",
			appURL:   "http://localhost:8090/",
			parts:    []string{},
			expected: "http://localhost:8090", // TrimSuffix should handle the trailing slash
		},
		{
			name:     "one part",
			appURL:   "http://example.com",
			parts:    []string{"one"},
			expected: "http://example.com/one",
		},
		{
			name:     "multiple parts",
			appURL:   "http://example.com",
			parts:    []string{"alpha", "beta", "gamma"},
			expected: "http://example.com/alpha/beta/gamma",
		},
		{
			name:     "parts with spaces needing escaping",
			appURL:   "http://example.com",
			parts:    []string{"path with spaces", "another part"},
			expected: "http://example.com/path%20with%20spaces/another%20part",
		},
		{
			name:     "parts with slashes needing escaping",
			appURL:   "http://example.com",
			parts:    []string{"a/b", "c"},
			expected: "http://example.com/a%2Fb/c", // url.PathEscape escapes '/'
		},
		{
			name:     "AppURL with subpath, no trailing slash",
			appURL:   "http://localhost/sub",
			parts:    []string{"resource"},
			expected: "http://localhost/sub/resource",
		},
		{
			name:     "AppURL with subpath, with trailing slash",
			appURL:   "http://localhost/sub/",
			parts:    []string{"item"},
			expected: "http://localhost/sub/item",
		},
		{
			name:     "empty parts in the middle",
			appURL:   "http://localhost",
			parts:    []string{"first", "", "third"},
			expected: "http://localhost/first/third",
		},
		{
			name:     "leading and trailing empty parts",
			appURL:   "http://localhost",
			parts:    []string{"", "path", ""},
			expected: "http://localhost/path",
		},
		{
			name:     "parts with various special characters",
			appURL:   "https://test.dev/",
			parts:    []string{"p@th?", "key=value&"},
			expected: "https://test.dev/p@th%3F/key=value&",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub.App = &mockApp{appURL: tt.appURL}
			got := hub.MakeLink(tt.parts...)
			assert.Equal(t, tt.expected, got, "MakeLink generated URL does not match expected")
		})
	}
}

func TestAuthKeyGeneration(t *testing.T) {
	hub := getTestHub(t)

	// Test Case 1: Key generation (no existing key)
	t.Run("KeyGeneration", func(t *testing.T) {
		tempDir := t.TempDir()

		// Ensure authKey is initially empty
		hub.SetAuthKey("")

		// Set the data directory for the hub
		hub.App = &mockApp{dataDir: tempDir}

		// Generate auth key
		hub.generateAuthKey()

		// Check if auth key was generated
		authKey := hub.GetAuthKey()
		assert.NotEmpty(t, authKey, "Auth key should be generated")
		assert.True(t, strings.HasPrefix(authKey, "base64:"), "Auth key should start with 'base64:'")

		// Check if auth key file was created
		authKeyPath := filepath.Join(tempDir, "auth_key")
		info, err := os.Stat(authKeyPath)
		assert.NoError(t, err, "Auth key file should be created")
		assert.False(t, info.IsDir(), "Auth key path should be a file, not a directory")

		// Verify the file contains the same key
		keyData, err := os.ReadFile(authKeyPath)
		require.NoError(t, err)
		assert.Equal(t, authKey, string(keyData), "File should contain the same auth key")
	})

	// Test Case 2: Existing key
	t.Run("ExistingKey", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a test auth key file
		expectedAuthKey := "base64:dGVzdC1hdXRoLWtleS1mb3ItdGVzdGluZw=="
		authKeyPath := filepath.Join(tempDir, "auth_key")
		err := os.WriteFile(authKeyPath, []byte(expectedAuthKey), 0600)
		require.NoError(t, err, "Failed to write pre-existing auth key")

		// Set the data directory for the hub
		hub.App = &mockApp{dataDir: tempDir}

		// Reset authKey to ensure it's loaded from file
		hub.SetAuthKey("")

		// Load auth key
		hub.generateAuthKey()

		// Check if auth key was loaded correctly
		authKey := hub.GetAuthKey()
		assert.Equal(t, expectedAuthKey, authKey, "Auth key should match the existing key")
	})

	// Test Case 3: Error cases
	t.Run("ErrorCases", func(t *testing.T) {
		t.Run("InvalidDirectory", func(t *testing.T) {
			// Set an invalid data directory
			hub.App = &mockApp{dataDir: "/nonexistent/directory"}

			// Reset authKey
			hub.SetAuthKey("")

			// Generate auth key (should fall back to default)
			hub.generateAuthKey()

			// Should still have a key (fallback)
			authKey := hub.GetAuthKey()
			assert.NotEmpty(t, authKey, "Should have fallback auth key")
			assert.True(t, strings.HasPrefix(authKey, "base64:"), "Fallback auth key should start with 'base64:'")
		})
	})
}

// mockApp implements the minimal App interface for testing
type mockApp struct {
	dataDir string
	appURL  string
}

func (m *mockApp) DataDir() string {
	return m.dataDir
}

func (m *mockApp) Settings() interface{} {
	return &mockSettings{appURL: m.appURL}
}

type mockSettings struct {
	appURL string
}

func (m *mockSettings) Meta() interface{} {
	return &mockMeta{appURL: m.appURL}
}

type mockMeta struct {
	appURL string
}

func (m *mockMeta) AppURL() string {
	return m.appURL
}

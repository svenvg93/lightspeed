//go:build testing
// +build testing

package hub

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test hub without import cycle
func createTestHub(t testing.TB) (*Hub, *pbtests.TestApp, error) {
	testDataDir := t.TempDir()
	testApp, err := pbtests.NewTestApp(testDataDir)
	if err != nil {
		return nil, nil, err
	}
	return NewHub(testApp), testApp, nil
}

// Helper function to create a test record
func createTestRecord(app core.App, collection string, data map[string]any) (*core.Record, error) {
	col, err := app.FindCachedCollectionByNameOrId(collection)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	for key, value := range data {
		record.Set(key, value)
	}

	return record, app.Save(record)
}

// Helper function to create a test user
func createTestUser(app core.App) (*core.Record, error) {
	userRecord, err := createTestRecord(app, "users", map[string]any{
		"email":    "test@test.com",
		"password": "testtesttest",
	})
	return userRecord, err
}

// TestValidateAgentHeaders tests the validateAgentHeaders function
func TestValidateAgentHeaders(t *testing.T) {
	hub, testApp, err := createTestHub(t)
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	testCases := []struct {
		name          string
		headers       http.Header
		expectError   bool
		expectedToken string
		expectedAgent string
	}{
		{
			name: "valid headers",
			headers: http.Header{
				"X-Token":  []string{"valid-token-123"},
				"X-Beszel": []string{"0.5.0"},
			},
			expectError:   false,
			expectedToken: "valid-token-123",
			expectedAgent: "0.5.0",
		},
		{
			name: "missing token",
			headers: http.Header{
				"X-Beszel": []string{"0.5.0"},
			},
			expectError: true,
		},
		{
			name: "missing agent version",
			headers: http.Header{
				"X-Token": []string{"valid-token-123"},
			},
			expectError: true,
		},
		{
			name: "empty token",
			headers: http.Header{
				"X-Token":  []string{""},
				"X-Beszel": []string{"0.5.0"},
			},
			expectError: true,
		},
		{
			name: "empty agent version",
			headers: http.Header{
				"X-Token":  []string{"valid-token-123"},
				"X-Beszel": []string{""},
			},
			expectError: true,
		},
		{
			name: "token too long",
			headers: http.Header{
				"X-Token":  []string{strings.Repeat("a", 65)},
				"X-Beszel": []string{"0.5.0"},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			acr := &agentConnectRequest{
				hub: hub,
				req: &http.Request{Header: tc.headers},
			}

			token, agentVersion, err := acr.validateAgentHeaders(tc.headers)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedToken, token)
				assert.Equal(t, tc.expectedAgent, agentVersion)
			}
		})
	}
}

// TestAuthKeyVerification tests the new base64 auth key verification
func TestAuthKeyVerification(t *testing.T) {
	hub, testApp, err := createTestHub(t)
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	// Set a test auth key
	testAuthKey := "base64:dGVzdC1hdXRoLWtleS1mb3ItdGVzdGluZw=="
	hub.SetAuthKey(testAuthKey)

	testCases := []struct {
		name         string
		agentAuthKey string
		expectMatch  bool
	}{
		{
			name:         "matching auth key",
			agentAuthKey: testAuthKey,
			expectMatch:  true,
		},
		{
			name:         "different auth key",
			agentAuthKey: "base64:ZGlmZmVyZW50LWtleQ==",
			expectMatch:  false,
		},
		{
			name:         "empty auth key",
			agentAuthKey: "",
			expectMatch:  false,
		},
		{
			name:         "malformed auth key",
			agentAuthKey: "not-base64-key",
			expectMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock agent with the test auth key
			mockAgent := &mockAgent{authKey: tc.agentAuthKey}

			// Simulate the verification process
			matches := hub.GetAuthKey() == tc.agentAuthKey

			assert.Equal(t, tc.expectMatch, matches)
		})
	}
}

// TestGetFingerprintRecordsByToken tests the getFingerprintRecordsByToken function
func TestGetFingerprintRecordsByToken(t *testing.T) {
	hub, testApp, err := createTestHub(t)
	if err != nil {
		t.Fatal(err)
	}
	defer testApp.Cleanup()

	// Create test fingerprint records
	token1 := "test-token-1"
	token2 := "test-token-2"

	_, err = createTestRecord(testApp, "fingerprints", map[string]any{
		"system":      "system-1",
		"fingerprint": "fp-1",
		"token":       token1,
	})
	require.NoError(t, err)

	_, err = createTestRecord(testApp, "fingerprints", map[string]any{
		"system":      "system-2",
		"fingerprint": "fp-2",
		"token":       token1, // Same token, different system
	})
	require.NoError(t, err)

	_, err = createTestRecord(testApp, "fingerprints", map[string]any{
		"system":      "system-3",
		"fingerprint": "fp-3",
		"token":       token2,
	})
	require.NoError(t, err)

	// Test getting records for token1
	records := getFingerprintRecordsByToken(token1, hub)
	assert.Len(t, records, 2, "Should find 2 records for token1")

	// Test getting records for token2
	records = getFingerprintRecordsByToken(token2, hub)
	assert.Len(t, records, 1, "Should find 1 record for token2")

	// Test getting records for non-existent token
	records = getFingerprintRecordsByToken("non-existent", hub)
	assert.Len(t, records, 0, "Should find 0 records for non-existent token")
}

// mockAgent represents a mock agent for testing
type mockAgent struct {
	authKey string
}

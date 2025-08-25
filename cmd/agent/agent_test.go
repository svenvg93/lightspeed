package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAuthKey(t *testing.T) {
	// Generate a test RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Encode to base64 format
	keyBytes := x509.MarshalPKCS1PublicKey(&privKey.PublicKey)
	pubKeyData := "base64:" + base64.StdEncoding.EncodeToString(keyBytes)

	tests := []struct {
		name        string
		opts        cmdOptions
		envVars     map[string]string
		setupFiles  map[string][]byte
		wantErr     bool
		errContains string
	}{
		{
			name: "load key from flag",
			opts: cmdOptions{
				key: pubKeyData,
			},
		},
		{
			name: "load key from env var",
			envVars: map[string]string{
				"KEY": pubKeyData,
			},
		},
		{
			name: "load key from file",
			envVars: map[string]string{
				"KEY_FILE": "testkey.txt",
			},
			setupFiles: map[string][]byte{
				"testkey.txt": []byte(pubKeyData),
			},
		},
		{
			name:        "error when no key provided",
			wantErr:     true,
			errContains: "no authentication key provided",
		},
		{
			name: "error on invalid key file",
			envVars: map[string]string{
				"KEY_FILE": "nonexistent.txt",
			},
			wantErr:     true,
			errContains: "failed to read key file",
		},
		{
			name: "error on invalid key data",
			opts: cmdOptions{
				key: "invalid-key-data",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			if len(tt.setupFiles) > 0 {
				tmpDir := t.TempDir()
				for name, content := range tt.setupFiles {
					path := filepath.Join(tmpDir, name)
					err := os.WriteFile(path, content, 0600)
					require.NoError(t, err)
					if tt.envVars != nil {
						tt.envVars["KEY_FILE"] = path
					}
				}
			}

			// Set up environment
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			authKey, err := tt.opts.loadAuthKey()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, authKey)
			assert.True(t, strings.HasPrefix(authKey, "base64:"))
		})
	}
}

func TestParseFlags(t *testing.T) {
	// Save original command line arguments and restore after test
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	tests := []struct {
		name     string
		args     []string
		expected cmdOptions
	}{
		{
			name: "no flags",
			args: []string{"cmd"},
			expected: cmdOptions{
				key: "",
			},
		},
		{
			name: "key flag only",
			args: []string{"cmd", "-key", "testkey"},
			expected: cmdOptions{
				key: "testkey",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(tt.args[0], flag.ExitOnError)
			os.Args = tt.args

			var opts cmdOptions
			opts.parse()
			flag.Parse()

			assert.Equal(t, tt.expected, opts)
		})
	}
}

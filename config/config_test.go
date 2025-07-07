package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func createConfigFile(t *testing.T, content string) string {
	f, err := os.CreateTemp(os.TempDir(), "swo-config-test")
	require.NoError(t, err, "creating a temporary file should not fail")

	n, err := f.Write([]byte(content))
	require.Equal(t, n, len(content))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(f.Name())
	})

	return f.Name()
}

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name          string
		configFile    string
		apiURL        string
		token         string
		expected      Config
		expectedError error
		action        func()
	}{
		{
			name: "read full config file",
			expected: Config{
				APIURL: "https://api.solarwinds.com",
				Token:  "123456",
			},
			configFile: func() string {
				yamlStr := `
token: 123456
api-url: https://api.solarwinds.com
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name: "read token from config file",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "123456",
			},
			configFile: func() string {
				yamlStr := "token: 123456"
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name: "read token from env var",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "tokenFromEnvVar",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "tokenFromEnvVar")
				require.NoError(t, err)
			},
		},
		{
			name:          "missing token",
			expectedError: errMissingToken,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("SWO_API_TOKEN", "")
			_ = os.Setenv("SWO_API_URL", "")

			if tc.action != nil {
				tc.action()
			}

			cfg, err := Init(tc.configFile, DefaultAPIURL, "")
			require.True(t, errors.Is(err, tc.expectedError), "error: %v, expected: %v", err, tc.expectedError)
			if tc.expectedError != nil {
				return
			}

			require.Equal(t, &tc.expected, cfg)
		})
	}
}

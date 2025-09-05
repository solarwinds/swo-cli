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
			name:   "CLI args override config file",
			apiURL: "https://cli.example.com",
			token:  "cli_token",
			expected: Config{
				APIURL: "https://cli.example.com",
				Token:  "cli_token",
			},
			configFile: func() string {
				yamlStr := `
token: config_token
api-url: https://config.example.com
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:   "CLI args override env vars",
			apiURL: "https://cli.example.com",
			token:  "cli_token",
			expected: Config{
				APIURL: "https://cli.example.com",
				Token:  "cli_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "env_token")
				require.NoError(t, err)
				err = os.Setenv("SWO_API_URL", "https://env.example.com")
				require.NoError(t, err)
			},
		},
		{
			name: "env vars override config file",
			expected: Config{
				APIURL: "https://env.example.com",
				Token:  "env_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "env_token")
				require.NoError(t, err)
				err = os.Setenv("SWO_API_URL", "https://env.example.com")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `
token: config_token
api-url: https://config.example.com
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:  "partial CLI override - token only",
			token: "cli_token",
			expected: Config{
				APIURL: "https://env.example.com",
				Token:  "cli_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_URL", "https://env.example.com")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `
token: config_token
api-url: https://config.example.com
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:   "partial CLI override - apiURL only",
			apiURL: "https://cli.example.com",
			expected: Config{
				APIURL: "https://cli.example.com",
				Token:  "env_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "env_token")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `
token: config_token
api-url: https://config.example.com
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
			name:  "fallback to default API URL",
			token: "test_token",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "test_token",
			},
		},
		{
			name:          "missing token",
			expectedError: errMissingToken,
		},
		{
			name:  "missing API URL should use default",
			token: "test_token",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "test_token",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("SWO_API_TOKEN", "")
			_ = os.Setenv("SWO_API_URL", "")

			if tc.action != nil {
				tc.action()
			}

			cfg, err := Init(tc.configFile, tc.apiURL, tc.token)
			require.True(t, errors.Is(err, tc.expectedError), "error: %v, expected: %v", err, tc.expectedError)
			if tc.expectedError != nil {
				return
			}

			require.Equal(t, &tc.expected, cfg)
		})
	}
}

func TestTrimmingWhitespace(t *testing.T) {
	testCases := []struct {
		name       string
		configFile string
		apiURL     string
		token      string
		expected   Config
		action     func()
	}{
		{
			name:   "trim CLI arguments",
			apiURL: "  https://cli.example.com  ",
			token:  "  cli_token  ",
			expected: Config{
				APIURL: "https://cli.example.com",
				Token:  "cli_token",
			},
		},
		{
			name: "trim env variables",
			expected: Config{
				APIURL: "https://env.example.com",
				Token:  "env_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "  env_token  ")
				require.NoError(t, err)
				err = os.Setenv("SWO_API_URL", "  https://env.example.com  ")
				require.NoError(t, err)
			},
		},
		{
			name: "trim config file values",
			expected: Config{
				APIURL: "https://config.example.com",
				Token:  "config_token",
			},
			configFile: func() string {
				yamlStr := `
token: "  config_token  "
api-url: "  https://config.example.com  "
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:   "whitespace-only CLI args should be treated as empty",
			apiURL: "   ",
			token:  "   ",
			expected: Config{
				APIURL: "https://env.example.com",
				Token:  "env_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "env_token")
				require.NoError(t, err)
				err = os.Setenv("SWO_API_URL", "https://env.example.com")
				require.NoError(t, err)
			},
		},
		{
			name: "whitespace-only env vars should be treated as empty",
			expected: Config{
				APIURL: "https://config.example.com",
				Token:  "config_token",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "   ")
				require.NoError(t, err)
				err = os.Setenv("SWO_API_URL", "   ")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `
token: config_token
api-url: https://config.example.com
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:  "whitespace-only config values should be treated as empty",
			token: "cli_token",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "cli_token",
			},
			configFile: func() string {
				yamlStr := `
token: "   "
api-url: "   "
`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:  "mixed whitespace scenarios",
			token: "  cli_token  ", // CLI token with whitespace
			expected: Config{
				APIURL: "https://env.example.com", // env var should be trimmed
				Token:  "cli_token",               // CLI token should be trimmed
			},
			action: func() {
				err := os.Setenv("SWO_API_URL", "  https://env.example.com  ")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `
token: "  config_token  "
api-url: "  https://config.example.com  "
`
				return createConfigFile(t, yamlStr)
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("SWO_API_TOKEN", "")
			_ = os.Setenv("SWO_API_URL", "")

			if tc.action != nil {
				tc.action()
			}

			cfg, err := Init(tc.configFile, tc.apiURL, tc.token)
			require.NoError(t, err)
			require.Equal(t, &tc.expected, cfg)
		})
	}
}

func TestErrorCases(t *testing.T) {
	testCases := []struct {
		name          string
		configFile    string
		apiURL        string
		token         string
		expectedError error
		action        func()
	}{
		{
			name:          "missing token - no sources",
			expectedError: errMissingToken,
		},
		{
			name:          "whitespace-only token from all sources",
			expectedError: errMissingToken,
			apiURL:        "https://test.com",
			token:         "   ",
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "   ")
				require.NoError(t, err)
			},
			configFile: func() string {
				yamlStr := `token: "   "`
				return createConfigFile(t, yamlStr)
			}(),
		},
		{
			name:   "invalid YAML config file",
			apiURL: "https://test.com",
			// Don't provide token via CLI so config file will be read
			configFile: func() string {
				yamlStr := `
token: test_token
api-url: [invalid yaml structure
`
				return createConfigFile(t, yamlStr)
			}(),
			expectedError: errors.New("error while unmarshaling"), // This will be wrapped, so we'll check with Contains
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("SWO_API_TOKEN", "")
			_ = os.Setenv("SWO_API_URL", "")

			if tc.action != nil {
				tc.action()
			}

			cfg, err := Init(tc.configFile, tc.apiURL, tc.token)

			if tc.name == "invalid YAML config file" {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error while unmarshaling")
				require.Nil(t, cfg)
			} else if tc.name == "missing API URL uses default" {
				require.NoError(t, err)
				require.Equal(t, DefaultAPIURL, cfg.APIURL)
				require.Equal(t, "test_token", cfg.Token)
			} else {
				require.True(t, errors.Is(err, tc.expectedError), "error: %v, expected: %v", err, tc.expectedError)
				require.Nil(t, cfg)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		apiURL   string
		token    string
		expected Config
		action   func()
	}{
		{
			name:   "tabs and newlines get trimmed",
			apiURL: "\t\nhttps://test.com\t\n",
			token:  "\t\ntest_token\t\n",
			expected: Config{
				APIURL: "https://test.com",
				Token:  "test_token",
			},
		},
		{
			name:  "empty strings vs whitespace strings from env",
			token: "cli_token",
			expected: Config{
				APIURL: DefaultAPIURL,
				Token:  "cli_token",
			},
			action: func() {
				// Set empty string vs whitespace - both should be treated as empty
				err := os.Setenv("SWO_API_URL", "")
				require.NoError(t, err)
			},
		},
		{
			name:   "zero-width unicode spaces are NOT trimmed by strings.TrimSpace",
			apiURL: "\u200B\u200C\u200D\u2060", // Zero-width spaces
			token:  "valid_token",
			expected: Config{
				APIURL: "\u200B\u200C\u200D\u2060", // These should remain as-is since TrimSpace doesn't handle Unicode
				Token:  "valid_token",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Setenv("SWO_API_TOKEN", "")
			_ = os.Setenv("SWO_API_URL", "")

			if tc.action != nil {
				tc.action()
			}

			cfg, err := Init("", tc.apiURL, tc.token)
			require.NoError(t, err)
			require.Equal(t, &tc.expected, cfg)
		})
	}
}

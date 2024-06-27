package logs

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	fixedTime, err := time.Parse(time.DateTime, "2000-01-01 10:00:30")
	require.NoError(t, err)

	testCases := []struct {
		name          string
		action        func()
		opts          *Options
		expected      Options
		expectedError error
	}{
		{
			name: "read full config file",
			opts: &Options{configFile: configFile},
			expected: Options{
				args:       []string{},
				configFile: configFile,
				ApiUrl:     "https://api.solarwinds.com",
				Token:      "123456",
			},
			action: func() {
				yamlStr := `
token: 123456
api-url: https://api.solarwinds.com
`
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name: "read token from config file",
			opts: &Options{configFile: configFile, ApiUrl: DefaultApiUrl},
			expected: Options{
				args:       []string{},
				configFile: configFile,
				ApiUrl:     DefaultApiUrl,
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name: "read token from env var",
			opts: &Options{configFile: DefaultConfigFile, ApiUrl: DefaultApiUrl},
			expected: Options{
				args:       []string{},
				configFile: DefaultConfigFile,
				ApiUrl:     DefaultApiUrl,
				Token:      "tokenFromEnvVar",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "tokenFromEnvVar")
				require.NoError(t, err)
			},
		},
		{
			name: "missing token",
			opts: &Options{},
			expected: Options{
				args: []string{},
			},
			expectedError: errMissingToken,
		},
		{
			name: "parse human readable min time",
			opts: &Options{configFile: configFile, minTime: "5 seconds ago"},
			expected: Options{
				args:       []string{},
				configFile: configFile,
				minTime:    "2000-01-01T10:00:25Z",
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
				now = fixedTime
			},
		},
		{
			name: "parse human readable max time",
			opts: &Options{configFile: configFile, maxTime: "in 5 seconds"},
			expected: Options{
				args:       []string{},
				configFile: configFile,
				maxTime:    "2000-01-01T10:00:35Z",
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
				now = fixedTime
			},
		},
		{
			name: "fail parsing min time",
			opts: &Options{minTime: "what?"},
			action: func() {
				now = fixedTime
			},
			expectedError: errMinTimeFlag,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(configFile)
			if tc.action != nil {
				tc.action()
			}

			err := tc.opts.Init([]string{})
			require.True(t, errors.Is(err, tc.expectedError), "error: %v, expected: %v", err, tc.expectedError)
			if tc.expectedError != nil {
				return
			}

			require.Equal(t, &tc.expected, tc.opts)
		})

		os.Setenv("SWO_API_TOKEN", "")
	}
}

func TestParseTime(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RFC3339",
			input:    "2000-01-01T12:13:14Z",
			expected: "2000-01-01T12:13:14Z",
		},
		{
			name:     "RFC822Z",
			input:    "04 Feb 00 13:14 MST",
			expected: "2000-02-04T13:14:00Z",
		},
		{
			name:     "human readable",
			input:    "5 seconds ago",
			expected: "2000-01-01T10:00:25Z",
		},
		{
			name:     "append UTC at the end",
			input:    "2024-05-13 13:00:00 UTC",
			expected: "2024-05-13T13:00:00Z",
		},
	}

	fixedTime, err := time.Parse(time.DateTime, "2000-01-01 10:00:30")
	require.NoError(t, err)

	now = fixedTime

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTime(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, result)
		})
	}
}

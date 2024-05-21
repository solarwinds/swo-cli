package logs

import (
	"errors"
	"os"
	"path/filepath"
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
		flags         []string
		action        func()
		expected      Options
		expectedError error
	}{
		{
			name:  "default flag values",
			flags: []string{"--configfile", filepath.Join(os.TempDir(), "config-file.yaml")},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				ApiUrl:     defaultApiUrl,
				Token:      "123456",
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name:  "many flags",
			flags: []string{"--configfile", filepath.Join(os.TempDir(), "config-file.yaml"), "--count", "5", "--group", "groupValue", "--system", "systemValue", "--color", "program"},
			expected: Options{
				args:       []string{},
				count:      5,
				group:      "groupValue",
				system:     "systemValue",
				color:      program,
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
				ApiUrl:     defaultApiUrl,
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name:  "many flags and args",
			flags: []string{"--configfile", filepath.Join(os.TempDir(), "config-file.yaml"), "--count", "5", "--group", "groupValue", "one", "two", "three"},
			expected: Options{
				args:       []string{"one", "two", "three"},
				count:      5,
				group:      "groupValue",
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
				ApiUrl:     defaultApiUrl,
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name:          "invalid color value",
			flags:         []string{"--color", "yellow"},
			expected:      Options{},
			expectedError: errColorFlag,
		},
		{
			name:  "read full config file",
			flags: []string{"--configfile", filepath.Join(os.TempDir(), "config-file.yaml")},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
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
			name:  "read token from config file",
			flags: []string{"--configfile", filepath.Join(os.TempDir(), "config-file.yaml")},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
				ApiUrl:     defaultApiUrl,
				Token:      "123456",
			},
			action: func() {
				yamlStr := "token: 123456"
				createConfigFile(t, configFile, yamlStr)
			},
		},
		{
			name:  "read token from env var",
			flags: []string{},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: defaultConfigFile,
				ApiUrl:     defaultApiUrl,
				Token:      "tokenFromEnvVar",
			},
			action: func() {
				err := os.Setenv("SWO_API_TOKEN", "tokenFromEnvVar")
				require.NoError(t, err)
			},
		},
		{
			name:  "missing token",
			flags: []string{},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: defaultConfigFile,
				ApiUrl:     defaultApiUrl,
			},
			expectedError: errMissingToken,
		},
		{
			name:  "parse human readable min time",
			flags: []string{"--min-time", "5 seconds ago", "--configfile", filepath.Join(os.TempDir(), "config-file.yaml")},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
				ApiUrl:     defaultApiUrl,
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
			name:  "parse human readable max time",
			flags: []string{"--max-time", "in 5 seconds", "--configfile", filepath.Join(os.TempDir(), "config-file.yaml")},
			expected: Options{
				args:       []string{},
				count:      defaultCount,
				configFile: filepath.Join(os.TempDir(), "config-file.yaml"),
				ApiUrl:     defaultApiUrl,
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
			name:  "fail parsing min time",
			flags: []string{"--min-time", "what?"},
			action: func() {
				now = fixedTime
			},
			expectedError: errMinTimeFlag,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(filepath.Join(os.TempDir(), "config-file.yaml"))
			if tc.action != nil {
				tc.action()
			}

			cmd := NewLogsCommand()
			err := cmd.Init(tc.flags)
			require.True(t, errors.Is(err, tc.expectedError), "error: %v, expected: %v", err, tc.expectedError)
			if tc.expectedError != nil {
				return
			}

			require.Equal(t, &tc.expected, cmd.opts)
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

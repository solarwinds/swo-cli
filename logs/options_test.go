package logs

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
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
			name: "parse human readable min time",
			opts: &Options{minTime: "5 seconds ago"},
			expected: Options{
				args:    []string{},
				minTime: inLocal(fixedTime.Add(-5 * time.Second)),
			},
			action: func() {
				now = fixedTime
			},
		},
		{
			name: "parse human readable max time",
			opts: &Options{maxTime: "in 5 seconds"},
			expected: Options{
				args:    []string{},
				maxTime: inLocal(fixedTime.Add(5 * time.Second)),
			},
			action: func() {
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

		_ = os.Setenv("SWO_API_TOKEN", "")
	}
}

func TestParseTime(t *testing.T) {
	fixedTime, err := time.Parse(time.DateTime, "2000-01-01 10:00:30")
	require.NoError(t, err)

	now = fixedTime

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RFC3339",
			input:    "2000-01-01T12:13:14Z",
			expected: inLocal(time.Date(2000, 1, 1, 12, 13, 14, 0, time.UTC)),
		},
		{
			name:     "RFC822Z",
			input:    "04 Feb 00 13:14 +0000",
			expected: inLocal(time.Date(2000, 2, 4, 13, 14, 0, 0, time.UTC)),
		},
		{
			name:     "human readable",
			input:    "5 seconds ago",
			expected: inLocal(fixedTime.Add(-5 * time.Second)),
		},
		{
			name:     "append UTC at the end",
			input:    "2024-05-13 13:00:00 UTC",
			expected: "2024-05-13T13:00:00Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseTime(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, result)
		})
	}
}

// inLocal formats t as parseTime does: RFC3339 in the machine's local zone.
func inLocal(t time.Time) string {
	return t.In(time.Local).Format(time.RFC3339)
}

package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	configFile = filepath.Join(os.TempDir(), "config-file.yaml")
	logsData   = LogsData{
		Logs: []Log{
			{
				Time:     time.Now(),
				Message:  "messageOne",
				Hostname: "hostnameOne",
				Severity: "severityOne",
				Program:  "programOne",
			},
			{
				Time:     time.Now(),
				Message:  "messageTwo",
				Hostname: "hostnameTwo",
				Severity: "severityTwo",
				Program:  "programTwo",
			},
		},
		PageInfo: PageInfo{PrevPage: "prevPageValue"},
	}
)

func createConfigFile(t *testing.T, filename, content string) {
	_ = os.Remove(filename)
	f, err := os.Create(filename)
	require.NoError(t, err, "creating a temporary file should not fail")

	n, err := f.Write([]byte(content))
	require.Equal(t, n, len(content))
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(filename)
	})
}

func TestPrepareRequest(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	token := "1234567"
	yamlStr := fmt.Sprintf("token: %s", token)
	createConfigFile(t, configFile, yamlStr)

	fixedTime, err := time.Parse(time.DateTime, "2000-01-01 10:00:30")
	require.NoError(t, err)
	now = fixedTime

	testCases := []struct {
		name           string
		options        *Options
		expectedValues map[string][]string
		expectedError  error
	}{
		{
			name:           "default request",
			options:        &Options{configFile: configFile, ApiUrl: DefaultApiUrl},
			expectedValues: map[string][]string{},
		},
		{
			name: "custom count group startTime and endTime",
			options: &Options{
				configFile: configFile,
				group:      "groupValue",
				minTime:    "10 seconds ago",
				maxTime:    "2 seconds ago",
			},
			expectedValues: map[string][]string{
				"group":     {"groupValue"},
				"startTime": {"2000-01-01T10:00:20Z"},
				"endTime":   {"2000-01-01T10:00:28Z"},
			},
		},
		{
			name:    "system flag",
			options: &Options{configFile: configFile, system: "systemValue"},
			expectedValues: map[string][]string{
				"filter": {`host:"systemValue"`},
			},
		},
		{
			name: "system flag with filter",
			options: &Options{
				args:       []string{`"access denied"`, "1.2.3.4", "-sshd"},
				configFile: configFile,
				system:     "systemValue",
			},
			expectedValues: map[string][]string{
				"filter": func() []string {
					escaped := url.PathEscape("filter=host:\"systemValue\" \"access denied\" 1.2.3.4 -sshd")
					values, err := url.ParseQuery(escaped)
					require.NoError(t, err)
					value, ok := values["filter"]
					require.True(t, ok)
					return value
				}(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, err)
			client, err := NewClient(tc.options)
			require.NoError(t, err)

			request, err := client.prepareRequest(context.Background(), "")
			require.NoError(t, err)

			values := request.URL.Query()
			for k, v := range tc.expectedValues {
				require.ElementsMatch(t, v, values[k])
			}

			header := request.Header
			for k, v := range map[string][]string{
				"Authorization": {fmt.Sprintf("Bearer %s", token)},
				"Accept":        {"application/json"},
			} {
				require.ElementsMatch(t, v, header[k])
			}
		})
	}

}

func TestRun(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	handler := func(w http.ResponseWriter, _ *http.Request) {
		data, err := json.Marshal(logsData)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")

		_, err = w.Write(data)
		require.NoError(t, err)
	}

	wg := sync.WaitGroup{}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		mux.HandleFunc("/v1/logs", handler)
		err = server.Serve(listener)
	}()

	token := "1234567"
	yamlStr := fmt.Sprintf(`
token: %s
api-url: %s
`, token, fmt.Sprintf("http://%s", listener.Addr().String()))
	createConfigFile(t, configFile, yamlStr)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	client, err := NewClient(&Options{
		configFile: configFile,
		json:       true,
	})
	require.NoError(t, err)

	client.output = w

	outputCompareDone := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(outputCompareDone)

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		expectedOutput := ""
		for i, l := range logsData.Logs {
			data, err := json.Marshal(l)
			require.NoError(t, err)

			expectedOutput += string(data)
			if i != len(logsData.Logs)-1 {
				expectedOutput += "\n"
			}
		}

		require.NoError(t, err)
		require.Equal(t, expectedOutput, string(output[:len(output)-1])) // last char is a new line character
	}()

	go func() {
		<-outputCompareDone
		_ = server.Shutdown(context.Background())
	}()

	err = client.Run(context.Background())
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	wg.Wait()
}

func TestPrintResultStandard(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	createConfigFile(t, configFile, "token: 1234567")
	client, err := NewClient(&Options{configFile: configFile})
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	client.output = w

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		expectStr := fmt.Sprintf(`%s hostnameOne programOne messageOne
%s hostnameTwo programTwo messageTwo
`, logsData.Logs[0].Time.Format("Jan 02 15:04:05"), logsData.Logs[1].Time.Format("Jan 02 15:04:05")) // SWO returns fresh logs as first in the logs list
		require.Equal(t, expectStr, string(output))
	}()

	err = client.printResult(logsData.Logs)
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	wg.Wait()
}

func TestPrintResultJSON(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	createConfigFile(t, configFile, "token: 1234567")
	client, err := NewClient(&Options{configFile: configFile, json: true})
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	client.output = w

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		expectedStr := `{"time":"%s","message":"messageOne","hostname":"hostnameOne","severity":"severityOne","program":"programOne"}
						{"time":"%s","message":"messageTwo","hostname":"hostnameTwo","severity":"severityTwo","program":"programTwo"}
		`
		trimmed := strings.TrimSpace(fmt.Sprintf(expectedStr, logsData.Logs[0].Time.Format(time.RFC3339Nano), logsData.Logs[1].Time.Format(time.RFC3339Nano)))
		trimmed = strings.ReplaceAll(trimmed, "\t", "")
		require.Equal(t, trimmed, string(output[:len(output)-1])) // last char is a new line character
	}()

	err = client.printResult(logsData.Logs)
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	wg.Wait()
}

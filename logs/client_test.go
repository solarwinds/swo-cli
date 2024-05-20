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

	"github.com/solarwinds/swo-cli/version"
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
		flags          []string
		expectedValues map[string][]string
		expectedError  error
	}{
		{
			name:  "default request",
			flags: []string{"--configfile", configFile},
			expectedValues: map[string][]string{
				"pageSize": {"100"},
			},
		},
		{
			name:  "custom pageSize group startTime and endTime",
			flags: []string{"--configfile", configFile, "--count", "8", "--group", "groupValue", "--min-time", "10 seconds ago", "--max-time", "2 seconds ago"},
			expectedValues: map[string][]string{
				"pageSize":  {"8"},
				"group":     {"groupValue"},
				"startTime": {"2000-01-01T10:00:20Z"},
				"endTime":   {"2000-01-01T10:00:28Z"},
			},
		},
		{
			name:  "system flag",
			flags: []string{"--configfile", configFile, "--system", "systemValue"},
			expectedValues: map[string][]string{
				"pageSize": {"100"},
				"filter":   {"host:systemValue"},
			},
		},
		{
			name:  "system flag with filter",
			flags: []string{"--configfile", configFile, "--system", "systemValue", "--", "\"access denied\"", "1.2.3.4", "-sshd"},
			expectedValues: map[string][]string{
				"pageSize": {"100"},
				"filter": func() []string {
					escaped := url.PathEscape("filter=host:systemValue \"access denied\" 1.2.3.4 -sshd")
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
			cmd := NewLogsCommand()
			err := cmd.Init(tc.flags)
			require.NoError(t, err)

			request, err := cmd.client.prepareRequest(context.Background())
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

	server := &http.Server{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		http.HandleFunc("/v1/logs", handler)
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

	cmd := NewLogsCommand()
	err = cmd.Init([]string{"--configfile", configFile, "--json"})
	require.NoError(t, err)

	cmd.client.output = w

	outputComapreDone := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(outputComapreDone)

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		data, err := json.Marshal(logsData)
		require.NoError(t, err)
		require.Equal(t, string(data), string(output[:len(output)-1])) // last char is a new line character
	}()

	go func() {
		<-outputComapreDone
		_ = server.Shutdown(context.Background())
	}()

	err = cmd.client.Run(context.Background())
	require.NoError(t, err)

	w.Close()

	wg.Wait()
}

func TestPrintResultStandard(t *testing.T) {
	location, err := time.LoadLocation("GMT")
	require.NoError(t, err)

	time.Local = location

	createConfigFile(t, configFile, "token: 1234567")
	cmd := NewLogsCommand()
	err = cmd.Init([]string{"--configfile", configFile})
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	cmd.client.output = w

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		expectStr := fmt.Sprintf(`%s hostnameTwo programTwo messageTwo
%s hostnameOne programOne messageOne
`, logsData.Logs[1].Time.Format("Jan 02 15:04:05"), logsData.Logs[0].Time.Format("Jan 02 15:04:05")) // SWO returns fresh logs as first in the logs list
		require.Equal(t, expectStr, string(output))
	}()

	err = cmd.client.printResult(&logsData)
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
	cmd := NewLogsCommand()
	err = cmd.Init([]string{"--configfile", configFile, "--json"})
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	cmd.client.output = w

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		output, err := io.ReadAll(r)
		require.NoError(t, err)

		expectedStr := `
		{
			"logs":[
				{
					"time":"%s",
					"message":"messageOne",
					"hostname":"hostnameOne",
					"severity":"severityOne",
					"program":"programOne"
				},
				{
					"time":"%s",
					"message":"messageTwo",
					"hostname":"hostnameTwo",
					"severity":"severityTwo",
					"program":"programTwo"
				}
			],
			"pageInfo":{
				"prevPage":"prevPageValue",
				"nextPage":""
			}
		}
		`
		trimmed := strings.TrimSpace(fmt.Sprintf(expectedStr, logsData.Logs[0].Time.Format(time.RFC3339Nano), logsData.Logs[1].Time.Format(time.RFC3339Nano)))
		trimmed = strings.ReplaceAll(trimmed, "\t", "")
		trimmed = strings.ReplaceAll(trimmed, "\n", "")
		require.Equal(t, trimmed, string(output[:len(output)-1])) // last char is a new line character
	}()

	err = cmd.client.printResult(&logsData)
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	wg.Wait()
}

func TestRunVersion(t *testing.T) {
	createConfigFile(t, configFile, "token: 1234567")

	cmd := NewLogsCommand()
	err := cmd.Init([]string{"--configfile", configFile, "--version"})
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)
	cmd.client.output = w

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		output, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, version.Version+"\n", string(output))
	}()

	err = cmd.client.Run(context.Background())
	require.NoError(t, err)

	w.Close()

	wg.Wait()
}

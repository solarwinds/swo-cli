package logs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/solarwinds/swo-cli/shared"
)

func newTestClient(t *testing.T, apiURL string, follow bool) *Client {
	t.Helper()
	opts := &Options{
		BaseOptions: shared.BaseOptions{Token: "secret-token", APIURL: apiURL},
		json:        true,
	}
	require.NoError(t, opts.Init([]string{}))
	opts.follow = follow

	client, err := NewClient(opts)
	require.NoError(t, err)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = devNull.Close() })
	client.output = devNull

	return client
}

func writePage(t *testing.T, w http.ResponseWriter, resp getLogsResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func TestRunTimesOutOnHungServer(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	client := newTestClient(t, server.URL, false)
	client.httpClient.Timeout = 50 * time.Millisecond

	err := client.Run(context.Background())
	require.ErrorIs(t, err, shared.ErrRequestTimeout)
	require.NotContains(t, err.Error(), "secret-token")
}

func TestRunFollowStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writePage(t, w, getLogsResponse{pageInfo: pageInfo{NextPage: "/v1/logs?skipToken=a"}})
		cancel()
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, true)
	client.followInterval = time.Hour

	start := time.Now()
	err := client.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestRunRepeatedNextPage(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writePage(t, w, getLogsResponse{Logs: logsData.Logs, pageInfo: pageInfo{NextPage: "/v1/logs?skipToken=same"}})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, false)

	err := client.Run(context.Background())
	require.ErrorIs(t, err, shared.ErrRepeatedNextPage)
	require.Equal(t, int32(2), requests.Load())
}

func TestRunFollowAllowsRepeatedNextPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) >= 3 {
			cancel()
		}
		writePage(t, w, getLogsResponse{pageInfo: pageInfo{NextPage: "/v1/logs?skipToken=same"}})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, true)
	client.followInterval = time.Millisecond

	err := client.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, requests.Load(), int32(3))
}

func TestRunRepeatedEmptyNextPageEndsResults(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := getLogsResponse{pageInfo: pageInfo{NextPage: "/v1/logs?skipToken=same"}}
		if requests.Add(1) == 1 {
			resp.Logs = logsData.Logs
		}
		writePage(t, w, resp)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, false)

	require.NoError(t, client.Run(context.Background()))
	require.Equal(t, int32(2), requests.Load())
}

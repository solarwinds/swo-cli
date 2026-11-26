package entities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/solarwinds/swo-cli/shared"
)

func newRobustnessTestClient(t *testing.T, opts *Options) *Client {
	t.Helper()
	client, err := NewClient(opts)
	require.NoError(t, err)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = devNull.Close() })
	client.output = devNull

	return client
}

func TestGetEntityTimesOutOnHungServer(t *testing.T) {
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

	client := newRobustnessTestClient(t, &Options{ID: "e-1", BaseOptions: shared.BaseOptions{Token: "secret-token", APIURL: server.URL}})
	client.httpClient.Timeout = 50 * time.Millisecond

	err := client.GetEntity(context.Background())
	require.ErrorIs(t, err, shared.ErrRequestTimeout)
	require.NotContains(t, err.Error(), "secret-token")
}

func TestListEntitiesRepeatedNextPage(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		resp := listEntitiesResponse{Entities: testEntities, pageInfo: pageInfo{NextPage: "/v1/entities?skipToken=same"}}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client := newRobustnessTestClient(t, &Options{Type: "Service", BaseOptions: shared.BaseOptions{Token: "token", APIURL: server.URL}})

	err := client.ListEntities(context.Background())
	require.ErrorIs(t, err, shared.ErrRepeatedNextPage)
	require.Equal(t, int32(2), requests.Load())
}

func TestEntityIDIsPathEscaped(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		mu.Unlock()
		if err := json.NewEncoder(w).Encode(testEntities[0]); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	for _, id := range []string{"../metadata", "a/b?c#d"} {
		opts := &Options{ID: id, Tags: map[string]string{"k": "v"}, BaseOptions: shared.BaseOptions{Token: "token", APIURL: server.URL}}
		client := newRobustnessTestClient(t, opts)
		require.NoError(t, client.GetEntity(context.Background()))
		require.NoError(t, client.UpdateEntity(context.Background()))
	}

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{
		"GET /v1/entities/..%2Fmetadata",
		"GET /v1/entities/..%2Fmetadata",
		"PUT /v1/entities/..%2Fmetadata",
		"GET /v1/entities/a%2Fb%3Fc%23d",
		"GET /v1/entities/a%2Fb%3Fc%23d",
		"PUT /v1/entities/a%2Fb%3Fc%23d",
	}, paths)
}

func TestDotEntityIDsAreRejected(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(server.Close)

	for _, id := range []string{".", ".."} {
		client := newRobustnessTestClient(t, &Options{ID: id, BaseOptions: shared.BaseOptions{Token: "token", APIURL: server.URL}})
		require.ErrorIs(t, client.GetEntity(context.Background()), errInvalidEntityID)
	}
	require.Zero(t, requests.Load())
}

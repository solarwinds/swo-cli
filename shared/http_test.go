package shared

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// newHungServer returns a server that never responds until the client gives up or the test ends.
func newHungServer(t *testing.T) *httptest.Server {
	t.Helper()
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
	return server
}

func TestDoTimesOutOnHungServer(t *testing.T) {
	server := newHungServer(t)

	req, err := NewRequest(context.Background(), http.MethodGet, server.URL, "secret-token", nil)
	require.NoError(t, err)

	start := time.Now()
	_, err = Do(&http.Client{Timeout: 50 * time.Millisecond}, req)
	require.ErrorIs(t, err, ErrRequestTimeout)
	require.NotContains(t, err.Error(), "secret-token")
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestDoCanceledContextIsNotTimeout(t *testing.T) {
	server := newHungServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	req, err := NewRequest(ctx, http.MethodGet, server.URL, "token", nil)
	require.NoError(t, err)

	time.AfterFunc(20*time.Millisecond, cancel)
	_, err = Do(NewHTTPClient(), req)
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrRequestTimeout)
}

func TestDoRejectsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		chunk := []byte(strings.Repeat("a", 1<<20))
		for written := int64(0); written <= MaxResponseBodySize; written += int64(len(chunk)) {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)

	req, err := NewRequest(context.Background(), http.MethodGet, server.URL, "token", nil)
	require.NoError(t, err)

	body, err := Do(NewHTTPClient(), req)
	require.ErrorIs(t, err, ErrResponseTooLarge)
	require.Nil(t, body)
}

func TestDoAcceptsBodyAtLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", int(MaxResponseBodySize))))
	}))
	t.Cleanup(server.Close)

	req, err := NewRequest(context.Background(), http.MethodGet, server.URL, "token", nil)
	require.NoError(t, err)

	body, err := Do(NewHTTPClient(), req)
	require.NoError(t, err)
	require.Len(t, body, int(MaxResponseBodySize))
}

func TestDoTruncatesErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("#", 10_000)))
	}))
	t.Cleanup(server.Close)

	req, err := NewRequest(context.Background(), http.MethodGet, server.URL, "token", nil)
	require.NoError(t, err)

	_, err = Do(NewHTTPClient(), req)
	require.ErrorIs(t, err, ErrInvalidAPIResponse)
	require.Contains(t, err.Error(), "500")
	require.Contains(t, err.Error(), "...(truncated)")
	require.Equal(t, maxErrorBodyLen, strings.Count(err.Error(), "#"))
}

func TestNewRequestHeaders(t *testing.T) {
	req, err := NewRequest(context.Background(), http.MethodPut, "https://example.com", "token", strings.NewReader("{}"))
	require.NoError(t, err)
	require.Equal(t, "Bearer token", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Accept"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))

	req, err = NewRequest(context.Background(), http.MethodGet, "https://example.com", "token", nil)
	require.NoError(t, err)
	require.Empty(t, req.Header.Get("Content-Type"))
}

func TestPageGuard(t *testing.T) {
	g := PageGuard{}
	require.NoError(t, g.Check("/v1/logs?skipToken=a"))
	require.NoError(t, g.Check("/v1/logs?skipToken=b"))
	require.ErrorIs(t, g.Check("/v1/logs?skipToken=a"), ErrRepeatedNextPage)
}

func TestSleep(t *testing.T) {
	require.NoError(t, Sleep(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	require.ErrorIs(t, Sleep(ctx, time.Hour), context.Canceled)
	require.Less(t, time.Since(start), time.Second)
}

func TestTruncateKeepsValidUTF8(t *testing.T) {
	got := truncate([]byte("aé"), 2)
	require.Equal(t, "a...(truncated)", got)
	require.True(t, utf8.ValidString(got))
}

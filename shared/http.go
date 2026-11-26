package shared

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
	"unicode/utf8"
)

const (
	// RequestTimeout bounds a single API request, including reading the response body.
	RequestTimeout = 30 * time.Second
	// MaxResponseBodySize is the largest response body the CLI accepts.
	MaxResponseBodySize int64 = 32 << 20
	// maxErrorBodyLen is how much of an error response body is included in error messages.
	maxErrorBodyLen = 512
)

var (
	// ErrInvalidAPIResponse indicates a non-2xx status code was received from the API
	ErrInvalidAPIResponse = errors.New("received non-2xx status code")
	// ErrRequestTimeout indicates the API did not respond within the request timeout
	ErrRequestTimeout = errors.New("request to SWO timed out")
	// ErrResponseTooLarge indicates the response body exceeded MaxResponseBodySize
	ErrResponseTooLarge = errors.New("response body too large")
	// ErrRepeatedNextPage indicates the API returned a page link that was already fetched
	ErrRepeatedNextPage = errors.New("API returned an already fetched nextPage")
)

// NewHTTPClient returns an HTTP client with RequestTimeout applied to each request.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: RequestTimeout}
}

// NewRequest creates an authenticated JSON API request.
func NewRequest(ctx context.Context, method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/json")
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// Do sends req and returns the response body, bounded by MaxResponseBodySize.
// Non-2xx responses return ErrInvalidAPIResponse with a truncated body.
func Do(client *http.Client, req *http.Request) ([]byte, error) {
	slog.Debug("API Request", "method", req.Method, "url", req.URL.String()) //nolint:gosec

	response, err := client.Do(req) //nolint:gosec
	if err != nil {
		return nil, transportError(req, "error while sending http request to SWO", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			slog.Error("Could not close http body", "error", err)
		}
	}()

	slog.Debug("Response status", "status_code", response.StatusCode, "status", response.Status)

	content, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBodySize+1))
	if err != nil {
		return nil, transportError(req, "error while reading http response body from SWO", err)
	}
	tooLarge := int64(len(content)) > MaxResponseBodySize

	slog.Debug("Response body", "length_bytes", len(content))

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("%w: %d, response body: %s", ErrInvalidAPIResponse, response.StatusCode, truncate(content, maxErrorBodyLen))
	}

	if tooLarge {
		return nil, fmt.Errorf("%w: exceeds %d bytes", ErrResponseTooLarge, MaxResponseBodySize)
	}

	return content, nil
}

func transportError(req *http.Request, msg string, err error) error {
	var netErr net.Error
	if req.Context().Err() == nil && errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %w", ErrRequestTimeout, err)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	// Cut at a rune boundary so the message stays valid UTF-8.
	for n > 0 && !utf8.RuneStart(b[n]) {
		n--
	}
	return string(b[:n]) + "...(truncated)"
}

// PageGuard detects pagination loops caused by the API returning an already fetched page.
type PageGuard map[string]struct{}

// Check records nextPage and returns ErrRepeatedNextPage if it was seen before.
func (g PageGuard) Check(nextPage string) error {
	if _, ok := g[nextPage]; ok {
		return ErrRepeatedNextPage
	}
	g[nextPage] = struct{}{}
	return nil
}

// Sleep waits for d or until ctx is done, returning ctx.Err() in the latter case.
func Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

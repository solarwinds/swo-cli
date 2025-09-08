// Package logs provides a client for retrieving and displaying logs from the SWO API.
package logs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/solarwinds/swo-cli/shared"
)

var (
	// ErrInvalidAPIResponse indicates a non-2xx status code was received from the API
	ErrInvalidAPIResponse = errors.New("received non-2xx status code")
	// ErrInvalidDateTime indicates a timestamp could not be parsed
	ErrInvalidDateTime = errors.New("could not parse timestamp")
	// ErrNoContent indicates an empty response body was received from the API
	ErrNoContent = errors.New("no content")
)

// Client is a logs client
type Client struct {
	opts       *Options
	httpClient http.Client
	output     *os.File
}

type log struct {
	Time     time.Time `json:"time"`
	Message  string    `json:"message"`
	Hostname string    `json:"hostname"`
	Severity string    `json:"severity"`
	Program  string    `json:"program"`
}

type pageInfo struct {
	PrevPage string `json:"prevPage"`
	NextPage string `json:"nextPage"`
}

type getLogsResponse struct {
	Logs     []log `json:"logs"`
	pageInfo `json:"pageInfo"`
}

// NewClient creates a new logs client
func NewClient(opts *Options) (*Client, error) {
	// Configure logging based on verbose flag
	shared.SetupLogger(opts.Verbose)

	return &Client{
		httpClient: *http.DefaultClient,
		opts:       opts,
		output:     os.Stdout,
	}, nil
}

func (c *Client) prepareRequest(ctx context.Context, nextPage string) (*http.Request, error) {
	var logsEndpoint string
	var err error
	params := url.Values{}
	if nextPage == "" {
		logsEndpoint, err = url.JoinPath(c.opts.APIURL, "v1/logs")
		if c.opts.follow {
			params.Add("direction", "tail")
		} else {
			params.Add("direction", "forward")
		}

		params.Add("pageSize", "1000")

		if c.opts.group != "" {
			params.Add("group", c.opts.group)
		}
		if c.opts.minTime != "" {
			params.Add("startTime", c.opts.minTime)
		}
		if c.opts.maxTime != "" {
			params.Add("endTime", c.opts.maxTime)
		}

		var filter string
		if c.opts.system != "" {
			filter = fmt.Sprintf(`host:"%s"`, c.opts.system)
		}
		if len(c.opts.args) != 0 {
			if len(filter) == 0 {
				filter = strings.Join(c.opts.args, " ")
			} else {
				filter = filter + " " + strings.Join(c.opts.args, " ")
			}
		}

		if filter != "" {
			params.Add("filter", filter)
		}
	} else {
		u, err := url.Parse(nextPage)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nextPage field: %w", err)
		}

		logsEndpoint, err = url.JoinPath(c.opts.APIURL, u.Path)
		if err != nil {
			return nil, err
		}

		params, err = url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, err
		}

		if c.opts.follow {
			params.Del("endTime")
		}
	}

	if err != nil {
		return nil, err
	}

	logsURL, err := url.Parse(logsEndpoint)
	if err != nil {
		return nil, err
	}

	logsURL.RawQuery = params.Encode()

	slog.Debug("API Request", "method", "GET", "url", logsURL.String())

	request, err := http.NewRequestWithContext(ctx, "GET", logsURL.String(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) printResult(logs []log) error {
	for _, l := range logs {
		l.Time = l.Time.Local()
		if c.opts.json {
			log, err := json.Marshal(l)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(c.output, string(log))
		} else {
			_, _ = fmt.Fprintf(c.output, "%s %s %s %s\n", l.Time.Format("Jan 02 15:04:05"), l.Hostname, l.Program, l.Message)
		}
	}

	return nil
}

func (c *Client) getLogs(ctx context.Context, nextPage string) (*getLogsResponse, error) {
	request, err := c.prepareRequest(ctx, nextPage)
	if err != nil {
		return nil, fmt.Errorf("error while preparing http request to SWO: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error while sending http request to SWO: %w", err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			slog.Error("Could not close https body", "error", err)
		}
	}()

	slog.Debug("Response status", "status_code", response.StatusCode, "status", response.Status)

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading http response body from SWO: %w", err)
	}

	slog.Debug("Response body", "length_bytes", len(content))

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("%w: %d, response body: %s", ErrInvalidAPIResponse, response.StatusCode, string(content))
	}

	if len(content) == 0 {
		return nil, ErrNoContent
	}

	var logs getLogsResponse
	err = json.Unmarshal(content, &logs)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling http response body from SWO: %w", err)
	}

	return &logs, nil
}

// Run executes the logs retrieval and printing process
func (c *Client) Run(ctx context.Context) error {
	var nextPage string

	for {
		logs, err := c.getLogs(ctx, nextPage)
		if err != nil {
			return err
		}

		err = c.printResult(logs.Logs)
		if err != nil {
			return fmt.Errorf("failed to print result: %w", err)
		}

		if c.opts.follow && len(logs.Logs) == 0 {
			time.Sleep(2 * time.Second)
		}

		if logs.NextPage == "" {
			break
		}

		nextPage = logs.NextPage
	}

	return nil
}

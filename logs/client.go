package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/solarwinds/swo-cli/version"
)

type Client struct {
	opts       *Options
	httpClient http.Client
	output     *os.File
}

type Log struct {
	Time     time.Time `json:"time"`
	Message  string    `json:"message"`
	Hostname string    `json:"hostname"`
	Severity string    `json:"severity"`
	Program  string    `json:"program"`
}

type PageInfo struct {
	PrevPage string `json:"prevPage"`
	NextPage string `json:"nextPage"`
}

type LogsData struct {
	Logs     []Log `json:"logs"`
	PageInfo `json:"pageInfo"`
}

func NewClient(opts *Options) (*Client, error) {
	return &Client{
		httpClient: *http.DefaultClient,
		opts:       opts,
		output:     os.Stdout,
	}, nil
}

func (c *Client) prepareRequest(ctx context.Context) (*http.Request, error) {
	logsEndpoint, err := url.JoinPath(c.opts.ApiUrl, "v1/logs")
	if err != nil {
		return nil, err
	}

	logsUrl, err := url.Parse(logsEndpoint)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if c.opts.count != 0 {
		params.Add("pageSize", strconv.Itoa(int(c.opts.count)))
	}
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
		filter = fmt.Sprintf("host:%s", c.opts.system)
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

	logsUrl.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, "GET", logsUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) printResult(logs *LogsData) error {
	if c.opts.json {
		jsonFormat, err := json.Marshal(logs)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(c.output, string(jsonFormat))
		return err
	}

	for i := len(logs.Logs) - 1; i >= 0; i-- {
		l := logs.Logs[i]
		fmt.Fprintf(c.output, "%s %s %s %s\n", l.Time.Format("Jan 02 15:04:05"), l.Hostname, l.Program, l.Message)
	}

	return nil
}

func (c *Client) Run(ctx context.Context) error {
	if c.opts.version {
		fmt.Fprintln(c.output, version.Version)
		return nil
	}

	request, err := c.prepareRequest(ctx)
	if err != nil {
		return fmt.Errorf("error while preparing http request to SWO: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("error while sending http request to SWO: %w", err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			slog.Error("Could not close https body", "error", err)
		}
	}()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error while reading http response body from SWO: %w", err)
	}

	if !(response.StatusCode >= 200 && response.StatusCode < 300) {
		return fmt.Errorf("received %d status code, response body: %s", response.StatusCode, string(content))
	}

	if len(content) == 0 {
		return nil
	}

	var logs LogsData
	err = json.Unmarshal(content, &logs)
	if err != nil {
		return fmt.Errorf("error while unmarshaling http response body from SWO: %w", err)
	}

	return c.printResult(&logs)
}

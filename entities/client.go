// Package entities provides a client for retrieving and managing entities from the SWO API.
package entities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/solarwinds/swo-cli/shared"
)

const (
	// DefaultPageSize for retrieving list of enties etc.
	DefaultPageSize = 100
)

var (
	// ErrInvalidAPIResponse indicates a non-2xx status code was received from the API
	ErrInvalidAPIResponse = errors.New("received non-2xx status code")
	// ErrNoContent indicates an empty response body was received from the API
	ErrNoContent = errors.New("no content")
)

// Client is an entities client
type Client struct {
	opts       *Options
	httpClient http.Client
	output     *os.File
}

// Entity represents an entity from the SWO API
type Entity struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	CreatedTime   string                 `json:"createdTime,omitempty"`
	UpdatedTime   string                 `json:"updatedTime,omitempty"`
	LastSeenTime  string                 `json:"lastSeenTime"`
	InMaintenance bool                   `json:"inMaintenance"`
	Tags          map[string]*string     `json:"tags"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

type pageInfo struct {
	PrevPage string `json:"prevPage"`
	NextPage string `json:"nextPage"`
}

type listEntitiesResponse struct {
	Entities []Entity `json:"entities"`
	pageInfo `json:"pageInfo"`
}

type listTypesResponse struct {
	Types []string `json:"types"`
}

// NewClient creates a new entities client
func NewClient(opts *Options) (*Client, error) {
	// Configure logging based on verbose flag
	shared.SetupLogger(opts.Verbose)

	return &Client{
		httpClient: *http.DefaultClient,
		opts:       opts,
		output:     os.Stdout,
	}, nil
}

func (c *Client) prepareListRequest(ctx context.Context, nextPage string) (*http.Request, error) {
	var entitiesEndpoint string
	var err error
	params := url.Values{}

	if nextPage == "" {
		entitiesEndpoint, err = url.JoinPath(c.opts.APIURL, "v1/entities")
		if err != nil {
			return nil, err
		}

		// Type is required
		params.Add("type", c.opts.Type)

		// Name is optional
		if c.opts.Name != "" {
			params.Add("name", c.opts.Name)
		}

		params.Add("pageSize", strconv.Itoa(DefaultPageSize))
	} else {
		u, err := url.Parse(nextPage)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nextPage field: %w", err)
		}

		entitiesEndpoint, err = url.JoinPath(c.opts.APIURL, u.Path)
		if err != nil {
			return nil, err
		}

		params, err = url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, err
		}
	}

	entitiesURL, err := url.Parse(entitiesEndpoint)
	if err != nil {
		return nil, err
	}

	entitiesURL.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, "GET", entitiesURL.String(), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) prepareGetRequest(ctx context.Context) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.opts.APIURL, "v1/entities", c.opts.ID)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) prepareUpdateRequest(ctx context.Context, entity *Entity) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.opts.APIURL, "v1/entities", c.opts.ID)
	if err != nil {
		return nil, err
	}

	// Update the entity tags with the new ones
	if entity.Tags == nil {
		entity.Tags = make(map[string]*string)
	}
	for key, value := range c.opts.Tags {
		entity.Tags[key] = &value
	}

	jsonData, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "PUT", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) prepareListTypesRequest(ctx context.Context) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.opts.APIURL, "v1/metadata/entities/types")
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.opts.Token))
	request.Header.Add("Accept", "application/json")

	return request, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	slog.Debug("Sending HTTP request", "method", req.Method, "url", req.URL.String())

	response, err := c.httpClient.Do(req)
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

	return content, nil
}

func (c *Client) printEntities(entities []Entity) error {
	for _, entity := range entities {
		if c.opts.JSON {
			jsonData, err := json.Marshal(entity)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(c.output, string(jsonData))
		} else {
			_, _ = fmt.Fprintf(c.output, "ID: %s, Type: %s", entity.ID, entity.Type)
			if entity.Name != "" {
				_, _ = fmt.Fprintf(c.output, ", Name: %s", entity.Name)
			}
			if entity.DisplayName != "" {
				_, _ = fmt.Fprintf(c.output, ", DisplayName: %s", entity.DisplayName)
			}
			_, _ = fmt.Fprintf(c.output, ", InMaintenance: %t", entity.InMaintenance)
			if len(entity.Tags) > 0 {
				_, _ = fmt.Fprintf(c.output, ", Tags: ")
				first := true
				for key, value := range entity.Tags {
					if !first {
						_, _ = fmt.Fprintf(c.output, ", ")
					}
					val := ""
					if value != nil {
						val = *value
					}
					_, _ = fmt.Fprintf(c.output, "%s=%s", key, val)
					first = false
				}
			}
			_, _ = fmt.Fprintln(c.output)
		}
	}
	return nil
}

func (c *Client) printEntity(entity *Entity) error {
	if c.opts.JSON {
		jsonData, err := json.Marshal(entity)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(c.output, string(jsonData))
	} else {
		_, _ = fmt.Fprintf(c.output, "ID: %s\n", entity.ID)
		_, _ = fmt.Fprintf(c.output, "Type: %s\n", entity.Type)
		if entity.Name != "" {
			_, _ = fmt.Fprintf(c.output, "Name: %s\n", entity.Name)
		}
		if entity.DisplayName != "" {
			_, _ = fmt.Fprintf(c.output, "DisplayName: %s\n", entity.DisplayName)
		}
		if entity.CreatedTime != "" {
			_, _ = fmt.Fprintf(c.output, "CreatedTime: %s\n", entity.CreatedTime)
		}
		if entity.UpdatedTime != "" {
			_, _ = fmt.Fprintf(c.output, "UpdatedTime: %s\n", entity.UpdatedTime)
		}
		_, _ = fmt.Fprintf(c.output, "LastSeenTime: %s\n", entity.LastSeenTime)
		_, _ = fmt.Fprintf(c.output, "InMaintenance: %t\n", entity.InMaintenance)

		if len(entity.Tags) > 0 {
			_, _ = fmt.Fprintf(c.output, "Tags:\n")
			for key, value := range entity.Tags {
				val := ""
				if value != nil {
					val = *value
				}
				_, _ = fmt.Fprintf(c.output, "  %s: %s\n", key, val)
			}
		}

		if len(entity.Attributes) > 0 {
			_, _ = fmt.Fprintf(c.output, "Attributes:\n")
			for key, value := range entity.Attributes {
				_, _ = fmt.Fprintf(c.output, "  %s: %v\n", key, value)
			}
		}
	}
	return nil
}

func (c *Client) printTypes(types []string) error {
	if c.opts.JSON {
		jsonData, err := json.Marshal(map[string][]string{"types": types})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(c.output, string(jsonData))
	} else {
		for _, entityType := range types {
			_, _ = fmt.Fprintln(c.output, entityType)
		}
	}
	return nil
}

// ListEntities retrieves and displays entities
func (c *Client) ListEntities(ctx context.Context) error {
	var nextPage string

	for {
		request, err := c.prepareListRequest(ctx, nextPage)
		if err != nil {
			return fmt.Errorf("error while preparing http request to SWO: %w", err)
		}

		content, err := c.doRequest(request)
		if err != nil {
			return err
		}

		if len(content) == 0 {
			return ErrNoContent
		}

		var response listEntitiesResponse
		err = json.Unmarshal(content, &response)
		if err != nil {
			return fmt.Errorf("error while unmarshaling http response body from SWO: %w", err)
		}

		err = c.printEntities(response.Entities)
		if err != nil {
			return fmt.Errorf("failed to print entities: %w", err)
		}

		if response.NextPage == "" {
			break
		}

		nextPage = response.NextPage
	}

	return nil
}

// GetEntity retrieves and displays a single entity by ID
func (c *Client) GetEntity(ctx context.Context) error {
	request, err := c.prepareGetRequest(ctx)
	if err != nil {
		return fmt.Errorf("error while preparing http request to SWO: %w", err)
	}

	content, err := c.doRequest(request)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return ErrNoContent
	}

	var entity Entity
	err = json.Unmarshal(content, &entity)
	if err != nil {
		return fmt.Errorf("error while unmarshaling http response body from SWO: %w", err)
	}

	return c.printEntity(&entity)
}

// UpdateEntity updates entity tags
func (c *Client) UpdateEntity(ctx context.Context) error {
	// First, get the current entity
	getRequest, err := c.prepareGetRequest(ctx)
	if err != nil {
		return fmt.Errorf("error while preparing get request to SWO: %w", err)
	}

	content, err := c.doRequest(getRequest)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return ErrNoContent
	}

	var entity Entity
	err = json.Unmarshal(content, &entity)
	if err != nil {
		return fmt.Errorf("error while unmarshaling entity from SWO: %w", err)
	}

	// Now update with the new tags
	updateRequest, err := c.prepareUpdateRequest(ctx, &entity)
	if err != nil {
		return fmt.Errorf("error while preparing update request to SWO: %w", err)
	}

	// Use doRequest for consistency - empty content is acceptable for updates
	_, err = c.doRequest(updateRequest)
	if err != nil {
		return err
	}

	if !c.opts.JSON {
		_, _ = fmt.Fprintf(c.output, "Entity %s updated successfully\n", c.opts.ID)
	} else {
		_, _ = fmt.Fprintf(c.output, `{"status":"success","id":"%s"}\n`, c.opts.ID)
	}

	return nil
}

// ListTypes retrieves and displays all available entity types
func (c *Client) ListTypes(ctx context.Context) error {
	request, err := c.prepareListTypesRequest(ctx)
	if err != nil {
		return fmt.Errorf("error while preparing http request to SWO: %w", err)
	}

	content, err := c.doRequest(request)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return ErrNoContent
	}

	var response listTypesResponse
	err = json.Unmarshal(content, &response)
	if err != nil {
		return fmt.Errorf("error while unmarshaling http response body from SWO: %w", err)
	}

	return c.printTypes(response.Types)
}

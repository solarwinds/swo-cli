package entities

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/solarwinds/swo-cli/shared"
	"github.com/stretchr/testify/require"
)

var (
	configFile   = filepath.Join(os.TempDir(), "config-file.yaml")
	testEntities = []Entity{
		{
			ID:            "e-1234567890",
			Type:          "SyslogHost",
			Name:          "test-host-1",
			DisplayName:   "Test Host 1",
			CreatedTime:   "2024-01-01T00:00:00Z",
			UpdatedTime:   "2024-01-02T00:00:00Z",
			LastSeenTime:  "2024-01-03T00:00:00Z",
			InMaintenance: false,
			Tags: map[string]*string{
				"environment": stringPtr("production"),
				"team":        stringPtr("backend"),
			},
			Attributes: map[string]interface{}{
				"hostname": "test-host-1.example.com",
				"os":       "linux",
			},
		},
		{
			ID:            "e-9876543210",
			Type:          "Service",
			Name:          "test-service-1",
			DisplayName:   "Test Service 1",
			CreatedTime:   "2024-01-01T00:00:00Z",
			UpdatedTime:   "2024-01-02T00:00:00Z",
			LastSeenTime:  "2024-01-03T00:00:00Z",
			InMaintenance: true,
			Tags: map[string]*string{
				"environment": stringPtr("staging"),
				"version":     stringPtr("1.2.3"),
			},
		},
	}
	testTypes = []string{"Service", "ServiceInstance", "KubernetesCluster", "SyslogHost"}
)

func stringPtr(s string) *string {
	return &s
}

func TestNewClient(t *testing.T) {
	opts := NewOptions()
	opts.Token = "test-token"
	opts.APIURL = "https://api.example.com"

	client, err := NewClient(opts)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, opts, client.opts)
}

func TestOptions_ParseTags(t *testing.T) {
	testCases := []struct {
		name        string
		tagStrings  []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:       "valid tags",
			tagStrings: []string{"env=production", "team=backend", "version=1.2.3"},
			expected: map[string]string{
				"env":     "production",
				"team":    "backend",
				"version": "1.2.3",
			},
			expectError: false,
		},
		{
			name:       "tags with spaces",
			tagStrings: []string{"env = production ", " team = backend"},
			expected: map[string]string{
				"env":  "production",
				"team": "backend",
			},
			expectError: false,
		},
		{
			name:        "invalid tag format",
			tagStrings:  []string{"invalid-tag"},
			expectError: true,
		},
		{
			name:        "empty key",
			tagStrings:  []string{"=value"},
			expectError: true,
		},
		{
			name:       "empty value allowed",
			tagStrings: []string{"key="},
			expected: map[string]string{
				"key": "",
			},
			expectError: false,
		},
		{
			name:       "tag with equals in value",
			tagStrings: []string{"config=key=value"},
			expected: map[string]string{
				"config": "key=value",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := NewOptions()
			err := opts.ParseTags(tc.tagStrings)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, opts.Tags)
			}
		})
	}
}

func TestOptions_Validate(t *testing.T) {
	t.Run("ValidateForGet", func(t *testing.T) {
		// Valid case
		opts := NewOptions()
		opts.ID = "e-1234567890"
		require.NoError(t, opts.ValidateForGet())

		// Invalid case - empty ID
		opts.ID = ""
		require.Error(t, opts.ValidateForGet())

		// Invalid case - whitespace only ID
		opts.ID = "   "
		require.Error(t, opts.ValidateForGet())
	})

	t.Run("ValidateForList", func(t *testing.T) {
		// Valid case
		opts := NewOptions()
		opts.Type = "Service"
		require.NoError(t, opts.ValidateForList())

		// Invalid case - empty type
		opts.Type = ""
		require.Error(t, opts.ValidateForList())

		// Invalid case - whitespace only type
		opts.Type = "   "
		require.Error(t, opts.ValidateForList())
	})

	t.Run("ValidateForUpdate", func(t *testing.T) {
		// Valid case
		opts := NewOptions()
		opts.ID = "e-1234567890"
		opts.Tags = map[string]string{"env": "production"}
		require.NoError(t, opts.ValidateForUpdate())

		// Invalid case - empty ID
		opts.ID = ""
		require.Error(t, opts.ValidateForUpdate())

		// Invalid case - no tags
		opts.ID = "e-1234567890"
		opts.Tags = map[string]string{}
		require.Error(t, opts.ValidateForUpdate())
	})
}

func TestPrepareListRequest(t *testing.T) {
	testCases := []struct {
		name           string
		options        *Options
		nextPage       string
		expectedParams map[string]string
	}{
		{
			name: "basic list request",
			options: &Options{
				Type:     "Service",
				PageSize: 100,
				BaseOptions: shared.BaseOptions{
					Token:  "test-token",
					APIURL: "https://api.example.com"},
			},
			nextPage: "",
			expectedParams: map[string]string{
				"type":     "Service",
				"pageSize": "100",
			},
		},
		{
			name: "list with name filter",
			options: &Options{
				Type:     "Service",
				Name:     "test-service",
				PageSize: 50,
				BaseOptions: shared.BaseOptions{
					Token:  "test-token",
					APIURL: "https://api.example.com"},
			},
			nextPage: "",
			expectedParams: map[string]string{
				"type":     "Service",
				"name":     "test-service",
				"pageSize": "50",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.options)
			require.NoError(t, err)

			request, err := client.prepareListRequest(context.Background(), tc.nextPage)
			require.NoError(t, err)

			values := request.URL.Query()
			for k, v := range tc.expectedParams {
				require.Equal(t, v, values.Get(k))
			}

			// Check headers
			require.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
			require.Equal(t, "application/json", request.Header.Get("Accept"))
		})
	}
}

func TestPrepareGetRequest(t *testing.T) {
	opts := &Options{
		ID: "e-1234567890",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: "https://api.example.com"},
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	request, err := client.prepareGetRequest(context.Background())
	require.NoError(t, err)

	expectedURL := "https://api.example.com/v1/entities/e-1234567890"
	require.Equal(t, expectedURL, request.URL.String())
	require.Equal(t, "GET", request.Method)
	require.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
	require.Equal(t, "application/json", request.Header.Get("Accept"))
}

func TestPrepareUpdateRequest(t *testing.T) {
	opts := &Options{
		ID: "e-1234567890",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: "https://api.example.com"},
		Tags: map[string]string{"env": "production", "team": "backend"},
	}

	entity := &Entity{
		ID:   "e-1234567890",
		Type: "Service",
		Tags: map[string]*string{
			"existing": stringPtr("value"),
		},
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	request, err := client.prepareUpdateRequest(context.Background(), entity)
	require.NoError(t, err)

	expectedURL := "https://api.example.com/v1/entities/e-1234567890"
	require.Equal(t, expectedURL, request.URL.String())
	require.Equal(t, "PUT", request.Method)
	require.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
	require.Equal(t, "application/json", request.Header.Get("Content-Type"))
	require.Equal(t, "application/json", request.Header.Get("Accept"))

	// Check that entity tags were updated
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var updatedEntity Entity
	err = json.Unmarshal(body, &updatedEntity)
	require.NoError(t, err)

	require.Equal(t, "production", *updatedEntity.Tags["env"])
	require.Equal(t, "backend", *updatedEntity.Tags["team"])
	require.Equal(t, "value", *updatedEntity.Tags["existing"])
}

func TestPrepareListTypesRequest(t *testing.T) {
	opts := &Options{
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: "https://api.example.com"},
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	request, err := client.prepareListTypesRequest(context.Background())
	require.NoError(t, err)

	expectedURL := "https://api.example.com/v1/metadata/entities/types"
	require.Equal(t, expectedURL, request.URL.String())
	require.Equal(t, "GET", request.Method)
	require.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
	require.Equal(t, "application/json", request.Header.Get("Accept"))
}

func TestListEntities(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/entities", r.URL.Path)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := listEntitiesResponse{
			Entities: testEntities,
			pageInfo: pageInfo{NextPage: ""},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	opts := &Options{
		Type: "Service",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
		JSON: false,
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	// Capture output
	tempFile, err := os.CreateTemp("", "test-output")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	client.output = tempFile

	err = client.ListEntities(context.Background())
	require.NoError(t, err)

	// Read and verify output
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)
	output, err := io.ReadAll(tempFile)
	require.NoError(t, err)

	outputStr := string(output)
	require.Contains(t, outputStr, "e-1234567890")
	require.Contains(t, outputStr, "SyslogHost")
	require.Contains(t, outputStr, "test-host-1")
}

func TestGetEntity(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/entities/e-1234567890", r.URL.Path)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testEntities[0])
	}))
	defer server.Close()

	opts := &Options{
		ID: "e-1234567890",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
		JSON: false,
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	// Capture output
	tempFile, err := os.CreateTemp("", "test-output")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	client.output = tempFile

	err = client.GetEntity(context.Background())
	require.NoError(t, err)

	// Read and verify output
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)
	output, err := io.ReadAll(tempFile)
	require.NoError(t, err)

	outputStr := string(output)
	require.Contains(t, outputStr, "ID: e-1234567890")
	require.Contains(t, outputStr, "Type: SyslogHost")
	require.Contains(t, outputStr, "Name: test-host-1")
	require.Contains(t, outputStr, "InMaintenance: false")
}

func TestUpdateEntity(t *testing.T) {
	// Track requests
	var getRequest, putRequest *http.Request

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getRequest = r
			require.Equal(t, "/v1/entities/e-1234567890", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(testEntities[0])
		} else if r.Method == "PUT" {
			putRequest = r
			require.Equal(t, "/v1/entities/e-1234567890", r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	opts := &Options{
		ID: "e-1234567890",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
		Tags: map[string]string{"newTag": "newValue"},
		JSON: false,
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	// Capture output
	tempFile, err := os.CreateTemp("", "test-output")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	client.output = tempFile

	err = client.UpdateEntity(context.Background())
	require.NoError(t, err)

	// Verify both requests were made
	require.NotNil(t, getRequest)
	require.NotNil(t, putRequest)

	// Read and verify output
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)
	output, err := io.ReadAll(tempFile)
	require.NoError(t, err)

	outputStr := string(output)
	require.Contains(t, outputStr, "Entity e-1234567890 updated successfully")
}

func TestListTypes(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/metadata/entities/types", r.URL.Path)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := listTypesResponse{Types: testTypes}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	opts := &Options{
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
		JSON: false,
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	// Capture output
	tempFile, err := os.CreateTemp("", "test-output")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	client.output = tempFile

	err = client.ListTypes(context.Background())
	require.NoError(t, err)

	// Read and verify output
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)
	output, err := io.ReadAll(tempFile)
	require.NoError(t, err)

	outputStr := string(output)
	for _, entityType := range testTypes {
		require.Contains(t, outputStr, entityType)
	}
}

func TestJSONOutput(t *testing.T) {
	// Test JSON output for ListEntities
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := listEntitiesResponse{
			Entities: testEntities[:1], // Just one entity for simpler testing
			pageInfo: pageInfo{NextPage: ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	opts := &Options{
		Type: "Service",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
		JSON: true,
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	// Capture output
	tempFile, err := os.CreateTemp("", "test-output")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	client.output = tempFile

	err = client.ListEntities(context.Background())
	require.NoError(t, err)

	// Read and verify output is valid JSON
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)
	output, err := io.ReadAll(tempFile)
	require.NoError(t, err)

	// Should be valid JSON
	var entity Entity
	trimmed := strings.TrimSpace(string(output))
	err = json.Unmarshal([]byte(trimmed), &entity)
	require.NoError(t, err)
	require.Equal(t, "e-1234567890", entity.ID)
}

func TestErrorHandling(t *testing.T) {
	// Test API error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	opts := &Options{
		Type: "Service",
		BaseOptions: shared.BaseOptions{
			Token:  "test-token",
			APIURL: server.URL},
	}

	client, err := NewClient(opts)
	require.NoError(t, err)

	err = client.ListEntities(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

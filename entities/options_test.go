package entities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	opts := NewOptions()
	require.NotNil(t, opts)
	require.NotNil(t, opts.Tags)
	require.Empty(t, opts.Tags)
}

func TestValidation(t *testing.T) {
	t.Run("ValidateForGet", func(t *testing.T) {
		tests := []struct {
			name        string
			id          string
			expectError bool
		}{
			{"valid ID", "e-1234567890", false},
			{"empty ID", "", true},
			{"whitespace ID", "   ", true},
			{"tab and space ID", "\t  \n", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opts := NewOptions()
				opts.ID = tt.id
				err := opts.ValidateForGet()
				if tt.expectError {
					require.Error(t, err)
					require.Equal(t, errMissingEntityID, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("ValidateForList", func(t *testing.T) {
		tests := []struct {
			name        string
			entityType  string
			expectError bool
		}{
			{"valid type", "Service", false},
			{"empty type", "", true},
			{"whitespace type", "   ", true},
			{"tab and space type", "\t  \n", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opts := NewOptions()
				opts.Type = tt.entityType
				err := opts.ValidateForList()
				if tt.expectError {
					require.Error(t, err)
					require.Equal(t, errMissingEntityType, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("ValidateForUpdate", func(t *testing.T) {
		tests := []struct {
			name        string
			id          string
			tags        map[string]string
			expectError bool
			expectedErr error
		}{
			{
				name:        "valid update",
				id:          "e-1234567890",
				tags:        map[string]string{"env": "production"},
				expectError: false,
			},
			{
				name:        "empty ID",
				id:          "",
				tags:        map[string]string{"env": "production"},
				expectError: true,
				expectedErr: errMissingEntityID,
			},
			{
				name:        "whitespace ID",
				id:          "   ",
				tags:        map[string]string{"env": "production"},
				expectError: true,
				expectedErr: errMissingEntityID,
			},
			{
				name:        "no tags",
				id:          "e-1234567890",
				tags:        map[string]string{},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opts := NewOptions()
				opts.ID = tt.id
				opts.Tags = tt.tags
				err := opts.ValidateForUpdate()
				if tt.expectError {
					require.Error(t, err)
					if tt.expectedErr != nil {
						require.Equal(t, tt.expectedErr, err)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestParseTagsEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		tagStrings  []string
		expected    map[string]string
		expectError bool
	}{
		{
			name:        "empty slice",
			tagStrings:  []string{},
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name:       "multiple equals signs",
			tagStrings: []string{"config=app=test=value"},
			expected: map[string]string{
				"config": "app=test=value",
			},
			expectError: false,
		},
		{
			name:        "missing equals",
			tagStrings:  []string{"invalid"},
			expectError: true,
		},
		{
			name:        "multiple invalid tags",
			tagStrings:  []string{"valid=ok", "invalid", "also=good"},
			expectError: true,
		},
		{
			name:       "unicode characters",
			tagStrings: []string{"环境=生产", "tëam=bäckend"},
			expected: map[string]string{
				"环境":   "生产",
				"tëam": "bäckend",
			},
			expectError: false,
		},
		{
			name:       "special characters",
			tagStrings: []string{"host-name=web-server.example.com", "port=8080"},
			expected: map[string]string{
				"host-name": "web-server.example.com",
				"port":      "8080",
			},
			expectError: false,
		},
		{
			name:        "only whitespace key",
			tagStrings:  []string{"  =value"},
			expectError: true,
		},
		{
			name: "very long key and value",
			tagStrings: []string{
				"very-long-key-with-many-characters-that-should-still-work=very-long-value-with-many-characters-that-should-also-work-without-any-issues",
			},
			expected: map[string]string{
				"very-long-key-with-many-characters-that-should-still-work": "very-long-value-with-many-characters-that-should-also-work-without-any-issues",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			err := opts.ParseTags(tt.tagStrings)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, opts.Tags)
			}
		})
	}
}

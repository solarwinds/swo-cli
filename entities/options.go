package entities

import (
	"errors"
	"fmt"
	"strings"

	"github.com/solarwinds/swo-cli/shared"
)

var (
	errMissingEntityID   = errors.New("entity ID is required")
	errMissingEntityType = errors.New("entity type is required")
	errInvalidTag        = errors.New("invalid tag format, expected key=value")
	errAtLeastOneTag     = errors.New("at least one tag is required for update")
)

// Options represents the command line options for the entities command
type Options struct {
	shared.BaseOptions // Embedded base options (Verbose, Token, APIURL)
	ID                 string
	Type               string
	Name               string
	PageSize           int
	Tags               map[string]string
	JSON               bool
}

// NewOptions creates a new Options instance
func NewOptions() *Options {
	return &Options{
		Tags: make(map[string]string),
	}
}

// ParseTags parses tag strings in key=value format into a map
func (o *Options) ParseTags(tagStrings []string) error {
	o.Tags = make(map[string]string)
	for _, tagStr := range tagStrings {
		parts := strings.SplitN(tagStr, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("%w: %s", errInvalidTag, tagStr)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("%w: empty key in %s", errInvalidTag, tagStr)
		}
		o.Tags[key] = value
	}
	return nil
}

// ValidateForGet validates the options for get operations
func (o *Options) ValidateForGet() error {
	if strings.TrimSpace(o.ID) == "" {
		return errMissingEntityID
	}
	return nil
}

// ValidateForList validates options for list operation
func (o *Options) ValidateForList() error {
	if strings.TrimSpace(o.Type) == "" {
		return errMissingEntityType
	}
	return nil
}

// ValidateForUpdate validates options for update operation
func (o *Options) ValidateForUpdate() error {
	if strings.TrimSpace(o.ID) == "" {
		return errMissingEntityID
	}
	if len(o.Tags) == 0 {
		return errAtLeastOneTag
	}
	return nil
}

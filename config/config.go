// Package config implements configuration loading and parsing
package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFile is the default path to the config file
	DefaultConfigFile = "~/.swo-cli.yml"
	// DefaultAPIURL is the default URL of the SWO API
	DefaultAPIURL = "https://api.na-01.cloud.solarwinds.com"
	// APIURLContextKey is the context key for the API URL
	APIURLContextKey = "api-url"
	// TokenContextKey is the context key for the API token
	TokenContextKey = "token"
)

var (
	errMissingToken  = errors.New("failed to find token")
	errMissingAPIURL = errors.New("failed to find API URL")
)

// Config represents the base configuration for the SWO CLI
type Config struct {
	APIURL string `yaml:"api-url"`
	Token  string `yaml:"token"`
}

// Init initializes the configuration by loading from the specified config file,
// environment variables, and command line flags.
// Precedence: CLI flags, environment, config file
func Init(configPath string, apiURL string, apiToken string) (*Config, error) {
	config := &Config{
		APIURL: apiURL,
		Token:  apiToken,
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	localConfig := filepath.Join(cwd, ".swo-cli.yaml")
	if _, err := os.Stat(localConfig); err == nil {
		configPath = localConfig
	} else if strings.HasPrefix(configPath, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error while resolving current user to read configuration file: %w", err)
		}

		configPath = filepath.Join(usr.HomeDir, configPath[2:])
	}
	configPath = filepath.Clean(configPath)

	if content, err := os.ReadFile(configPath); err == nil {
		err = yaml.Unmarshal(content, config)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshaling %s config file: %w", configPath, err)
		}
	}

	if token := os.Getenv("SWO_API_TOKEN"); token != "" {
		config.Token = token
	}

	if url := os.Getenv("SWO_API_URL"); url != "" {
		config.APIURL = url
	}

	if config.Token == "" {
		return nil, errMissingToken
	}

	if config.APIURL == "" {
		return nil, errMissingAPIURL
	}

	return config, nil
}

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
	TokenContextKey = "api-token"
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
	// initialize values from CMD line
	config := &Config{
		APIURL: strings.TrimSpace(apiURL),
		Token:  strings.TrimSpace(apiToken),
	}

	// if empty, try ENV variables
	if config.APIURL == "" {
		config.APIURL = strings.TrimSpace(os.Getenv("SWO_API_URL"))
	}
	if config.Token == "" {
		config.Token = strings.TrimSpace(os.Getenv("SWO_API_TOKEN"))
	}

	if config.APIURL == "" || config.Token == "" {
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

		configFromFile := &Config{}
		if content, err := os.ReadFile(configPath); err == nil {
			err = yaml.Unmarshal(content, configFromFile)
			if err != nil {
				return nil, fmt.Errorf("error while unmarshaling %s config file: %w", configPath, err)
			}
		}

		if config.APIURL == "" {
			config.APIURL = strings.TrimSpace(configFromFile.APIURL)
		}

		if config.Token == "" {
			config.Token = strings.TrimSpace(configFromFile.Token)
		}

	}

	if config.APIURL == "" {
		config.APIURL = DefaultAPIURL
	}

	if config.Token == "" {
		return nil, errMissingToken
	}

	return config, nil
}

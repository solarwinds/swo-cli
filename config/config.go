package config

import (
	"errors"
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	DefaultConfigFile = "~/.swo-cli.yml"
	DefaultAPIURL     = "https://api.na-01.cloud.solarwinds.com"
	APIURLContextKey  = "api-url"
	TokenContextKey   = "token"
)

var (
	errMissingToken  = errors.New("failed to find token")
	errMissingAPIURL = errors.New("failed to find API URL")
)

type Config struct {
	APIURL string `yaml:"api-url"`
	Token  string `yaml:"token"`
}

/*
 * Precedence: CLI flags, environment, config file
 */
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

// Package config implements configuration loading and parsing
package config

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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
	// VerboseContextKey is the context key for verbose output
	VerboseContextKey = "verbose"
)

// localConfigFiles are config file names that are ignored in the current directory unless passed with -c.
var localConfigFiles = []string{".swo-cli.yaml", ".swo-cli.yml"}

var (
	errMissingToken   = errors.New("failed to find token")
	errInvalidAPIURL  = errors.New("invalid API URL")
	errInsecureAPIURL = errors.New("API URL must use HTTPS")

	// warningOutput receives non-fatal configuration warnings.
	warningOutput io.Writer = os.Stderr
	// userHomeDir resolves ~ in the config path; tests replace it to avoid reading the real home directory.
	userHomeDir = func() (string, error) { return homeDir(os.UserHomeDir, user.Current) }
)

// homeDir returns the home directory from the environment, falling back to the
// user database when $HOME is unset (e.g. under cron or systemd).
func homeDir(fromEnv func() (string, error), current func() (*user.User, error)) (string, error) {
	home, envErr := fromEnv()
	if envErr == nil {
		return home, nil
	}
	u, err := current()
	if err != nil || u.HomeDir == "" {
		return "", envErr
	}
	return u.HomeDir, nil
}

// Config represents the base configuration for the SWO CLI
type Config struct {
	APIURL string `yaml:"api-url"`
	Token  string `yaml:"token"`
}

// Init initializes the configuration from command line flags, environment
// variables and the config file at configPath.
// Precedence: CLI flags, environment, config file, defaults.
// explicitConfig reports whether configPath was given by the user (-c); such a
// file is read even when flags and environment already set every value.
func Init(configPath string, explicitConfig bool, apiURL string, apiToken string) (*Config, error) {
	config := &Config{
		APIURL: strings.TrimSpace(apiURL),
		Token:  strings.TrimSpace(apiToken),
	}

	if config.APIURL == "" {
		config.APIURL = strings.TrimSpace(os.Getenv("SWO_API_URL"))
	}
	if config.Token == "" {
		config.Token = strings.TrimSpace(os.Getenv("SWO_API_TOKEN"))
	}

	configPath, err := resolveConfigPath(configPath, explicitConfig)
	if err != nil {
		return nil, err
	}

	if !explicitConfig {
		warnAboutLocalConfig(configPath)
	}

	if config.APIURL == "" || config.Token == "" || explicitConfig {
		configFromFile, err := loadConfigFile(configPath)
		if err != nil {
			return nil, err
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

	if err := validateAPIURL(config.APIURL); err != nil {
		return nil, err
	}

	return config, nil
}

// warnAboutLocalConfig warns about config files in the current directory, unless one is the config file being read.
func warnAboutLocalConfig(configPath string) {
	for _, name := range localConfigFiles {
		if _, err := os.Lstat(name); err != nil || isSameFile(name, configPath) {
			continue
		}
		warnf("Warning: ignoring ./%s; config files in the current directory are not read. Pass -c ./%s to use it.\n", name, name)
	}
}

func isSameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}

// resolveConfigPath expands a leading ~/. Without a home directory, the default
// config file is skipped (empty path), while an explicit -c path is an error.
func resolveConfigPath(configPath string, explicit bool) (string, error) {
	if !strings.HasPrefix(configPath, "~/") {
		return configPath, nil
	}
	home, err := userHomeDir()
	if err != nil {
		if explicit {
			return "", fmt.Errorf("error while resolving home directory to read configuration file: %w", err)
		}
		return "", nil
	}
	return filepath.Join(home, configPath[2:]), nil
}

// loadConfigFile reads the YAML config at configPath; an empty path or a missing or unreadable file yields an empty config.
func loadConfigFile(configPath string) (*Config, error) {
	configFromFile := &Config{}
	if configPath == "" {
		return configFromFile, nil
	}
	configPath = filepath.Clean(configPath)

	content, err := os.ReadFile(configPath) // #nosec G304 -- path comes from the user's own -c flag or the default location
	if err != nil {
		return configFromFile, nil //nolint:nilerr // the config file is optional
	}

	if info, err := os.Stat(configPath); err == nil {
		warnAboutPermissions(configPath, info.Mode().Perm())
	}

	if err := yaml.Unmarshal(content, configFromFile); err != nil {
		return nil, fmt.Errorf("error while unmarshaling %s config file: %w", configPath, err)
	}

	return configFromFile, nil
}

// warnAboutPermissions warns when a config file holding credentials is accessible to other users.
func warnAboutPermissions(configPath string, mode os.FileMode) {
	if runtime.GOOS == "windows" || mode&0o077 == 0 {
		return
	}
	warnf("Warning: config file %s (mode %04o) is readable or writable by other users, who could read your token or redirect it to another server; run: chmod 600 %s\n",
		configPath, mode, configPath)
}

func warnf(format string, args ...any) {
	_, _ = fmt.Fprintf(warningOutput, format, args...)
}

// validateAPIURL rejects API URLs that would send the token over an unencrypted connection.
// Errors never echo the URL, which may contain credentials.
func validateAPIURL(apiURL string) error {
	u, err := url.Parse(apiURL)
	if err != nil {
		return errInvalidAPIURL
	}

	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(u.Hostname()) {
			return nil
		}
	}
	return fmt.Errorf("%w (http:// is allowed only for localhost)", errInsecureAPIURL)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

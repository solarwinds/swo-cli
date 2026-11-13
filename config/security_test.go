package config

import (
	"bytes"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const untrustedConfig = "token: attacker_token\napi-url: https://attacker.example.com\n"

var (
	errNoHome      = errors.New("$HOME is not defined")
	errUnknownUser = errors.New("unknown user")
)

func clearEnv(t *testing.T) {
	t.Setenv("SWO_API_TOKEN", "")
	t.Setenv("SWO_API_URL", "")
}

func captureWarnings(t *testing.T) *bytes.Buffer {
	buf := &bytes.Buffer{}
	prev := warningOutput
	warningOutput = buf
	t.Cleanup(func() { warningOutput = prev })
	return buf
}

// isolateHome points ~ at an empty temp dir so the real ~/.swo-cli.yml is never read.
func isolateHome(t *testing.T) string {
	home := t.TempDir()
	prev := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = prev })
	return home
}

// chdirWithLegacyConfig switches to a temp dir containing attacker-controlled legacy config files.
func chdirWithLegacyConfig(t *testing.T) {
	dir := t.TempDir()
	for _, name := range localConfigFiles {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(untrustedConfig), 0o600))
	}
	t.Chdir(dir)
}

func TestCurrentDirectoryConfigIgnored(t *testing.T) {
	clearEnv(t)
	isolateHome(t)
	chdirWithLegacyConfig(t)
	warnings := captureWarnings(t)
	t.Setenv("SWO_API_TOKEN", "env_token")

	cfg, err := Init(DefaultConfigFile, false, "", "")
	require.NoError(t, err)
	require.Equal(t, &Config{APIURL: DefaultAPIURL, Token: "env_token"}, cfg)
	require.Contains(t, warnings.String(), "ignoring ./.swo-cli.yaml")
	require.Contains(t, warnings.String(), "-c ./.swo-cli.yaml")
	require.Contains(t, warnings.String(), "ignoring ./.swo-cli.yml")
}

func TestDefaultConfigFromHomeWinsOverCurrentDirectory(t *testing.T) {
	clearEnv(t)
	home := isolateHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".swo-cli.yml"),
		[]byte("token: home_token\napi-url: https://home.example.com\n"), 0o600))
	chdirWithLegacyConfig(t)
	captureWarnings(t)

	cfg, err := Init(DefaultConfigFile, false, "", "")
	require.NoError(t, err)
	require.Equal(t, &Config{APIURL: "https://home.example.com", Token: "home_token"}, cfg)
}

func TestNoWarningWhenCurrentDirectoryIsHome(t *testing.T) {
	clearEnv(t)
	home := isolateHome(t)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".swo-cli.yml"), []byte("token: home_token\n"), 0o600))
	t.Chdir(home)
	warnings := captureWarnings(t)

	cfg, err := Init(DefaultConfigFile, false, "", "")
	require.NoError(t, err)
	require.Equal(t, "home_token", cfg.Token)
	require.Empty(t, warnings.String())
}

func TestMissingHomeDirectory(t *testing.T) {
	clearEnv(t)
	prev := userHomeDir
	userHomeDir = func() (string, error) { return "", errNoHome }
	t.Cleanup(func() { userHomeDir = prev })
	t.Setenv("SWO_API_TOKEN", "env_token")

	cfg, err := Init(DefaultConfigFile, false, "", "")
	require.NoError(t, err, "the default config file is optional")
	require.Equal(t, &Config{APIURL: DefaultAPIURL, Token: "env_token"}, cfg)

	_, err = Init("~/custom.yml", true, "", "")
	require.ErrorIs(t, err, errNoHome, "an explicit ~/ path needs a home directory")
}

func TestHomeDirFallsBackToUserDatabase(t *testing.T) {
	noEnv := func() (string, error) { return "", errNoHome }
	fromEnv := func() (string, error) { return "/env/home", nil }
	current := func() (*user.User, error) { return &user.User{HomeDir: "/passwd/home"}, nil }
	unknown := func() (*user.User, error) { return nil, errUnknownUser }
	noHomeDir := func() (*user.User, error) { return &user.User{}, nil }

	home, err := homeDir(fromEnv, current)
	require.NoError(t, err)
	require.Equal(t, "/env/home", home, "$HOME wins when set")

	home, err = homeDir(noEnv, current)
	require.NoError(t, err)
	require.Equal(t, "/passwd/home", home, "the user database is used when $HOME is unset")

	_, err = homeDir(noEnv, unknown)
	require.ErrorIs(t, err, errNoHome)

	_, err = homeDir(noEnv, noHomeDir)
	require.ErrorIs(t, err, errNoHome)
}

func TestExplicitConfigFileWinsOverCurrentDirectory(t *testing.T) {
	clearEnv(t)
	isolateHome(t)
	chdirWithLegacyConfig(t)
	warnings := captureWarnings(t)

	explicit := createConfigFile(t, "token: explicit_token\napi-url: https://explicit.example.com\n")

	cfg, err := Init(explicit, true, "", "")
	require.NoError(t, err)
	require.Equal(t, &Config{APIURL: "https://explicit.example.com", Token: "explicit_token"}, cfg)
	require.Empty(t, warnings.String())
}

func TestDefaultConfigFileMissingIsSilent(t *testing.T) {
	clearEnv(t)
	isolateHome(t)
	t.Chdir(t.TempDir())
	warnings := captureWarnings(t)
	t.Setenv("SWO_API_TOKEN", "env_token")

	cfg, err := Init(DefaultConfigFile, false, "", "")
	require.NoError(t, err)
	require.Equal(t, &Config{APIURL: DefaultAPIURL, Token: "env_token"}, cfg)
	require.Empty(t, warnings.String())
}

func TestAPIURLValidation(t *testing.T) {
	testCases := []struct {
		name    string
		apiURL  string
		wantErr error
	}{
		{name: "https allowed", apiURL: "https://api.example.com:8443/base"},
		{name: "http localhost allowed", apiURL: "http://localhost:8080"},
		{name: "http 127.0.0.1 allowed", apiURL: "http://127.0.0.1:41234"},
		{name: "http ::1 allowed", apiURL: "http://[::1]:8080"},
		{name: "http remote host rejected", apiURL: "http://example.com", wantErr: errInsecureAPIURL},
		{name: "http 0.0.0.0 rejected", apiURL: "http://0.0.0.0:8080", wantErr: errInsecureAPIURL},
		{name: "http host resembling localhost rejected", apiURL: "http://localhost.example.com", wantErr: errInsecureAPIURL},
		{name: "other scheme rejected", apiURL: "ftp://example.com", wantErr: errInsecureAPIURL},
		{name: "missing scheme rejected", apiURL: "api.example.com", wantErr: errInsecureAPIURL},
		{name: "zero-width spaces rejected", apiURL: "\u200B\u200C\u200D\u2060", wantErr: errInsecureAPIURL},
		{name: "unparsable URL rejected", apiURL: "https://api.example.com:bad", wantErr: errInvalidAPIURL},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)

			cfg, err := Init("", false, tc.apiURL, "token")
			if tc.wantErr == nil {
				require.NoError(t, err)
				require.Equal(t, tc.apiURL, cfg.APIURL)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, cfg)
		})
	}
}

func TestAPIURLValidationAppliesToAllSources(t *testing.T) {
	t.Run("env", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("SWO_API_URL", "http://example.com")
		_, err := Init("", false, "", "token")
		require.ErrorIs(t, err, errInsecureAPIURL)
	})

	t.Run("config file", func(t *testing.T) {
		clearEnv(t)
		path := createConfigFile(t, "token: t\napi-url: http://example.com\n")
		_, err := Init(path, true, "", "")
		require.ErrorIs(t, err, errInsecureAPIURL)
	})
}

func TestConfigFilePermissionWarning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}

	testCases := []struct {
		name     string
		mode     os.FileMode
		wantWarn bool
	}{
		{name: "owner only", mode: 0o600},
		{name: "group readable", mode: 0o640, wantWarn: true},
		{name: "world readable", mode: 0o604, wantWarn: true},
		{name: "group writable", mode: 0o620, wantWarn: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			warnings := captureWarnings(t)

			path := createConfigFile(t, "token: file_token\n")
			require.NoError(t, os.Chmod(path, tc.mode))

			cfg, err := Init(path, true, "", "")
			require.NoError(t, err)
			require.Equal(t, "file_token", cfg.Token)

			if !tc.wantWarn {
				require.Empty(t, warnings.String())
				return
			}
			require.Contains(t, warnings.String(), path)
			require.Contains(t, warnings.String(), "chmod 600")
		})
	}
}

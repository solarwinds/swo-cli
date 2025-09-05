# GitHub Copilot Instructions for swo-cli

## Repository Overview
**swo-cli** is a Go-based command-line interface for SolarWinds Observability (SWO). It's a standalone tool for retrieving, searching, and live-tailing logs from the SWO API.

## Architecture & Key Components

### 1. Main Entry Point (`cmd/swo/main.go`)
- Uses `urfave/cli/v2` framework for command-line interface
- Defines global flags: `--api-url`, `--api-token`, `--config`
- Version is injected during build via `goreleaser` (`var version = "v1.3.3"`)
- Configuration is initialized through `config.Init()` in a `Before` hook
- **IMPORTANT**: CLI flag names are `"api-token"` and `"api-url"`

### 2. Configuration System (`config/`)
**File**: `config/config.go`
- **Configuration Precedence**: CLI flags → Environment variables → Config files
- **Environment Variables**: `SWO_API_TOKEN`, `SWO_API_URL`
- **Config File**: YAML format (`.swo-cli.yml`) in home directory or current working directory
- **Constants**:
  - `DefaultAPIURL = "https://api.na-01.cloud.solarwinds.com"`
  - `APIURLContextKey = "api-url"`
  - `TokenContextKey = "token"`
- **Test Coverage**: `config/config_test.go` with comprehensive test cases

### 3. Logs Package (`logs/`)
**Command Structure**: `swo logs get [options] [search terms]`

**Key Files**:
- `logs/command.go` - Main logs command setup
- `logs/command_get.go` - Get subcommand implementation
- `logs/client.go` - HTTP client for SWO API (227 lines)
- `logs/options.go` - Time parsing, filter handling (119 lines)
- `logs/client_test.go` - Comprehensive client tests
- `logs/options_test.go` - Options and time parsing tests

**Features**:
- Live tailing (`--follow`/`-f`)
- Time range filtering (`--min-time`, `--max-time`)
- Group/system filtering (`--group`/`-g`, `--system`/`-s`)
- JSON output (`--json`/`-j`)

## Key Dependencies
```go
github.com/urfave/cli/v2 v2.27.7     // CLI framework
gopkg.in/yaml.v3 v3.0.1              // YAML configuration parsing
github.com/olebedev/when v1.1.0      // Natural language date/time parsing
github.com/stretchr/testify v1.11.1  // Testing framework
```

## API Integration
- **Base Endpoint**: `/v1/logs`
- **Authentication**: Bearer token via `Authorization` header
- **Pagination**: Supports `nextPage`/`prevPage` for result navigation
- **Filtering**: Query parameters for group, system, time range, and text search
- **Direction**: `forward` (default) or `tail` (for live following)
- **Page Size**: 1000 entries per request

## Configuration Examples

### Environment Variables (Recommended)
```bash
export SWO_API_TOKEN="your_token_here"
export SWO_API_URL="https://api.na-01.cloud.solarwinds.com"
```

### Config File (`~/.swo-cli.yml`)
```yaml
token: your_token_here
api-url: https://api.na-01.cloud.solarwinds.com
```

## Time Handling
- **Supported Formats**: RFC3339, RFC822Z, natural language ("1 hour ago", "yesterday at noon")
- **Timezone Support**: Local time by default, explicit UTC with " UTC" suffix
- **Libraries**: Standard Go time package + `olebedev/when` for human-readable parsing
- **Default Min Time**: "1 hour ago" if not specified

## Build & Development

### Build Commands
```bash
make build                    # Build the swo binary
make ci-lint                  # Run golangci-lint
make test                     # Run all tests
go test ./config              # Run tests for a single package
```

### Release Process
- **Build Tool**: `goreleaser` with GitHub Actions
- **Platforms**: Linux, macOS, Windows (ARM64 + AMD64)
- **Binary Signing**: Windows executables are code-signed
- **Version Injection**: `-ldflags "-s -w -X 'main.version=v{{ .Version }}'"`

## Testing Strategy
- **Unit Tests**: Comprehensive coverage for config, options, and client logic
- **Mock Servers**: HTTP test servers for API integration testing (`net/http/httptest`)
- **Time Mocking**: Fixed time zones (GMT) for consistent test results
- **Environment Isolation**: Clean env vars between tests
- **Test Data**: `logsData` variable provides consistent test fixtures

### Test Patterns
```go
// Time setup for consistent tests
location, err := time.LoadLocation("GMT")
require.NoError(t, err)
time.Local = location

// Environment cleanup
_ = os.Setenv("SWO_API_TOKEN", "")
_ = os.Setenv("SWO_API_URL", "")
```

## Code Style & Patterns

### Error Handling
```go
var (
    errMissingToken  = errors.New("failed to find token")
    errMissingAPIURL = errors.New("failed to find API URL")
    ErrInvalidAPIResponse = errors.New("received non-2xx status code")
)
```

### Constants
- Use constants for context keys and default values
- Group related constants together
- Use descriptive names with package prefix

### Struct Tags
```go
type Config struct {
    APIURL string `yaml:"api-url"`
    Token  string `yaml:"token"`
}
```

## Security Considerations
- **Token Security**: Environment variables recommended over config files
- **Safe Input**: Documentation uses `read -s` for secure token entry
- **No Token Logging**: Tokens are not exposed in logs or debug output
- **Bearer Authentication**: Always use `Authorization: Bearer <token>` header

## Output Formats
- **Standard**: `Jan 02 15:04:05 hostname program message`
- **JSON**: Raw JSON objects for programmatic processing
- **Colors**: Preserves ANSI color codes from log sources

## Development Guidelines
1. **Follow existing patterns** for CLI flags and configuration
2. **Always add tests** for new functionality
3. **Use consistent error handling** with predefined error variables
4. **Maintain backward compatibility** for configuration and CLI
5. **Document new environment variables** in README.md
6. **Use `make test`** for local testing
7. **Run `make ci-lint`** before committing
8. **Test with both environment variables and config files**

## Important Notes for Copilot
- Environment variables must be **exported** to be available to child processes
- Time parsing supports both absolute and relative formats
- JSON output is line-delimited, not array format
- Configuration precedence is strictly enforced: CLI > ENV > File
- All HTTP requests require Bearer authentication
- Live tailing (`--follow`) uses different API parameters (`direction=tail`)
- IMPORTANT! When chaning code, always validate that build and test pass
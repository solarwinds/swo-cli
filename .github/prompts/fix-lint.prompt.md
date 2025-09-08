---
mode: 'agent'
model: 'Claude Sonnet 4'
description: "This prompt helps you fix common Go linting issues found by golangci-lint in the SWO CLI codebase."
---

# Fix Go Linting Issues

## Overview
This prompt helps you fix common Go linting issues found by golangci-lint in the SWO CLI codebase.

## Instructions
Use this prompt to fix the most common linting violations in Go code. Run `make lint` first to see current issues, then use this prompt to fix them systematically.

## Common Linting Issues and Solutions

### 1. err113: Dynamic Error Creation
**Issue**: `do not define dynamic errors, use wrapped static errors instead`

**Problem Code**:
```go
return errors.New("some dynamic error message")
```

**Fix**:
```go
// Define static errors at package level
var (
    ErrSomeCondition = errors.New("some error message")
)

// Use in functions
return ErrSomeCondition
// Or wrap with context
return fmt.Errorf("additional context: %w", ErrSomeCondition)
```

### 2. errcheck: Unchecked Error Returns
**Issue**: `Error return value of 'function' is not checked`

**Problem Code**:
```go
file.Close()
json.NewEncoder(w).Encode(data)
os.Remove(filename)
```

**Fix**:
```go
// For defer statements in tests (where error handling isn't critical)
defer func() { _ = file.Close() }()

// For regular code, handle appropriately
if err := json.NewEncoder(w).Encode(data); err != nil {
    return fmt.Errorf("failed to encode response: %w", err)
}

// For cleanup operations where errors are expected
if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
    t.Logf("Failed to remove temp file: %v", err)
}
```

### 3. revive: Exported Function Comments
**Issue**: `comment on exported method should be of the form "MethodName ..."`

**Problem Code**:
```go
// Validate validates the options for different operations
func (o *Options) ValidateForGet() error {
```

**Fix**:
```go
// ValidateForGet validates the options for get operations
func (o *Options) ValidateForGet() error {
```

### 4. revive: Unused Parameters
**Issue**: `parameter 'name' seems to be unused, consider removing or renaming it as _`

**Problem Code**:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    // r is not used
    w.WriteHeader(200)
}
```

**Fix**:
```go
func handler(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(200)
}
```

### 5. staticcheck: Tagged Switch
**Issue**: `could use tagged switch on variable`

**Problem Code**:
```go
if tc.name == "case1" {
    // handle case1
} else if tc.name == "case2" {
    // handle case2
}
```

**Fix**:
```go
switch tc.name {
case "case1":
    // handle case1
case "case2":
    // handle case2
}
```

## Test-Specific Patterns

### Error Handling in Tests
```go
// For file operations in tests
defer func() {
    if err := os.Remove(tempFile.Name()); err != nil && !os.IsNotExist(err) {
        t.Logf("Failed to cleanup temp file: %v", err)
    }
}()

// For closing resources in tests
defer func() {
    if err := tempFile.Close(); err != nil {
        t.Logf("Failed to close temp file: %v", err)
    }
}()
```

### HTTP Response Writing in Test Handlers
```go
// Instead of ignoring the error
w.Write([]byte("response"))

// Handle it appropriately for tests
if _, err := w.Write([]byte("response")); err != nil {
    t.Errorf("Failed to write response: %v", err)
    return
}
```

## Workflow
1. Run `make lint` to see current issues
2. Use the patterns above to fix each type of issue
3. Run `make fix-lint` to auto-fix formatting
4. Run `make lint` again to verify fixes
5. Commit the changes

## SWO CLI Specific Patterns

### Error Definitions
Define package-level errors in each package:
```go
// In entities/errors.go
var (
    ErrMissingEntityID   = errors.New("entity ID is required")
    ErrMissingEntityType = errors.New("entity type is required")
    ErrInvalidTag        = errors.New("invalid tag format, expected key=value")
    ErrAtLeastOneTag     = errors.New("at least one tag is required for update")
)
```

### API Client Error Handling
```go
// In client methods
if response.StatusCode < 200 || response.StatusCode > 299 {
    return fmt.Errorf("%w: %d, response body: %s", ErrInvalidAPIResponse, response.StatusCode, string(content))
}
```

### Configuration Error Handling
```go
// In config package
if config.Token == "" {
    return nil, ErrMissingToken
}
if config.APIURL == "" {
    return nil, ErrMissingAPIURL
}
```

Remember: Good error handling makes the CLI more user-friendly and debuggable!

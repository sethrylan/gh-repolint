# Contributing

## Development Setup

### Prerequisites

- Go 1.21 or later
- GitHub CLI (`gh`) installed and authenticated
- Access to a GitHub repository for testing

### Testing

```bash
# Run all tests
go test -race ./...
```

### Running Locally

After building, you can test the commands directly. Use `--config <file>` to specify a custom configuration file.

```bash
# Run the linter on the current repository
go run .

# Run with verbose output
go run . -v

# Display merged configuration
go run . config

# Generate a starter configuration
go run . init

# Apply fixes to the current repository
go run . --fix
```

### Local Installation

To install the extension locally for testing with symbolic links (so changes are reflected immediately):

```bash
gh extension install .
```

This creates a symlink from the gh extensions directory to your local development copy. Any subsequent `go build` will update the binary that `gh repolint` uses.

To uninstall:

```bash
gh extension remove repolint
```

## Architecture

### Checks and Fixes

The codebase is organized around two main concepts:

1. **Checks** (`checks/` package): Validate repository configuration and report issues
2. **Fixes** (`fix/` package): Automatically remediate issues found by checks

### Adding a New Check

When adding a new check, you must implement both the check and corresponding fix:

1. **Create the check** in `checks/`:
   - Implement the `Check` interface
   - Return `Issue` structs with meaningful messages
   - Set `Fixable: true` for issues that can be automatically fixed
   - **Populate the `Data` field** with structured data the fixer needs (see below)

2. **Create the fix** in `fix/`:
   - Implement the `Fixer` interface
   - Read structured data from `issue.Data` using the defined constants

3. **Register the check** in `checks/check.go` (in `NewRunner`)

4. **Register the fixer** in `fix/fixer.go` (in `NewOrchestrator`)

5. **Update the README** to document the new configuration options

### Passing Data from Checks to Fixes

Checks pass structured data to fixes via the `Issue.Data` field (`map[string]string`). This avoids fragile message parsing and provides type-safe data transfer.

When adding new data keys, define them as constants in `checks/check.go` to ensure consistency between checks and fixes.

## Pull Requests

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Ensure tests pass: `go test ./...`
5. Ensure code is formatted: `go fmt ./...`
6. Submit a pull request with a clear description

# gh-repolint

A GitHub CLI extension that lints GitHub repositories against a set of configurable best practices.

## Installation

```bash
gh extension install sethrylan/gh-repolint
```

## Usage

```bash
# Run all checks on the current repository
gh repolint
```

<details>
  <summary>Example output</summary>

```sh
$ gh repolint
Repository validation failed:
  [settings] Issues is enabled but should be disabled (fixable)
  [settings] Wiki is enabled but should be disabled (fixable)
  [settings] Projects is enabled but should be disabled (fixable)
  [settings] Actions can approve PRs is disabled but should be enabled (fixable)
  [settings] Merge commits are allowed but should be disallowed (fixable)
  [settings] Rebase merge is allowed but should be disallowed (fixable)
  [settings] Auto-merge is disabled but should be enabled (fixable)
  [settings] Delete branch on merge is disabled but should be enabled (fixable)
  [settings] Dependabot alerts is disabled but should be enabled (fixable)
  [settings] Dependabot security updates is disabled but should be enabled (fixable)
  [rulesets(main)] Ruleset 'main' does not exist (fixable)
  [files(.github/dependabot.yml)] File '.github/dependabot.yml' does not exist (fixable)
  [files(.github/workflows/ci.yml)] File '.github/workflows/ci.yml' does not exist (fixable)
  [files(.github/workflows/conventional-commits.yml)] File '.github/workflows/conventional-commits.yml' does not exist (fixable)
  [files(.github/workflows/dependabot-auto.yml)] File '.github/workflows/dependabot-auto.yml' does not exist (fixable)
  [files(LICENSE)] File 'LICENSE' does not exist (fixable)
  [files(.golangci.yml)] File '.golangci.yml' does not exist (fixable)

Run with --fix to automatically fix 17 issue(s)
Error: found 17 issue(s)
```

</details>

```bash
# Auto-fix issues where possible
gh repolint --fix

# Skip specific checks
gh repolint --skip settings,dependabot

# Show verbose output
gh repolint -v

# Display merged configuration with source annotations
gh repolint config

# Generate a starter configuration file
gh repolint init
```

## Configuration

Configuration files can be written from scratch using `gh repolint init`.

Configuration is defined in `.repolint.yml` files. The tool looks for configuration in two places:

1. **Repository-level**: `.repolint.yml` in the repository root
2. **Organization-level**: `.repolint.yml` in `<owner>/<owner>` repository

Repository configuration takes precedence over organization configuration. Run `gh repolint config` to see the merged configuration with color-coded source annotations. When both configurations exist, the following merge behavior applies:
- **Scalars**: Repository value overrides organization value
- **Arrays**: Repository array replaces organization array entirely
- **Objects**: Shallow merge, repository keys override organization keys

### Example Configuration

```yaml
# gh-repolint configuration
# See https://github.com/sethrylan/gh-repolint for documentation

checks:
  settings:
    issues: false
    wiki: false
    projects: false
    allow_actions_to_approve_prs: true
    pull_request_creation_policy: "collaborators"
    default_branch: "main"
    merge:
      allow_merge_commit: false
      allow_squash_merge: true
      allow_rebase_merge: false
      allow_auto_merge: true
      delete_branch_on_merge: true
    dependabot:
      alerts: true
      security_updates: true

  actions:
    require_pinned_versions: true
    require_timeout: false
    require_minimal_permissions: true

  rulesets:
    - name: "main"
      reference: "me/me/.repolint/ruleset.json"

  files:
    - name: .github/workflows/ci.yml
      reference: "me/me/.repolint/workflows/ci.yml"
    - name: .github/dependabot.yml
      reference: "me/me/.repolint/go.dependabot.yml"
```

### Reference Files

Some configurations support reference files for validation and automated fixes. A reference file can be a local or remote file path; e.g., `me/me/.repolint/workflows/ci.yml` or a local file like `.repolint/templates/ci.yml`. If the local file does not exist, it will attempt to fetch from the remote repository using the gh cli permissions. If the reference contains these template variables, they will be replaced.

- `{{.owner}}`
- `{{.repo}}`


## Checks

### Settings Check

Validates repository settings including:
- Feature toggles (issues, wiki, projects, discussions)
- Merge settings (allowed merge types, auto-merge, branch deletion)
- Default branch name pattern matching
- Actions workflow approval permissions
- Pull request creation policy (all users or collaborators only)
- Dependabot alerts and security updates

### Actions Check

Validates GitHub Actions workflows:
- Required workflows exist
- Action versions are pinned to SHA (except `actions/*`)
- Jobs have timeout configured
- Minimal permissions are set

### Dependabot Check

Validates Dependabot configuration:
- `.github/dependabot.yml` exists
- Commit message prefix follows convention

### Rulesets Check

Validates repository rulesets:
- Required rulesets exist and are active
- Review requirements (approvals, stale review dismissal, code owner review)
- Required status checks
- Linear history requirement
- Signed commits requirement

### Files Check

Validates that specified files match reference files:
- File exists in the repository
- File content matches the reference file exactly

Reference files can be local paths or remote repository paths (e.g., `owner/owner/.repolint/workflows/ci.yml`).

## Merge Behavior

When both organization and repository configs exist:
- **Scalars**: Repository value overrides organization value
- **Arrays**: Repository array replaces organization array entirely
- **Objects**: Shallow merge, repository keys override organization keys

## Exit Codes

- `0`: All checks passed
- `1`: One or more checks failed or an error occurred

## Development

```bash
# Build
go build -o gh-repolint .

# Run tests
go test ./...

# Run linter
golangci-lint run ./...
```


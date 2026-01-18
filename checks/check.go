// Package checks provides validation checks for GitHub repository configuration.
package checks

import (
	"context"

	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// CheckType represents the type of check
type CheckType string

// Check types for different validation categories
const (
	CheckTypeSettings CheckType = "settings"
	CheckTypeActions  CheckType = "actions"
	CheckTypeRulesets CheckType = "rulesets"
	CheckTypeFiles    CheckType = "files"
)

// Data keys for passing structured data from checks to fixers
const (
	DataKeyFileName    = "file_name"
	DataKeyReference   = "reference"
	DataKeyRulesetName = "ruleset_name"
	DataKeySetting     = "setting"
)

// Issue represents a linting issue found during a check
type Issue struct {
	Type    CheckType // The check type (e.g., CheckTypeFiles, CheckTypeSettings)
	Name    string    // The specific check name (e.g., "files(.github/dependabot.yml)")
	Message string
	Fixable bool
	Data    map[string]string // Structured data for fixers (e.g., file name, reference)
}

// Check is the interface that all checks must implement
type Check interface {
	Type() CheckType // Returns the check type (e.g., CheckTypeFiles)
	Name() string    // Returns the specific check name (e.g., "files(.github/dependabot.yml)")
	Run(ctx context.Context) ([]Issue, error)
}

// Runner executes all enabled checks
type Runner struct {
	client  *github.Client
	config  *config.Config
	checks  []Check
	verbose bool
}

// NewRunner creates a new check runner
func NewRunner(client *github.Client, cfg *config.Config, verbose bool) *Runner {
	runner := &Runner{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}

	// Initialize all checks
	runner.checks = []Check{
		NewSettingsCheck(client, cfg.Checks.Settings, verbose),
		NewActionsCheck(client, cfg.Checks.Actions, verbose),
	}

	// Add ruleset checks
	for _, rs := range cfg.Checks.Rulesets {
		runner.checks = append(runner.checks, NewRulesetsCheck(client, &rs, verbose))
	}

	// Add file checks
	for _, f := range cfg.Checks.Files {
		runner.checks = append(runner.checks, NewFilesCheck(client, &f, verbose))
	}

	return runner
}

// Run executes all enabled checks and returns all issues found
func (r *Runner) Run(ctx context.Context, skip []string) ([]Issue, error) {
	var allIssues []Issue

	skipMap := make(map[string]bool)
	for _, s := range skip {
		skipMap[s] = true
	}

	for _, check := range r.checks {

		if skipMap[check.Name()] {
			continue
		}

		issues, err := check.Run(ctx)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// GetCheckNames returns the names of all available checks
func (r *Runner) GetCheckNames() []string {
	names := make([]string, 0, len(r.checks))
	for _, check := range r.checks {
		names = append(names, check.Name())
	}
	return names
}

// CheckStatus represents the status of a check
type CheckStatus struct {
	Name    string
	Skipped bool
}

// GetCheckStatuses returns the status of all checks
func (r *Runner) GetCheckStatuses() []CheckStatus {
	statuses := make([]CheckStatus, 0, len(r.checks))
	for _, check := range r.checks {
		statuses = append(statuses, CheckStatus{
			Name: check.Name(),
		})
	}
	return statuses
}

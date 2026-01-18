// Package fix provides automatic remediation for repolint issues.
package fix

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// Result represents the result of a fix attempt
type Result struct {
	Issue checks.Issue
	Fixed bool
	Error error
}

// failedResult creates a Result indicating the fix failed with an error.
func failedResult(issue checks.Issue, err error) (*Result, error) {
	return &Result{
		Issue: issue,
		Fixed: false,
		Error: err,
	}, nil
}

// successResult creates a Result indicating the fix succeeded.
func successResult(issue checks.Issue) (*Result, error) {
	return &Result{
		Issue: issue,
		Fixed: true,
	}, nil
}

// Fixer is the interface for fixing issues
type Fixer interface {
	Name() string
	Fix(ctx context.Context, issue checks.Issue) (*Result, error)
}

// Orchestrator coordinates all fixers
type Orchestrator struct {
	client  *github.Client
	config  *config.Config
	fixers  map[checks.CheckType]Fixer
	verbose bool
}

// NewOrchestrator creates a new fix orchestrator
func NewOrchestrator(client *github.Client, cfg *config.Config, verbose bool) *Orchestrator {
	o := &Orchestrator{
		client:  client,
		config:  cfg,
		fixers:  make(map[checks.CheckType]Fixer),
		verbose: verbose,
	}

	// Register fixers
	o.fixers[checks.CheckTypeSettings] = NewSettingsFixer(client, cfg.Checks.Settings, verbose)
	o.fixers[checks.CheckTypeActions] = NewActionsFixer(client, cfg.Checks.Actions, verbose)
	o.fixers[checks.CheckTypeRulesets] = NewRulesetsFixer(client, cfg.Checks.Rulesets, verbose)
	o.fixers[checks.CheckTypeFiles] = NewFilesFixer(client, cfg.Checks.Files, verbose)

	return o
}

// Fix attempts to fix all fixable issues
func (o *Orchestrator) Fix(ctx context.Context, issues []checks.Issue) ([]Result, error) {
	var results []Result

	for _, issue := range issues {
		if !issue.Fixable {
			results = append(results, Result{
				Issue: issue,
				Fixed: false,
				Error: errors.New("issue is not fixable"),
			})
			continue
		}

		fixer, ok := o.fixers[issue.Type]
		if !ok {
			results = append(results, Result{
				Issue: issue,
				Fixed: false,
				Error: fmt.Errorf("no fixer for check type '%s'", issue.Type),
			})
			continue
		}

		result, err := fixer.Fix(ctx, issue)
		if err != nil {
			results = append(results, Result{
				Issue: issue,
				Fixed: false,
				Error: err,
			})
		} else {
			results = append(results, *result)
		}
	}

	return results, nil
}

// FixableCount returns the number of fixable issues
func FixableCount(issues []checks.Issue) int {
	count := 0
	for _, issue := range issues {
		if issue.Fixable {
			count++
		}
	}
	return count
}

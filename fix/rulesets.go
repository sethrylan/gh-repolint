package fix

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// RulesetsFixer fixes ruleset configuration issues
type RulesetsFixer struct {
	client  *github.Client
	configs []config.RulesetConfig
	verbose bool
}

// NewRulesetsFixer creates a new rulesets fixer
func NewRulesetsFixer(client *github.Client, cfgs []config.RulesetConfig, verbose bool) *RulesetsFixer {
	return &RulesetsFixer{
		client:  client,
		configs: cfgs,
		verbose: verbose,
	}
}

// Name returns the fixer name
func (f *RulesetsFixer) Name() string {
	return "rulesets"
}

// Fix attempts to fix a ruleset issue
func (f *RulesetsFixer) Fix(ctx context.Context, issue checks.Issue) (*Result, error) {
	// Get ruleset name from issue data
	rulesetName := issue.Data[checks.DataKeyRulesetName]
	if rulesetName == "" {
		return failedResult(issue, errors.New("issue data missing ruleset_name"))
	}

	// Find the config for this ruleset
	var cfg *config.RulesetConfig
	for i := range f.configs {
		if f.configs[i].Name == rulesetName {
			cfg = &f.configs[i]
			break
		}
	}

	if cfg == nil {
		return failedResult(issue, fmt.Errorf("no config found for ruleset '%s'", rulesetName))
	}

	if cfg.Reference == "" {
		return failedResult(issue, fmt.Errorf("ruleset '%s' has no reference specified", rulesetName))
	}

	// Fetch the reference ruleset JSON
	refRuleset, err := github.FetchReferenceRuleset(cfg.Reference, f.client)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to fetch reference ruleset: %w", err))
	}

	// Check if ruleset exists to determine if we need to create or update
	rulesets, err := f.client.GetRulesets()
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to fetch rulesets: %w", err))
	}

	var rulesetID int
	for _, rs := range rulesets {
		if rs.Name == cfg.Name {
			rulesetID = rs.ID
			break
		}
	}

	if rulesetID == 0 {
		// Ruleset doesn't exist, create it
		return f.createRuleset(issue, cfg, refRuleset)
	}

	// Ruleset exists, update it
	return f.updateRulesetByID(issue, cfg, refRuleset, rulesetID)
}

func (f *RulesetsFixer) createRuleset(issue checks.Issue, cfg *config.RulesetConfig, refRuleset *github.Ruleset) (*Result, error) {
	req := f.buildRulesetRequest(cfg, refRuleset)

	_, err := f.client.CreateRuleset(req)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to create ruleset: %w", err))
	}

	return successResult(issue)
}

func (f *RulesetsFixer) updateRulesetByID(issue checks.Issue, cfg *config.RulesetConfig, refRuleset *github.Ruleset, rulesetID int) (*Result, error) {
	req := f.buildRulesetRequest(cfg, refRuleset)

	if err := f.client.UpdateRuleset(rulesetID, req); err != nil {
		return failedResult(issue, fmt.Errorf("failed to update ruleset: %w", err))
	}

	return successResult(issue)
}

// buildRulesetRequest creates a RulesetCreateRequest from the reference ruleset
func (f *RulesetsFixer) buildRulesetRequest(cfg *config.RulesetConfig, refRuleset *github.Ruleset) *github.RulesetCreateRequest {
	// Ensure conditions have proper include/exclude arrays (GitHub API requires both)
	conditions := refRuleset.Conditions
	if conditions != nil && conditions.RefName != nil {
		if conditions.RefName.Include == nil {
			conditions.RefName.Include = []string{}
		}
		if conditions.RefName.Exclude == nil {
			conditions.RefName.Exclude = []string{}
		}
	}

	// Ensure bypass actors is not nil
	bypassActors := refRuleset.BypassActors
	if bypassActors == nil {
		bypassActors = []github.BypassActor{}
	}

	req := &github.RulesetCreateRequest{
		Name:         cfg.Name, // Use the configured name, not the reference name
		Target:       refRuleset.Target,
		Enforcement:  refRuleset.Enforcement,
		Conditions:   conditions,
		Rules:        refRuleset.Rules,
		BypassActors: bypassActors,
	}

	return req
}

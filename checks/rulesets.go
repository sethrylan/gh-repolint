package checks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// RulesetsCheck validates repository rulesets
type RulesetsCheck struct {
	client  *github.Client
	config  *config.RulesetConfig
	verbose bool
}

// NewRulesetsCheck creates a new rulesets check
func NewRulesetsCheck(client *github.Client, cfg *config.RulesetConfig, verbose bool) *RulesetsCheck {
	return &RulesetsCheck{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Type returns the check type
func (c *RulesetsCheck) Type() CheckType {
	return CheckTypeRulesets
}

// Name returns the check name
func (c *RulesetsCheck) Name() string {
	return "rulesets(" + c.config.Name + ")"
}

// Run executes the rulesets check
func (c *RulesetsCheck) Run(ctx context.Context) ([]Issue, error) {
	if c.config == nil {
		return nil, nil
	}

	if c.config.Reference == "" {
		return nil, fmt.Errorf("ruleset '%s' missing required reference field", c.config.Name)
	}

	var issues []Issue

	// Fetch the expected ruleset JSON from reference
	expectedRuleset, err := github.FetchReferenceRuleset(c.config.Reference, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reference ruleset: %w", err)
	}

	// Fetch all rulesets from the repository
	rulesets, err := c.client.GetRulesets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rulesets: %w", err)
	}

	// Find the ruleset by name
	var matchingRuleset *github.Ruleset
	for _, rs := range rulesets {
		if rs.Name == c.config.Name {
			// Fetch full ruleset details
			fullRuleset, err := c.client.GetRuleset(rs.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch ruleset details: %w", err)
			}
			matchingRuleset = fullRuleset
			break
		}
	}

	if matchingRuleset == nil {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Ruleset '%s' does not exist", c.config.Name),
			Fixable: true,
			Data: map[string]string{
				DataKeyRulesetName: c.config.Name,
				DataKeyReference:   c.config.Reference,
			},
		})
		return issues, nil
	}

	// Compare the actual ruleset with the expected ruleset from reference
	if !c.rulesetsMatch(matchingRuleset, expectedRuleset) {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Ruleset '%s' does not match reference '%s'", c.config.Name, c.config.Reference),
			Fixable: true,
			Data: map[string]string{
				DataKeyRulesetName: c.config.Name,
				DataKeyReference:   c.config.Reference,
			},
		})
	}

	return issues, nil
}

// rulesetsMatch compares two rulesets for equivalence
// It compares the fields that matter for configuration, ignoring ID and other runtime fields
func (c *RulesetsCheck) rulesetsMatch(actual, expected *github.Ruleset) bool {
	// Compare enforcement
	if actual.Enforcement != expected.Enforcement {
		return false
	}

	// Compare target
	if actual.Target != expected.Target {
		return false
	}

	// Compare conditions
	if !conditionsMatch(actual.Conditions, expected.Conditions) {
		return false
	}

	// Compare rules
	if !rulesMatch(actual.Rules, expected.Rules) {
		return false
	}

	// Compare bypass actors
	if !bypassActorsMatch(actual.BypassActors, expected.BypassActors) {
		return false
	}

	return true
}

// conditionsMatch compares ruleset conditions
func conditionsMatch(actual, expected *github.RulesetConditions) bool {
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}

	// Compare RefName conditions
	if actual.RefName == nil && expected.RefName == nil {
		return true
	}
	if actual.RefName == nil || expected.RefName == nil {
		return false
	}

	if !stringSlicesEqual(actual.RefName.Include, expected.RefName.Include) {
		return false
	}
	if !stringSlicesEqual(actual.RefName.Exclude, expected.RefName.Exclude) {
		return false
	}

	return true
}

// rulesMatch compares ruleset rules
func rulesMatch(actual, expected []github.RulesetRule) bool {
	if len(actual) != len(expected) {
		return false
	}

	// Build maps by rule type for comparison
	actualByType := make(map[string]github.RulesetRule)
	for _, rule := range actual {
		actualByType[rule.Type] = rule
	}

	expectedByType := make(map[string]github.RulesetRule)
	for _, rule := range expected {
		expectedByType[rule.Type] = rule
	}

	// Check that all expected rules exist and match
	for ruleType, expectedRule := range expectedByType {
		actualRule, ok := actualByType[ruleType]
		if !ok {
			return false
		}
		if !ruleParametersMatch(actualRule.Parameters, expectedRule.Parameters) {
			return false
		}
	}

	// Check that there are no extra rules in actual
	for ruleType := range actualByType {
		if _, ok := expectedByType[ruleType]; !ok {
			return false
		}
	}

	return true
}

// ruleParametersMatch compares rule parameters
func ruleParametersMatch(actual, expected map[string]any) bool {
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		// One is nil, the other is not - but empty maps should be considered equal to nil
		if actual == nil && len(expected) == 0 {
			return true
		}
		if expected == nil && len(actual) == 0 {
			return true
		}
		return false
	}

	// Compare JSON representations for deep equality
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return false
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}

	return string(actualJSON) == string(expectedJSON)
}

// bypassActorsMatch compares bypass actors
func bypassActorsMatch(actual, expected []github.BypassActor) bool {
	if len(actual) != len(expected) {
		return false
	}

	// Build a set of expected actors for comparison
	expectedSet := make(map[string]bool)
	for _, actor := range expected {
		key := fmt.Sprintf("%d:%s:%s", actor.ActorID, actor.ActorType, actor.BypassMode)
		expectedSet[key] = true
	}

	for _, actor := range actual {
		key := fmt.Sprintf("%d:%s:%s", actor.ActorID, actor.ActorType, actor.BypassMode)
		if !expectedSet[key] {
			return false
		}
	}

	return true
}

// stringSlicesEqual checks if two string slices are equal
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Package config provides configuration loading and merging for repolint.
package config

// Config represents the complete repolint configuration
type Config struct {
	Checks ChecksConfig `yaml:"checks" validate:"required"`
}

// ChecksConfig contains all check configurations
type ChecksConfig struct {
	Settings *SettingsConfig `yaml:"settings,omitempty"`
	Actions  *ActionsConfig  `yaml:"actions,omitempty"`
	Rulesets []RulesetConfig `yaml:"rulesets,omitempty"`
	Files    []FileConfig    `yaml:"files,omitempty"`
}

// SettingsConfig defines repository settings to validate
type SettingsConfig struct {
	Issues                    *bool                     `yaml:"issues,omitempty"`
	Wiki                      *bool                     `yaml:"wiki,omitempty"`
	Projects                  *bool                     `yaml:"projects,omitempty"`
	Discussions               *bool                     `yaml:"discussions,omitempty"`
	AllowActionsToApprovePRs  *bool                     `yaml:"allow_actions_to_approve_prs,omitempty"`
	PullRequestCreationPolicy string                    `yaml:"pull_request_creation_policy,omitempty"`
	Merge                     *MergeConfig              `yaml:"merge,omitempty"`
	DefaultBranch             string                    `yaml:"default_branch,omitempty"`
	Dependabot                *DependabotSettingsConfig `yaml:"dependabot,omitempty"`
}

// DependabotSettingsConfig defines Dependabot-related settings to validate
type DependabotSettingsConfig struct {
	// Alerts enables/disables Dependabot alerts (vulnerability alerts)
	Alerts *bool `yaml:"alerts,omitempty"`
	// SecurityUpdates enables/disables Dependabot security updates (automated security fixes)
	SecurityUpdates *bool `yaml:"security_updates,omitempty"`
}

// MergeConfig defines merge-related settings
type MergeConfig struct {
	AllowMergeCommit                         *bool `yaml:"allow_merge_commit,omitempty"`
	AllowSquashMerge                         *bool `yaml:"allow_squash_merge,omitempty"`
	AllowRebaseMerge                         *bool `yaml:"allow_rebase_merge,omitempty"`
	AllowAutoMerge                           *bool `yaml:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge                      *bool `yaml:"delete_branch_on_merge,omitempty"`
	AlwaysSuggestUpdatingPullRequestBranches *bool `yaml:"always_suggest_updating_pull_request_branches,omitempty"`
}

// ActionsConfig defines GitHub Actions workflow validation settings
type ActionsConfig struct {
	RequirePinnedVersions     *bool            `yaml:"require_pinned_versions,omitempty"`
	RequiredWorkflows         []WorkflowConfig `yaml:"required_workflows,omitempty"`
	RequireTimeout            *bool            `yaml:"require_timeout,omitempty"`
	MaxTimeoutMinutes         *int             `yaml:"max_timeout_minutes,omitempty"`
	RequireMinimalPermissions *bool            `yaml:"require_minimal_permissions,omitempty"`
}

// WorkflowConfig defines a required workflow file
type WorkflowConfig struct {
	Path      string `yaml:"path" validate:"required"`
	Reference string `yaml:"reference,omitempty"`
}

// RulesetConfig defines a repository ruleset configuration
// The reference field points to a JSON file exported via `gh ruleset export`
// Format: owner/repo/path/to/ruleset.json
type RulesetConfig struct {
	Name      string `yaml:"name" validate:"required"`
	Reference string `yaml:"reference" validate:"required"`
}

// FileConfig defines a file that should match a reference
// The reference field points to a file that the local file should match
// Format: owner/repo/path/to/file or local path
type FileConfig struct {
	Name      string `yaml:"name" validate:"required"`
	Reference string `yaml:"reference" validate:"required"`
}

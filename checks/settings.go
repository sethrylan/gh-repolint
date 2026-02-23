package checks

import (
	"context"
	"fmt"

	"github.com/gobwas/glob"

	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// SettingsCheck validates repository settings
type SettingsCheck struct {
	client  *github.Client
	config  *config.SettingsConfig
	verbose bool
}

// NewSettingsCheck creates a new settings check
func NewSettingsCheck(client *github.Client, cfg *config.SettingsConfig, verbose bool) *SettingsCheck {
	return &SettingsCheck{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Type returns the check type
func (c *SettingsCheck) Type() CheckType {
	return CheckTypeSettings
}

// Name returns the check name
func (c *SettingsCheck) Name() string {
	return "settings"
}

// Run executes the settings check
func (c *SettingsCheck) Run(ctx context.Context) ([]Issue, error) {
	if c.config == nil {
		return nil, nil
	}

	repo, err := c.client.GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}

	var issues []Issue

	// Check feature toggles
	if c.config.Issues != nil && repo.HasIssues != *c.config.Issues {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Issues is %s but should be %s", boolToEnabled(repo.HasIssues), boolToEnabled(*c.config.Issues)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "issues"},
		})
	}

	if c.config.Wiki != nil && repo.HasWiki != *c.config.Wiki {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Wiki is %s but should be %s", boolToEnabled(repo.HasWiki), boolToEnabled(*c.config.Wiki)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "wiki"},
		})
	}

	if c.config.Projects != nil && repo.HasProjects != *c.config.Projects {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Projects is %s but should be %s", boolToEnabled(repo.HasProjects), boolToEnabled(*c.config.Projects)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "projects"},
		})
	}

	if c.config.Discussions != nil && repo.HasDiscussions != *c.config.Discussions {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Discussions is %s but should be %s", boolToEnabled(repo.HasDiscussions), boolToEnabled(*c.config.Discussions)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "discussions"},
		})
	}

	// Check actions permissions
	if c.config.AllowActionsToApprovePRs != nil {
		perms, err := c.client.GetWorkflowPermissions()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch workflow permissions: %w", err)
		}
		if perms.CanApprovePullRequestReviews != *c.config.AllowActionsToApprovePRs {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Actions can approve PRs is %s but should be %s", boolToEnabled(perms.CanApprovePullRequestReviews), boolToEnabled(*c.config.AllowActionsToApprovePRs)),
				Fixable: true,
				Data:    map[string]string{DataKeySetting: "actions_approve_prs"},
			})
		}
	}

	// Check merge settings
	if c.config.Merge != nil {
		mergeIssues := c.checkMergeSettings(repo)
		issues = append(issues, mergeIssues...)
	}

	// Check default branch pattern
	if c.config.DefaultBranch != "" {
		g, err := glob.Compile(c.config.DefaultBranch)
		if err != nil {
			return nil, fmt.Errorf("invalid default_branch pattern: %w", err)
		}
		if !g.Match(repo.DefaultBranch) {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Default branch '%s' does not match pattern '%s'", repo.DefaultBranch, c.config.DefaultBranch),
				Fixable: false, // Branch renaming requires manual intervention
			})
		}
	}

	// Check pull request creation policy
	if c.config.PullRequestCreationPolicy != "" && repo.PullRequestCreationPolicy != c.config.PullRequestCreationPolicy {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Pull request creation policy is '%s' but should be '%s'", repo.PullRequestCreationPolicy, c.config.PullRequestCreationPolicy),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "pull_request_creation_policy"},
		})
	}

	// Check Dependabot settings
	if c.config.Dependabot != nil {
		dependabotIssues, err := c.checkDependabotSettings()
		if err != nil {
			return nil, err
		}
		issues = append(issues, dependabotIssues...)
	}

	return issues, nil
}

func (c *SettingsCheck) checkMergeSettings(repo *github.Repository) []Issue {
	var issues []Issue
	merge := c.config.Merge

	if merge.AllowMergeCommit != nil && repo.AllowMergeCommit != *merge.AllowMergeCommit {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Merge commits are %s but should be %s", boolToAllowed(repo.AllowMergeCommit), boolToAllowed(*merge.AllowMergeCommit)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "merge_commit"},
		})
	}

	if merge.AllowSquashMerge != nil && repo.AllowSquashMerge != *merge.AllowSquashMerge {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Squash merge is %s but should be %s", boolToAllowed(repo.AllowSquashMerge), boolToAllowed(*merge.AllowSquashMerge)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "squash_merge"},
		})
	}

	if merge.AllowRebaseMerge != nil && repo.AllowRebaseMerge != *merge.AllowRebaseMerge {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Rebase merge is %s but should be %s", boolToAllowed(repo.AllowRebaseMerge), boolToAllowed(*merge.AllowRebaseMerge)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "rebase_merge"},
		})
	}

	if merge.AllowAutoMerge != nil && repo.AllowAutoMerge != *merge.AllowAutoMerge {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Auto-merge is %s but should be %s", boolToEnabled(repo.AllowAutoMerge), boolToEnabled(*merge.AllowAutoMerge)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "auto_merge"},
		})
	}

	if merge.DeleteBranchOnMerge != nil && repo.DeleteBranchOnMerge != *merge.DeleteBranchOnMerge {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Delete branch on merge is %s but should be %s", boolToEnabled(repo.DeleteBranchOnMerge), boolToEnabled(*merge.DeleteBranchOnMerge)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "delete_branch_on_merge"},
		})
	}

	if merge.AlwaysSuggestUpdatingPullRequestBranches != nil && repo.AllowUpdateBranch != *merge.AlwaysSuggestUpdatingPullRequestBranches {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Always suggest updating PR branches is %s but should be %s", boolToEnabled(repo.AllowUpdateBranch), boolToEnabled(*merge.AlwaysSuggestUpdatingPullRequestBranches)),
			Fixable: true,
			Data:    map[string]string{DataKeySetting: "update_branch"},
		})
	}

	return issues
}

func boolToEnabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func boolToAllowed(b bool) string {
	if b {
		return "allowed"
	}
	return "disallowed"
}

func (c *SettingsCheck) checkDependabotSettings() ([]Issue, error) {
	var issues []Issue
	dep := c.config.Dependabot

	// Check Dependabot alerts (vulnerability alerts)
	if dep.Alerts != nil {
		enabled, err := c.client.GetVulnerabilityAlertsEnabled()
		if err != nil {
			return nil, fmt.Errorf("failed to check vulnerability alerts: %w", err)
		}
		if enabled != *dep.Alerts {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Dependabot alerts is %s but should be %s", boolToEnabled(enabled), boolToEnabled(*dep.Alerts)),
				Fixable: true,
				Data:    map[string]string{DataKeySetting: "dependabot_alerts"},
			})
		}
	}

	// Check Dependabot security updates (automated security fixes)
	if dep.SecurityUpdates != nil {
		fixes, err := c.client.GetAutomatedSecurityFixes()
		if err != nil {
			return nil, fmt.Errorf("failed to check automated security fixes: %w", err)
		}
		if fixes.Enabled != *dep.SecurityUpdates {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Dependabot security updates is %s but should be %s", boolToEnabled(fixes.Enabled), boolToEnabled(*dep.SecurityUpdates)),
				Fixable: true,
				Data:    map[string]string{DataKeySetting: "dependabot_security_updates"},
			})
		}
	}

	return issues, nil
}

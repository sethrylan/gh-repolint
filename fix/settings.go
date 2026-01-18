package fix

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// SettingsFixer fixes repository settings issues
type SettingsFixer struct {
	client  *github.Client
	config  *config.SettingsConfig
	verbose bool
}

// NewSettingsFixer creates a new repository fixer
func NewSettingsFixer(client *github.Client, cfg *config.SettingsConfig, verbose bool) *SettingsFixer {
	return &SettingsFixer{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Name returns the fixer name
func (f *SettingsFixer) Name() string {
	return "settings"
}

// Fix attempts to fix a repository issue
func (f *SettingsFixer) Fix(ctx context.Context, issue checks.Issue) (*Result, error) {
	setting := issue.Data[checks.DataKeySetting]
	if setting == "" {
		return failedResult(issue, errors.New("issue data missing setting"))
	}

	switch setting {
	case "actions_approve_prs":
		return f.fixActionsApprove(issue)
	case "dependabot_alerts":
		return f.fixDependabotAlerts(issue)
	case "dependabot_security_updates":
		return f.fixDependabotSecurityUpdates(issue)
	}

	// Handle repository settings fixes
	req := &github.RepoUpdateRequest{}

	switch setting {
	case "issues":
		req.HasIssues = f.config.Issues
	case "wiki":
		req.HasWiki = f.config.Wiki
	case "projects":
		req.HasProjects = f.config.Projects
	case "discussions":
		req.HasDiscussions = f.config.Discussions
	case "merge_commit":
		req.AllowMergeCommit = f.config.Merge.AllowMergeCommit
	case "squash_merge":
		req.AllowSquashMerge = f.config.Merge.AllowSquashMerge
	case "rebase_merge":
		req.AllowRebaseMerge = f.config.Merge.AllowRebaseMerge
	case "auto_merge":
		req.AllowAutoMerge = f.config.Merge.AllowAutoMerge
	case "delete_branch_on_merge":
		req.DeleteBranchOnMerge = f.config.Merge.DeleteBranchOnMerge
	case "update_branch":
		req.AllowUpdateBranch = f.config.Merge.AlwaysSuggestUpdatingPullRequestBranches
	default:
		return failedResult(issue, fmt.Errorf("unknown setting: %s", setting))
	}

	if err := f.client.UpdateRepository(req); err != nil {
		return failedResult(issue, fmt.Errorf("failed to update repository: %w", err))
	}

	return successResult(issue)
}

func (f *SettingsFixer) fixActionsApprove(issue checks.Issue) (*Result, error) {
	if f.config.AllowActionsToApprovePRs == nil {
		return failedResult(issue, errors.New("allow_actions_to_approve_prs not configured"))
	}

	if err := f.client.UpdateWorkflowPermissions(*f.config.AllowActionsToApprovePRs); err != nil {
		return failedResult(issue, fmt.Errorf("failed to update workflow permissions: %w", err))
	}

	return successResult(issue)
}

func (f *SettingsFixer) fixDependabotAlerts(issue checks.Issue) (*Result, error) {
	if f.config.Dependabot == nil || f.config.Dependabot.Alerts == nil {
		return failedResult(issue, errors.New("dependabot alerts not configured"))
	}

	var err error
	if *f.config.Dependabot.Alerts {
		err = f.client.EnableVulnerabilityAlerts()
	} else {
		err = f.client.DisableVulnerabilityAlerts()
	}

	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to update vulnerability alerts: %w", err))
	}

	return successResult(issue)
}

func (f *SettingsFixer) fixDependabotSecurityUpdates(issue checks.Issue) (*Result, error) {
	if f.config.Dependabot == nil || f.config.Dependabot.SecurityUpdates == nil {
		return failedResult(issue, errors.New("dependabot security updates not configured"))
	}

	var err error
	if *f.config.Dependabot.SecurityUpdates {
		err = f.client.EnableAutomatedSecurityFixes()
	} else {
		err = f.client.DisableAutomatedSecurityFixes()
	}

	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to update automated security fixes: %w", err))
	}

	return successResult(issue)
}

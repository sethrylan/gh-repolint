package config

// MergeConfigs merges owner and repo configs.
// Repo config takes precedence over owner config.
// Rules:
// - Scalar values: repo overrides owner
// - Arrays: repo replaces entirely (no appending)
// - Objects: shallow merge, repo keys override owner keys
func MergeConfigs(owner, repo *Config) *Config {
	if owner == nil && repo == nil {
		return nil
	}
	if owner == nil {
		return repo
	}
	if repo == nil {
		return owner
	}

	result := &Config{
		Checks: ChecksConfig{
			Settings: mergeSettingsConfig(owner.Checks.Settings, repo.Checks.Settings),
			Actions:  mergeActionsConfig(owner.Checks.Actions, repo.Checks.Actions),
			Rulesets: mergeRulesets(owner.Checks.Rulesets, repo.Checks.Rulesets),
			Files:    mergeFiles(owner.Checks.Files, repo.Checks.Files),
		},
	}

	return result
}

func mergeSettingsConfig(owner, repo *SettingsConfig) *SettingsConfig {
	if owner == nil && repo == nil {
		return nil
	}
	if owner == nil {
		return repo
	}
	if repo == nil {
		return owner
	}

	result := &SettingsConfig{
		Issues:                   mergeBoolPtr(owner.Issues, repo.Issues),
		Wiki:                     mergeBoolPtr(owner.Wiki, repo.Wiki),
		Projects:                 mergeBoolPtr(owner.Projects, repo.Projects),
		Discussions:              mergeBoolPtr(owner.Discussions, repo.Discussions),
		AllowActionsToApprovePRs: mergeBoolPtr(owner.AllowActionsToApprovePRs, repo.AllowActionsToApprovePRs),
		DefaultBranch:            mergeString(owner.DefaultBranch, repo.DefaultBranch),
		Merge:                    mergeMergeConfig(owner.Merge, repo.Merge),
		Dependabot:               mergeDependabotSettingsConfig(owner.Dependabot, repo.Dependabot),
	}

	return result
}

func mergeMergeConfig(owner, repo *MergeConfig) *MergeConfig {
	if owner == nil && repo == nil {
		return nil
	}
	if owner == nil {
		return repo
	}
	if repo == nil {
		return owner
	}

	return &MergeConfig{
		AllowMergeCommit:                         mergeBoolPtr(owner.AllowMergeCommit, repo.AllowMergeCommit),
		AllowSquashMerge:                         mergeBoolPtr(owner.AllowSquashMerge, repo.AllowSquashMerge),
		AllowRebaseMerge:                         mergeBoolPtr(owner.AllowRebaseMerge, repo.AllowRebaseMerge),
		AllowAutoMerge:                           mergeBoolPtr(owner.AllowAutoMerge, repo.AllowAutoMerge),
		DeleteBranchOnMerge:                      mergeBoolPtr(owner.DeleteBranchOnMerge, repo.DeleteBranchOnMerge),
		AlwaysSuggestUpdatingPullRequestBranches: mergeBoolPtr(owner.AlwaysSuggestUpdatingPullRequestBranches, repo.AlwaysSuggestUpdatingPullRequestBranches),
	}
}

func mergeActionsConfig(owner, repo *ActionsConfig) *ActionsConfig {
	if owner == nil && repo == nil {
		return nil
	}
	if owner == nil {
		return repo
	}
	if repo == nil {
		return owner
	}

	result := &ActionsConfig{
		RequirePinnedVersions:     mergeBoolPtr(owner.RequirePinnedVersions, repo.RequirePinnedVersions),
		RequireTimeout:            mergeBoolPtr(owner.RequireTimeout, repo.RequireTimeout),
		MaxTimeoutMinutes:         mergeIntPtr(owner.MaxTimeoutMinutes, repo.MaxTimeoutMinutes),
		RequireMinimalPermissions: mergeBoolPtr(owner.RequireMinimalPermissions, repo.RequireMinimalPermissions),
	}

	// Arrays: repo replaces entirely
	if repo.RequiredWorkflows != nil {
		result.RequiredWorkflows = repo.RequiredWorkflows
	} else {
		result.RequiredWorkflows = owner.RequiredWorkflows
	}

	return result
}

func mergeRulesets(owner, repo []RulesetConfig) []RulesetConfig {
	// Arrays: repo replaces entirely
	if repo != nil {
		return repo
	}
	return owner
}

func mergeFiles(owner, repo []FileConfig) []FileConfig {
	// Arrays: repo replaces entirely
	if repo != nil {
		return repo
	}
	return owner
}

func mergeDependabotSettingsConfig(owner, repo *DependabotSettingsConfig) *DependabotSettingsConfig {
	if owner == nil && repo == nil {
		return nil
	}
	if owner == nil {
		return repo
	}
	if repo == nil {
		return owner
	}

	return &DependabotSettingsConfig{
		Alerts:          mergeBoolPtr(owner.Alerts, repo.Alerts),
		SecurityUpdates: mergeBoolPtr(owner.SecurityUpdates, repo.SecurityUpdates),
	}
}

// Helper functions for merging values

func mergeBoolPtr(owner, repo *bool) *bool {
	if repo != nil {
		return repo
	}
	return owner
}

func mergeIntPtr(owner, repo *int) *int {
	if repo != nil {
		return repo
	}
	return owner
}

func mergeString(owner, repo string) string {
	if repo != "" {
		return repo
	}
	return owner
}

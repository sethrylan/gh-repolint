package config

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const (
	colorReset = "\033[0m"
	colorRepo  = "\033[36m" // Cyan for repo-level
	colorOwner = "\033[33m" // Yellow for owner-level
	colorRed   = "\033[31m" // Red for invalid
)

// ReferenceValidator is a function that validates a reference and returns an error if invalid
type ReferenceValidator func(reference string) error

// DisplayResult contains the result of displaying config
type DisplayResult struct {
	InvalidReferences []string
}

// DisplayConfig writes the merged config with color-coded source annotations
// Returns a DisplayResult containing any invalid references found
func DisplayConfig(w io.Writer, loaded *LoadedConfig, useColor bool, validator ReferenceValidator) *DisplayResult {
	result := &DisplayResult{}

	_, _ = fmt.Fprintln(w, "Configuration:")
	_, _ = fmt.Fprintln(w, "")

	if useColor {
		_, _ = fmt.Fprintf(w, "Legend: %srepo-level%s | %sowner-level%s\n",
			colorRepo, colorReset, colorOwner, colorReset)
	} else {
		_, _ = fmt.Fprintln(w, "Legend: [repo] repo-level | [owner] owner-level")
	}
	_, _ = fmt.Fprintln(w, "")

	displayChecks(w, loaded, useColor, 0, validator, result)

	return result
}

func displayChecks(w io.Writer, loaded *LoadedConfig, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	cfg := loaded.Config
	if cfg == nil {
		return
	}

	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "checks:")

	if cfg.Checks.Settings != nil {
		displaySettingsConfig(w, loaded, useColor, indent+2)
	}

	if cfg.Checks.Actions != nil {
		displayActionsConfig(w, loaded, useColor, indent+2)
	}

	if len(cfg.Checks.Rulesets) > 0 {
		displayRulesetsConfig(w, loaded, useColor, indent+2, validator, result)
	}

	if len(cfg.Checks.Files) > 0 {
		displayFilesConfig(w, loaded, useColor, indent+2, validator, result)
	}
}

func displaySettingsConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "settings:")

	cfg := loaded.Config.Checks.Settings
	repo := getRepoSettings(loaded)
	owner := getOwnerSettings(loaded)

	displayBoolField(w, "issues", cfg.Issues, getBoolSource(repo, owner, "Issues"), useColor, indent+2)
	displayBoolField(w, "wiki", cfg.Wiki, getBoolSource(repo, owner, "Wiki"), useColor, indent+2)
	displayBoolField(w, "projects", cfg.Projects, getBoolSource(repo, owner, "Projects"), useColor, indent+2)
	displayBoolField(w, "discussions", cfg.Discussions, getBoolSource(repo, owner, "Discussions"), useColor, indent+2)
	displayBoolField(w, "allow_actions_to_approve_prs", cfg.AllowActionsToApprovePRs, getBoolSource(repo, owner, "AllowActionsToApprovePRs"), useColor, indent+2)

	if cfg.PullRequestCreationPolicy != "" {
		source := SourceOwner
		if repo != nil && repo.PullRequestCreationPolicy != "" {
			source = SourceRepo
		}
		displayStringField(w, "pull_request_creation_policy", cfg.PullRequestCreationPolicy, source, useColor, indent+2)
	}

	if cfg.DefaultBranch != "" {
		source := SourceOwner
		if repo != nil && repo.DefaultBranch != "" {
			source = SourceRepo
		}
		displayStringField(w, "default_branch", cfg.DefaultBranch, source, useColor, indent+2)
	}

	if cfg.Merge != nil {
		displayMergeConfig(w, loaded, useColor, indent+2)
	}

	if cfg.Dependabot != nil {
		displayDependabotSettingsConfig(w, loaded, useColor, indent+2)
	}
}

func displayMergeConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "merge:")

	cfg := loaded.Config.Checks.Settings.Merge
	var repoMerge, ownerMerge *MergeConfig
	if loaded.RepoConfig != nil && loaded.RepoConfig.Checks.Settings != nil {
		repoMerge = loaded.RepoConfig.Checks.Settings.Merge
	}
	if loaded.OwnerConfig != nil && loaded.OwnerConfig.Checks.Settings != nil {
		ownerMerge = loaded.OwnerConfig.Checks.Settings.Merge
	}

	displayBoolField(w, "allow_merge_commit", cfg.AllowMergeCommit, getMergeBoolSource(repoMerge, ownerMerge, "AllowMergeCommit"), useColor, indent+2)
	displayBoolField(w, "allow_squash_merge", cfg.AllowSquashMerge, getMergeBoolSource(repoMerge, ownerMerge, "AllowSquashMerge"), useColor, indent+2)
	displayBoolField(w, "allow_rebase_merge", cfg.AllowRebaseMerge, getMergeBoolSource(repoMerge, ownerMerge, "AllowRebaseMerge"), useColor, indent+2)
	displayBoolField(w, "allow_auto_merge", cfg.AllowAutoMerge, getMergeBoolSource(repoMerge, ownerMerge, "AllowAutoMerge"), useColor, indent+2)
	displayBoolField(w, "delete_branch_on_merge", cfg.DeleteBranchOnMerge, getMergeBoolSource(repoMerge, ownerMerge, "DeleteBranchOnMerge"), useColor, indent+2)
	displayBoolField(w, "always_suggest_updating_pull_request_branches", cfg.AlwaysSuggestUpdatingPullRequestBranches, getMergeBoolSource(repoMerge, ownerMerge, "AlwaysSuggestUpdatingPullRequestBranches"), useColor, indent+2)
}

func displayDependabotSettingsConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "dependabot:")

	cfg := loaded.Config.Checks.Settings.Dependabot
	var repoDependabot, ownerDependabot *DependabotSettingsConfig
	if loaded.RepoConfig != nil && loaded.RepoConfig.Checks.Settings != nil {
		repoDependabot = loaded.RepoConfig.Checks.Settings.Dependabot
	}
	if loaded.OwnerConfig != nil && loaded.OwnerConfig.Checks.Settings != nil {
		ownerDependabot = loaded.OwnerConfig.Checks.Settings.Dependabot
	}

	displayBoolField(w, "alerts", cfg.Alerts, getDependabotBoolSource(repoDependabot, ownerDependabot, "Alerts"), useColor, indent+2)
	displayBoolField(w, "security_updates", cfg.SecurityUpdates, getDependabotBoolSource(repoDependabot, ownerDependabot, "SecurityUpdates"), useColor, indent+2)
}

func displayActionsConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "actions:")

	cfg := loaded.Config.Checks.Actions
	repo := getRepoActions(loaded)
	owner := getOwnerActions(loaded)

	displayBoolField(w, "require_pinned_versions", cfg.RequirePinnedVersions, getActionsBoolSource(repo, owner, "RequirePinnedVersions"), useColor, indent+2)
	displayBoolField(w, "require_timeout", cfg.RequireTimeout, getActionsBoolSource(repo, owner, "RequireTimeout"), useColor, indent+2)
	displayBoolField(w, "require_minimal_permissions", cfg.RequireMinimalPermissions, getActionsBoolSource(repo, owner, "RequireMinimalPermissions"), useColor, indent+2)

	if cfg.MaxTimeoutMinutes != nil {
		source := SourceOwner
		if repo != nil && repo.MaxTimeoutMinutes != nil {
			source = SourceRepo
		}
		displayIntField(w, "max_timeout_minutes", *cfg.MaxTimeoutMinutes, source, useColor, indent+2)
	}

	if len(cfg.RequiredWorkflows) > 0 {
		source := SourceOwner
		if repo != nil && repo.RequiredWorkflows != nil {
			source = SourceRepo
		}
		displayWorkflows(w, cfg.RequiredWorkflows, source, useColor, indent+2)
	}
}

func displayRulesetsConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "rulesets:")

	// Rulesets are arrays - repo replaces owner entirely
	source := SourceOwner
	if loaded.RepoConfig != nil && loaded.RepoConfig.Checks.Rulesets != nil {
		source = SourceRepo
	}

	for _, rs := range loaded.Config.Checks.Rulesets {
		displayRuleset(w, rs, source, useColor, indent+2, validator, result)
	}
}

func displayRuleset(w io.Writer, rs RulesetConfig, source Source, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "- name:", colorize(rs.Name, source, useColor))

	displayReferenceField(w, "reference", rs.Reference, source, useColor, indent+2, validator, result)
}

func displayFilesConfig(w io.Writer, loaded *LoadedConfig, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "files:")

	// Files are arrays - repo replaces owner entirely
	source := SourceOwner
	if loaded.RepoConfig != nil && loaded.RepoConfig.Checks.Files != nil {
		source = SourceRepo
	}

	for _, f := range loaded.Config.Checks.Files {
		displayFile(w, f, source, useColor, indent+2, validator, result)
	}
}

func displayFile(w io.Writer, f FileConfig, source Source, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "- name:", colorize(f.Name, source, useColor))

	displayReferenceField(w, "reference", f.Reference, source, useColor, indent+2, validator, result)

}

func displayWorkflows(w io.Writer, workflows []WorkflowConfig, source Source, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintln(w, "required_workflows:")

	for _, wf := range workflows {
		writeIndent(w, indent+2)
		_, _ = fmt.Fprintf(w, "- path: %s\n", colorize(wf.Path, source, useColor))
		if wf.Reference != "" {
			writeIndent(w, indent+4)
			_, _ = fmt.Fprintf(w, "reference: %s\n", colorize(wf.Reference, source, useColor))
		}
	}
}

// Helper functions

func writeIndent(w io.Writer, n int) {
	_, _ = fmt.Fprint(w, strings.Repeat(" ", n))
}

func displayBoolField(w io.Writer, name string, value *bool, source Source, useColor bool, indent int) {
	if value == nil {
		return
	}
	writeIndent(w, indent)
	_, _ = fmt.Fprintf(w, "%s: %s\n", name, colorize(strconv.FormatBool(*value), source, useColor))
}

func displayStringField(w io.Writer, name string, value string, source Source, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintf(w, "%s: %s\n", name, colorize(value, source, useColor))
}

func displayReferenceField(w io.Writer, name string, value string, source Source, useColor bool, indent int, validator ReferenceValidator, result *DisplayResult) {
	writeIndent(w, indent)

	// Validate the reference if validator is provided
	var statusIcon string
	isValid := true
	if validator != nil {
		err := validator(value)
		if err == nil {
			statusIcon = "✅"
		} else {
			isValid = false
			statusIcon = "❌"
			result.InvalidReferences = append(result.InvalidReferences, value)
		}
	}

	if isValid {
		_, _ = fmt.Fprintf(w, "%s: %s %s\n", name, colorize(value, source, useColor), statusIcon)
	} else {
		// For invalid references, show in red
		if useColor {
			_, _ = fmt.Fprintf(w, "%s: %s%s %s%s\n", name, colorRed, value, statusIcon, colorReset)
		} else {
			_, _ = fmt.Fprintf(w, "%s: %s %s [INVALID]\n", name, value, statusIcon)
		}
	}
}

func displayIntField(w io.Writer, name string, value int, source Source, useColor bool, indent int) {
	writeIndent(w, indent)
	_, _ = fmt.Fprintf(w, "%s: %s\n", name, colorize(strconv.Itoa(value), source, useColor))
}

func colorize(value string, source Source, useColor bool) string {
	if !useColor {
		switch source {
		case SourceRepo:
			return value + " [repo]"
		case SourceOwner:
			return value + " [owner]"
		default:
			return value
		}
	}

	switch source {
	case SourceRepo:
		return colorRepo + value + colorReset
	case SourceOwner:
		return colorOwner + value + colorReset
	default:
		return value
	}
}

// Source detection helpers

func getRepoSettings(loaded *LoadedConfig) *SettingsConfig {
	if loaded.RepoConfig != nil {
		return loaded.RepoConfig.Checks.Settings
	}
	return nil
}

func getOwnerSettings(loaded *LoadedConfig) *SettingsConfig {
	if loaded.OwnerConfig != nil {
		return loaded.OwnerConfig.Checks.Settings
	}
	return nil
}

func getRepoActions(loaded *LoadedConfig) *ActionsConfig {
	if loaded.RepoConfig != nil {
		return loaded.RepoConfig.Checks.Actions
	}
	return nil
}

func getOwnerActions(loaded *LoadedConfig) *ActionsConfig {
	if loaded.OwnerConfig != nil {
		return loaded.OwnerConfig.Checks.Actions
	}
	return nil
}

func getBoolSource(repo, owner *SettingsConfig, field string) Source {
	if repo != nil {
		v := reflect.ValueOf(repo).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceRepo
		}
	}
	if owner != nil {
		v := reflect.ValueOf(owner).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceOwner
		}
	}
	return SourceNone
}

func getMergeBoolSource(repo, owner *MergeConfig, field string) Source {
	if repo != nil {
		v := reflect.ValueOf(repo).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceRepo
		}
	}
	if owner != nil {
		v := reflect.ValueOf(owner).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceOwner
		}
	}
	return SourceNone
}

func getActionsBoolSource(repo, owner *ActionsConfig, field string) Source {
	if repo != nil {
		v := reflect.ValueOf(repo).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceRepo
		}
	}
	if owner != nil {
		v := reflect.ValueOf(owner).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceOwner
		}
	}
	return SourceNone
}

func getDependabotBoolSource(repo, owner *DependabotSettingsConfig, field string) Source {
	if repo != nil {
		v := reflect.ValueOf(repo).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceRepo
		}
	}
	if owner != nil {
		v := reflect.ValueOf(owner).Elem().FieldByName(field)
		if v.IsValid() && !v.IsNil() {
			return SourceOwner
		}
	}
	return SourceNone
}

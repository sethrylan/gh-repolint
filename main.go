package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/prompter"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/spf13/cobra"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/fix"
	"github.com/sethrylan/gh-repolint/github"
)

var (
	version = "dev"

	configFlag  string
	fixFlag     bool
	skipFlag    string
	verboseFlag bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gh-repolint",
		Short: "Lint GitHub repositories against organizational standards",
		Long: `gh-repolint is a GitHub CLI extension that validates repository
configuration against organizational standards defined in .repolint.yml`,
		RunE:         runLint,
		SilenceUsage: true,
	}

	rootCmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to config file (bypasses normal discovery)")
	rootCmd.Flags().BoolVar(&fixFlag, "fix", false, "Attempt to automatically fix issues")
	rootCmd.Flags().StringVar(&skipFlag, "skip", "", "Comma-separated list of checks to skip")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")

	// Config subcommand
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Validate and display the merged configuration",
		RunE:  runConfig,
	}
	rootCmd.AddCommand(configCmd)

	// Init subcommand
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive wizard to generate a starter .repolint.yaml",
		RunE:  runInit,
	}
	rootCmd.AddCommand(initCmd)

	// Version subcommand
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gh-repolint version %s\n", version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runLint(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Create GitHub client
	client, err := github.NewClient(repo.Owner, repo.Name, verboseFlag)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Check permissions
	if permErr := client.CheckPermissions(); permErr != nil {
		return permErr
	}

	// Load configuration
	loader := config.NewLoader(client)
	var loadedConfig *config.LoadedConfig
	if configFlag != "" {
		loadedConfig, err = loader.LoadFromFile(configFlag)
	} else {
		loadedConfig, err = loader.Load()
	}
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Parse skip flag
	var skip []string
	if skipFlag != "" {
		skip = strings.Split(skipFlag, ",")
		for i := range skip {
			skip[i] = strings.TrimSpace(skip[i])
		}
	}

	// Run checks
	runner := checks.NewRunner(client, loadedConfig.Config, verboseFlag)
	issues, err := runner.Run(ctx, skip)
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	// If no issues, report success
	if len(issues) == 0 {
		printSuccess(runner, verboseFlag)
		return nil
	}

	// If --fix, attempt to fix issues
	if fixFlag {
		return handleFix(ctx, client, loadedConfig.Config, issues)
	}

	// Report issues
	printIssues(issues)
	return fmt.Errorf("found %d issue(s)", len(issues))
}

func handleFix(ctx context.Context, client *github.Client, cfg *config.Config, issues []checks.Issue) error {
	orchestrator := fix.NewOrchestrator(client, cfg, verboseFlag)
	results, err := orchestrator.Fix(ctx, issues)
	if err != nil {
		return fmt.Errorf("fix failed: %w", err)
	}

	// Report results
	fixedCount := 0
	unfixedIssues := []checks.Issue{}

	for _, result := range results {
		if result.Fixed {
			fixedCount++
			fmt.Printf("  Fixed: [%s] %s\n", result.Issue.Name, result.Issue.Message)
		} else {
			unfixedIssues = append(unfixedIssues, result.Issue)
			if result.Error != nil {
				fmt.Printf("  Could not fix: [%s] %s (%s)\n", result.Issue.Name, result.Issue.Message, result.Error)
			} else {
				fmt.Printf("  Could not fix: [%s] %s (requires manual intervention)\n", result.Issue.Name, result.Issue.Message)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Fixed %d of %d issues\n", fixedCount, len(issues))

	if len(unfixedIssues) > 0 {
		return fmt.Errorf("%d issue(s) require manual intervention", len(unfixedIssues))
	}

	fmt.Println("All checks passed")
	return nil
}

func printSuccess(runner *checks.Runner, verbose bool) {
	fmt.Println("All checks passed")

	if verbose {
		for _, status := range runner.GetCheckStatuses() {
			if status.Skipped {
				fmt.Printf("  %s: skipped\n", status.Name)
			} else {
				fmt.Printf("  %s: validated\n", status.Name)
			}
		}
	}
}

func printIssues(issues []checks.Issue) {
	fmt.Println("Repository validation failed:")
	fixableCount := 0
	for _, issue := range issues {
		fixable := ""
		if issue.Fixable {
			fixable = " (fixable)"
			fixableCount++
		}
		fmt.Printf("  [%s] %s%s\n", issue.Name, issue.Message, fixable)
	}
	fmt.Println()
	if fixableCount > 0 {
		fmt.Printf("Run with --fix to automatically fix %d issue(s)\n", fixableCount)
	}
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Create GitHub client
	client, err := github.NewClient(repo.Owner, repo.Name, verboseFlag)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Load configuration
	loader := config.NewLoader(client)
	var loadedConfig *config.LoadedConfig
	if configFlag != "" {
		loadedConfig, err = loader.LoadFromFile(configFlag)
	} else {
		loadedConfig, err = loader.Load()
	}
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Check if terminal supports colors
	terminal := term.FromEnv()
	useColor := terminal.IsTerminalOutput()

	// Create reference validator
	validator := func(reference string) error {
		_, err := github.ResolveReferenceFile(reference, client)
		return err
	}

	// Display configuration with validation
	result := config.DisplayConfig(os.Stdout, loadedConfig, useColor, validator)

	// Check for invalid references
	if len(result.InvalidReferences) > 0 {
		fmt.Println()
		return fmt.Errorf("found %d invalid reference(s)", len(result.InvalidReferences))
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current repository for owner info
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Check if config already exists (check all supported extensions)
	var existingConfig string
	for _, name := range config.ConfigFileNames {
		if _, statErr := os.Stat(name); statErr == nil {
			existingConfig = name
			break
		}
	}
	if existingConfig != "" {
		fmt.Printf("Warning: %s already exists\n", existingConfig)
		p := prompter.New(os.Stdin, os.Stdout, os.Stderr)

		overwrite, confirmErr := p.Confirm("Overwrite existing file?", false)
		if confirmErr != nil {
			return confirmErr
		}
		if !overwrite {
			return nil
		}
	}

	p := prompter.New(os.Stdin, os.Stdout, os.Stderr)

	cfg := &config.Config{
		Checks: config.ChecksConfig{},
	}

	// Repository settings
	skipRepo, err := p.Confirm("Skip repository settings check?", false)
	if err != nil {
		return err
	}
	if !skipRepo {
		settingsCfg, settingsErr := promptSettingsConfig(p)
		if settingsErr != nil {
			return settingsErr
		}
		cfg.Checks.Settings = settingsCfg
	}

	// Actions check
	skipActions, err := p.Confirm("Skip actions/workflows check?", false)
	if err != nil {
		return err
	}
	if !skipActions {
		actionsCfg, actionsErr := promptActionsConfig(p)
		if actionsErr != nil {
			return actionsErr
		}
		cfg.Checks.Actions = actionsCfg
	}

	// Dependabot check
	skipDependabot, err := p.Confirm("Skip dependabot check?", false)
	if err != nil {
		return err
	}
	if !skipDependabot {
		dependabotFileCfg, dependabotErr := promptDependabotFileConfig(p, repo.Owner)
		if dependabotErr != nil {
			return dependabotErr
		}
		cfg.Checks.Files = append(cfg.Checks.Files, *dependabotFileCfg)
	}

	// Rulesets check
	skipRulesets, err := p.Confirm("Skip rulesets check?", false)
	if err != nil {
		return err
	}
	if !skipRulesets {
		rulesetsCfg, rulesetsErr := promptRulesetsConfig(p, repo.Owner)
		if rulesetsErr != nil {
			return rulesetsErr
		}
		cfg.Checks.Rulesets = rulesetsCfg
	}

	// Files check
	skipFiles, err := p.Confirm("Skip files check?", false)
	if err != nil {
		return err
	}
	if !skipFiles {
		filesCfg, err := promptFilesConfig(p, repo.Owner)
		if err != nil {
			return err
		}
		cfg.Checks.Files = append(cfg.Checks.Files, filesCfg...)
	}

	// Generate YAML
	content := generateConfigYAML(cfg)

	// Write file (always use the first/default config filename)
	if err := os.WriteFile(config.ConfigFileNames[0], []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created %s\n", config.ConfigFileNames[0])
	return nil
}

func promptSettingsConfig(p *prompter.Prompter) (*config.SettingsConfig, error) {
	cfg := &config.SettingsConfig{}

	issues, err := p.Confirm("Issues enabled?", true)
	if err != nil {
		return nil, err
	}
	cfg.Issues = &issues

	wiki, err := p.Confirm("Wiki enabled?", false)
	if err != nil {
		return nil, err
	}
	cfg.Wiki = &wiki

	projects, err := p.Confirm("Projects enabled?", false)
	if err != nil {
		return nil, err
	}
	cfg.Projects = &projects

	actionsApprove, err := p.Confirm("Allow actions to approve PRs?", false)
	if err != nil {
		return nil, err
	}
	cfg.AllowActionsToApprovePRs = &actionsApprove

	cfg.Merge = &config.MergeConfig{}
	mergeCommit, err := p.Confirm("Allow merge commits?", false)
	if err != nil {
		return nil, err
	}
	cfg.Merge.AllowMergeCommit = &mergeCommit

	squash, err := p.Confirm("Allow squash merge?", true)
	if err != nil {
		return nil, err
	}
	cfg.Merge.AllowSquashMerge = &squash

	rebase, err := p.Confirm("Allow rebase merge?", false)
	if err != nil {
		return nil, err
	}
	cfg.Merge.AllowRebaseMerge = &rebase

	autoMerge, err := p.Confirm("Allow auto-merge?", true)
	if err != nil {
		return nil, err
	}
	cfg.Merge.AllowAutoMerge = &autoMerge

	deleteBranch, err := p.Confirm("Delete branch on merge?", true)
	if err != nil {
		return nil, err
	}
	cfg.Merge.DeleteBranchOnMerge = &deleteBranch

	defaultBranch, err := p.Input("Default branch name:", "main")
	if err != nil {
		return nil, err
	}
	cfg.DefaultBranch = defaultBranch

	// Dependabot settings
	cfg.Dependabot = &config.DependabotSettingsConfig{}
	alerts, err := p.Confirm("Enable Dependabot alerts?", true)
	if err != nil {
		return nil, err
	}
	cfg.Dependabot.Alerts = &alerts

	securityUpdates, err := p.Confirm("Enable Dependabot security updates?", true)
	if err != nil {
		return nil, err
	}
	cfg.Dependabot.SecurityUpdates = &securityUpdates

	// Pull request creation policy
	prPolicies := []string{"all", "collaborators"}
	prPolicyIdx, err := p.Select("Pull request creation policy:", "all", prPolicies)
	if err != nil {
		return nil, err
	}
	cfg.PullRequestCreationPolicy = prPolicies[prPolicyIdx]

	return cfg, nil
}

func promptActionsConfig(p *prompter.Prompter) (*config.ActionsConfig, error) {
	cfg := &config.ActionsConfig{}

	pinnedVersions, err := p.Confirm("Require pinned action versions (SHA)?", true)
	if err != nil {
		return nil, err
	}
	cfg.RequirePinnedVersions = &pinnedVersions

	timeout, err := p.Confirm("Require timeout on jobs?", false)
	if err != nil {
		return nil, err
	}
	cfg.RequireTimeout = &timeout

	if timeout {
		maxTimeout := 60
		cfg.MaxTimeoutMinutes = &maxTimeout
	}

	minPerms, err := p.Confirm("Require minimal permissions?", true)
	if err != nil {
		return nil, err
	}
	cfg.RequireMinimalPermissions = &minPerms

	return cfg, nil
}

func promptDependabotFileConfig(p *prompter.Prompter, owner string) (*config.FileConfig, error) {
	cfg := &config.FileConfig{}
	cfg.Name = ".github/dependabot.yml"

	defaultRef := fmt.Sprintf("%s/%s/.repolint/dependabot.yml", owner, owner)
	reference, err := p.Input("Dependabot file reference:", defaultRef)
	if err != nil {
		return nil, err
	}
	cfg.Reference = reference

	return cfg, nil
}

func promptRulesetsConfig(p *prompter.Prompter, owner string) ([]config.RulesetConfig, error) {
	cfg := &config.RulesetConfig{}

	name, err := p.Input("Ruleset name:", "main")
	if err != nil {
		return nil, err
	}
	cfg.Name = name

	defaultRef := fmt.Sprintf("%s/%s/.repolint/ruleset.json", owner, owner)
	reference, err := p.Input("Ruleset file reference:", defaultRef)
	if err != nil {
		return nil, err
	}
	cfg.Reference = reference

	return []config.RulesetConfig{*cfg}, nil
}

func promptFilesConfig(p *prompter.Prompter, owner string) ([]config.FileConfig, error) {
	var files []config.FileConfig

	for {
		cfg := &config.FileConfig{}

		name, err := p.Input("File path to validate (empty to skip):", ".github/workflows/ci.yml")
		if err != nil {
			return nil, err
		}
		if name == "" {
			break
		}
		cfg.Name = name

		defaultRef := fmt.Sprintf("%s/%s/.repolint/%s", owner, owner, strings.ReplaceAll(name, ".github/", ""))
		reference, err := p.Input("Reference file (empty to skip):", defaultRef)
		if err != nil {
			return nil, err
		}
		if reference == "" {
			break
		}

		cfg.Reference = reference

		files = append(files, *cfg)

		addMore, err := p.Confirm("Add another file?", false)
		if err != nil {
			return nil, err
		}
		if !addMore {
			break
		}
	}

	return files, nil
}

func generateConfigYAML(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString("# gh-repolint configuration\n")
	sb.WriteString("# See https://github.com/sethrylan/gh-repolint for documentation\n\n")
	sb.WriteString("checks:\n")

	if cfg.Checks.Settings != nil {
		sb.WriteString("  settings:\n")
		if cfg.Checks.Settings.Issues != nil {
			fmt.Fprintf(&sb, "    issues: %t\n", *cfg.Checks.Settings.Issues)
		}
		if cfg.Checks.Settings.Wiki != nil {
			fmt.Fprintf(&sb, "    wiki: %t\n", *cfg.Checks.Settings.Wiki)
		}
		if cfg.Checks.Settings.Projects != nil {
			fmt.Fprintf(&sb, "    projects: %t\n", *cfg.Checks.Settings.Projects)
		}
		if cfg.Checks.Settings.AllowActionsToApprovePRs != nil {
			fmt.Fprintf(&sb, "    allow_actions_to_approve_prs: %t\n", *cfg.Checks.Settings.AllowActionsToApprovePRs)
		}
		if cfg.Checks.Settings.PullRequestCreationPolicy != "" {
			fmt.Fprintf(&sb, "    pull_request_creation_policy: \"%s\"\n", cfg.Checks.Settings.PullRequestCreationPolicy)
		}
		if cfg.Checks.Settings.DefaultBranch != "" {
			fmt.Fprintf(&sb, "    default_branch: \"%s\"\n", cfg.Checks.Settings.DefaultBranch)
		}
		if cfg.Checks.Settings.Merge != nil {
			sb.WriteString("    merge:\n")
			m := cfg.Checks.Settings.Merge
			if m.AllowMergeCommit != nil {
				fmt.Fprintf(&sb, "      allow_merge_commit: %t\n", *m.AllowMergeCommit)
			}
			if m.AllowSquashMerge != nil {
				fmt.Fprintf(&sb, "      allow_squash_merge: %t\n", *m.AllowSquashMerge)
			}
			if m.AllowRebaseMerge != nil {
				fmt.Fprintf(&sb, "      allow_rebase_merge: %t\n", *m.AllowRebaseMerge)
			}
			if m.AllowAutoMerge != nil {
				fmt.Fprintf(&sb, "      allow_auto_merge: %t\n", *m.AllowAutoMerge)
			}
			if m.DeleteBranchOnMerge != nil {
				fmt.Fprintf(&sb, "      delete_branch_on_merge: %t\n", *m.DeleteBranchOnMerge)
			}
		}
		if cfg.Checks.Settings.Dependabot != nil {
			sb.WriteString("    dependabot:\n")
			d := cfg.Checks.Settings.Dependabot
			if d.Alerts != nil {
				fmt.Fprintf(&sb, "      alerts: %t\n", *d.Alerts)
			}
			if d.SecurityUpdates != nil {
				fmt.Fprintf(&sb, "      security_updates: %t\n", *d.SecurityUpdates)
			}
		}
		sb.WriteString("\n")
	}

	if cfg.Checks.Actions != nil {
		sb.WriteString("  actions:\n")
		if cfg.Checks.Actions.RequirePinnedVersions != nil {
			fmt.Fprintf(&sb, "    require_pinned_versions: %t\n", *cfg.Checks.Actions.RequirePinnedVersions)
		}
		if cfg.Checks.Actions.RequireTimeout != nil {
			fmt.Fprintf(&sb, "    require_timeout: %t\n", *cfg.Checks.Actions.RequireTimeout)
		}
		if cfg.Checks.Actions.MaxTimeoutMinutes != nil {
			fmt.Fprintf(&sb, "    max_timeout_minutes: %d\n", *cfg.Checks.Actions.MaxTimeoutMinutes)
		}
		if cfg.Checks.Actions.RequireMinimalPermissions != nil {
			fmt.Fprintf(&sb, "    require_minimal_permissions: %t\n", *cfg.Checks.Actions.RequireMinimalPermissions)
		}
		sb.WriteString("\n")
	}

	if len(cfg.Checks.Rulesets) > 0 {
		sb.WriteString("  rulesets:\n")
		for _, rs := range cfg.Checks.Rulesets {
			fmt.Fprintf(&sb, "    - name: \"%s\"\n", rs.Name)
			fmt.Fprintf(&sb, "      reference: \"%s\"\n", rs.Reference)
		}
	}

	if len(cfg.Checks.Files) > 0 {
		sb.WriteString("  files:\n")
		for _, f := range cfg.Checks.Files {
			fmt.Fprintf(&sb, "    - name: \"%s\"\n", f.Name)
			fmt.Fprintf(&sb, "      reference: \"%s\"\n", f.Reference)
		}
	}

	return sb.String()
}

package github

// Repository represents a GitHub repository
type Repository struct {
	Name                string `json:"name"`
	FullName            string `json:"full_name"`
	DefaultBranch       string `json:"default_branch"`
	Archived            bool   `json:"archived"`
	HasIssues           bool   `json:"has_issues"`
	HasWiki             bool   `json:"has_wiki"`
	HasProjects         bool   `json:"has_projects"`
	HasDiscussions      bool   `json:"has_discussions"`
	PullRequestCreationPolicy string `json:"pull_request_creation_policy"`
	AllowMergeCommit          bool   `json:"allow_merge_commit"`
	AllowSquashMerge    bool   `json:"allow_squash_merge"`
	AllowRebaseMerge    bool   `json:"allow_rebase_merge"`
	AllowAutoMerge      bool   `json:"allow_auto_merge"`
	DeleteBranchOnMerge bool   `json:"delete_branch_on_merge"`
	AllowUpdateBranch   bool   `json:"allow_update_branch"`
}

// ActionsPermissions represents repository actions permissions
type ActionsPermissions struct {
	Enabled            bool     `json:"enabled"`
	AllowedActions     string   `json:"allowed_actions"`
	GithubOwnedAllowed bool     `json:"github_owned_allowed"`
	VerifiedAllowed    bool     `json:"verified_allowed"`
	PatternsAllowed    []string `json:"patterns_allowed"`
}

// WorkflowPermissions represents workflow permissions settings
type WorkflowPermissions struct {
	DefaultWorkflowPermissions   string `json:"default_workflow_permissions"`
	CanApprovePullRequestReviews bool   `json:"can_approve_pull_request_reviews"`
}

// Ruleset represents a GitHub repository ruleset
type Ruleset struct {
	ID           int                `json:"id"`
	Name         string             `json:"name"`
	Target       string             `json:"target"`
	Enforcement  string             `json:"enforcement"`
	Conditions   *RulesetConditions `json:"conditions,omitempty"`
	Rules        []RulesetRule      `json:"rules"`
	BypassActors []BypassActor      `json:"bypass_actors,omitempty"`
}

// RulesetConditions represents the conditions for a ruleset
type RulesetConditions struct {
	RefName *RefNameCondition `json:"ref_name,omitempty"`
}

// RefNameCondition represents branch/tag conditions
type RefNameCondition struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// RulesetRule represents a single rule in a ruleset
type RulesetRule struct {
	Type       string         `json:"type"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// BypassActor represents an actor that can bypass the ruleset
type BypassActor struct {
	ActorID    int    `json:"actor_id"`
	ActorType  string `json:"actor_type"`
	BypassMode string `json:"bypass_mode"`
}

// FileContent represents a file's content from GitHub API
type FileContent struct {
	Type        string `json:"type"`
	Encoding    string `json:"encoding"`
	Size        int    `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
	GitURL      string `json:"git_url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

// DependabotConfig represents the dependabot.yml structure
type DependabotConfig struct {
	Version int                        `yaml:"version"`
	Updates []DependabotUpdate         `yaml:"updates"`
	Groups  map[string]DependabotGroup `yaml:"groups,omitempty"`
}

// DependabotUpdate represents a single update configuration
type DependabotUpdate struct {
	PackageEcosystem string                     `yaml:"package-ecosystem"`
	Directory        string                     `yaml:"directory"`
	Schedule         DependabotSchedule         `yaml:"schedule"`
	CommitMessage    *DependabotCommitMsg       `yaml:"commit-message,omitempty"`
	Assignees        []string                   `yaml:"assignees,omitempty"`
	Reviewers        []string                   `yaml:"reviewers,omitempty"`
	Labels           []string                   `yaml:"labels,omitempty"`
	Groups           map[string]DependabotGroup `yaml:"groups,omitempty"`
}

// DependabotGroup represents a grouping configuration for Dependabot updates
type DependabotGroup struct {
	AppliesTo      string   `yaml:"applies-to,omitempty"`
	Patterns       []string `yaml:"patterns,omitempty"`
	DependencyType string   `yaml:"dependency-type,omitempty"`
	UpdateTypes    []string `yaml:"update-types,omitempty"`
}

// DependabotSchedule represents the update schedule
type DependabotSchedule struct {
	Interval string `yaml:"interval"`
	Day      string `yaml:"day,omitempty"`
	Time     string `yaml:"time,omitempty"`
	Timezone string `yaml:"timezone,omitempty"`
}

// DependabotCommitMsg represents commit message settings
type DependabotCommitMsg struct {
	Prefix            string `yaml:"prefix,omitempty"`
	PrefixDevelopment string `yaml:"prefix-development,omitempty"`
	Include           string `yaml:"include,omitempty"`
}

// AutomatedSecurityFixes represents the status of automated security fixes (Dependabot security updates)
type AutomatedSecurityFixes struct {
	Enabled bool `json:"enabled"`
	Paused  bool `json:"paused"`
}

// Workflow represents a GitHub Actions workflow file
type Workflow struct {
	Name        string                 `yaml:"name,omitempty"`
	On          any                    `yaml:"on"`
	Permissions any                    `yaml:"permissions,omitempty"`
	Env         map[string]string      `yaml:"env,omitempty"`
	Jobs        map[string]WorkflowJob `yaml:"jobs"`
}

// WorkflowJob represents a job in a workflow
type WorkflowJob struct {
	Name           string            `yaml:"name,omitempty"`
	RunsOn         any               `yaml:"runs-on"`
	Permissions    any               `yaml:"permissions,omitempty"`
	TimeoutMinutes int               `yaml:"timeout-minutes,omitempty"`
	Steps          []WorkflowStep    `yaml:"steps"`
	Needs          any               `yaml:"needs,omitempty"`
	If             string            `yaml:"if,omitempty"`
	Env            map[string]string `yaml:"env,omitempty"`
	Strategy       *JobStrategy      `yaml:"strategy,omitempty"`
}

// JobStrategy represents the strategy for a job
type JobStrategy struct {
	Matrix      map[string]any `yaml:"matrix,omitempty"`
	FailFast    *bool          `yaml:"fail-fast,omitempty"`
	MaxParallel int            `yaml:"max-parallel,omitempty"`
}

// WorkflowStep represents a step in a workflow job
type WorkflowStep struct {
	Name             string            `yaml:"name,omitempty"`
	ID               string            `yaml:"id,omitempty"`
	Uses             string            `yaml:"uses,omitempty"`
	Run              string            `yaml:"run,omitempty"`
	With             map[string]string `yaml:"with,omitempty"`
	Env              map[string]string `yaml:"env,omitempty"`
	If               string            `yaml:"if,omitempty"`
	WorkingDirectory string            `yaml:"working-directory,omitempty"`
}

// RulesetCreateRequest represents a request to create a ruleset
type RulesetCreateRequest struct {
	Name         string             `json:"name"`
	Target       string             `json:"target"`
	Enforcement  string             `json:"enforcement"`
	Conditions   *RulesetConditions `json:"conditions,omitempty"`
	Rules        []RulesetRule      `json:"rules"`
	BypassActors []BypassActor      `json:"bypass_actors,omitempty"`
}

// RepoUpdateRequest represents a request to update repository settings
type RepoUpdateRequest struct {
	HasIssues           *bool `json:"has_issues,omitempty"`
	HasWiki             *bool `json:"has_wiki,omitempty"`
	HasProjects         *bool `json:"has_projects,omitempty"`
	HasDiscussions      *bool `json:"has_discussions,omitempty"`
	PullRequestCreationPolicy *string `json:"pull_request_creation_policy,omitempty"`
	AllowMergeCommit          *bool   `json:"allow_merge_commit,omitempty"`
	AllowSquashMerge          *bool   `json:"allow_squash_merge,omitempty"`
	AllowRebaseMerge    *bool `json:"allow_rebase_merge,omitempty"`
	AllowAutoMerge      *bool `json:"allow_auto_merge,omitempty"`
	DeleteBranchOnMerge *bool `json:"delete_branch_on_merge,omitempty"`
	AllowUpdateBranch   *bool `json:"allow_update_branch,omitempty"`
}

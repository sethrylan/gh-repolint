package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// ActionsCheck validates GitHub Actions workflows
type ActionsCheck struct {
	client  *github.Client
	config  *config.ActionsConfig
	verbose bool
}

// NewActionsCheck creates a new actions check
func NewActionsCheck(client *github.Client, cfg *config.ActionsConfig, verbose bool) *ActionsCheck {
	return &ActionsCheck{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Type returns the check type
func (c *ActionsCheck) Type() CheckType {
	return CheckTypeActions
}

// Name returns the check name
func (c *ActionsCheck) Name() string {
	return "actions"
}

// Run executes the actions check
func (c *ActionsCheck) Run(ctx context.Context) ([]Issue, error) {
	if c.config == nil {
		return nil, nil
	}

	var issues []Issue

	// Check required workflows
	for _, wfConfig := range c.config.RequiredWorkflows {
		wfIssues, err := c.checkWorkflow(wfConfig)
		if err != nil {
			return nil, err
		}
		issues = append(issues, wfIssues...)
	}

	// Check all workflow files for general rules
	workflowFiles, err := c.findWorkflowFiles()
	if err != nil {
		return nil, err
	}

	for _, wfPath := range workflowFiles {
		wfIssues, err := c.checkWorkflowRules(wfPath)
		if err != nil {
			return nil, err
		}
		issues = append(issues, wfIssues...)
	}

	return issues, nil
}

func (c *ActionsCheck) checkWorkflow(wfConfig config.WorkflowConfig) ([]Issue, error) {
	var issues []Issue

	// Check if workflow file exists
	if !c.client.FileExists(wfConfig.Path) {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Required workflow '%s' is missing", wfConfig.Path),
			Fixable: wfConfig.Reference != "",
			Data: map[string]string{
				DataKeyFileName:  wfConfig.Path,
				DataKeyReference: wfConfig.Reference,
			},
		})
		return issues, nil
	}

	// If reference is specified, check content matches
	if wfConfig.Reference != "" {
		matchIssues, err := c.checkWorkflowReference(wfConfig)
		if err != nil {
			return nil, err
		}
		issues = append(issues, matchIssues...)
	}

	return issues, nil
}

func (c *ActionsCheck) checkWorkflowReference(wfConfig config.WorkflowConfig) ([]Issue, error) {
	var issues []Issue

	// Parse reference: owner/repo/path
	parts := strings.SplitN(wfConfig.Reference, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid reference format: %s (expected owner/repo/path)", wfConfig.Reference)
	}

	refOwner, refRepo, refPath := parts[0], parts[1], parts[2]

	// Fetch reference content
	refContent, err := c.client.GetRemoteFileContent(refOwner, refRepo, refPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reference workflow: %w", err)
	}

	interpolatedRef, err := c.client.HydrateTemplate(refContent)
	if err != nil {
		return nil, fmt.Errorf("failed to interpolate reference template: %w", err)
	}

	// Read actual workflow content
	actualContent, err := c.client.GetLocalFileContent(wfConfig.Path)
	if err != nil {
		return nil, err
	}

	// Compare YAML structures (not raw content)
	if !yamlEqual(string(interpolatedRef), string(actualContent)) {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("Workflow '%s' does not match reference '%s'", wfConfig.Path, wfConfig.Reference),
			Fixable: true,
			Data: map[string]string{
				DataKeyFileName:  wfConfig.Path,
				DataKeyReference: wfConfig.Reference,
			},
		})
	}

	return issues, nil
}

func (c *ActionsCheck) findWorkflowFiles() ([]string, error) {
	workflowDir := ".github/workflows"

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			files = append(files, filepath.Join(workflowDir, name))
		}
	}

	return files, nil
}

func (c *ActionsCheck) checkWorkflowRules(wfPath string) ([]Issue, error) {
	var issues []Issue

	wf, content, err := github.ReadLocalWorkflowFile(wfPath)
	if err != nil {
		return nil, err
	}

	// Check pinned versions
	if c.config.RequirePinnedVersions != nil && *c.config.RequirePinnedVersions {
		pinIssues := c.checkPinnedVersions(wfPath, string(content))
		issues = append(issues, pinIssues...)
	}

	// Check timeout
	if c.config.RequireTimeout != nil && *c.config.RequireTimeout {
		timeoutIssues := c.checkTimeout(wfPath, wf)
		issues = append(issues, timeoutIssues...)
	}

	// Check minimal permissions
	if c.config.RequireMinimalPermissions != nil && *c.config.RequireMinimalPermissions {
		permIssues := c.checkPermissions(wfPath, wf)
		issues = append(issues, permIssues...)
	}

	return issues, nil
}

func (c *ActionsCheck) checkPinnedVersions(wfPath, content string) []Issue {
	var issues []Issue

	// Regex to match uses: statements
	usesRegex := regexp.MustCompile(`uses:\s*([^\s@]+)@([^\s]+)`)
	matches := usesRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		action := match[1]
		version := match[2]

		// Skip first-party actions (actions/*, github/*, cli/*, and dependabot/*)
		if strings.HasPrefix(action, "actions/") ||
			strings.HasPrefix(action, "github/") ||
			strings.HasPrefix(action, "cli/") ||
			strings.HasPrefix(action, "dependabot/") {
			continue
		}

		// Check if version is a SHA (40 hex characters)
		if !isSHA(version) {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Action '%s@%s' in '%s' is not pinned to a SHA", action, version, wfPath),
				Fixable: false,
			})
		}
	}

	return issues
}

func (c *ActionsCheck) checkTimeout(wfPath string, wf *github.Workflow) []Issue {
	var issues []Issue

	for jobName, job := range wf.Jobs {
		if job.TimeoutMinutes == 0 {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Job '%s' in '%s' does not have timeout-minutes set", jobName, wfPath),
				Fixable: false,
			})
		} else if c.config.MaxTimeoutMinutes != nil && job.TimeoutMinutes > *c.config.MaxTimeoutMinutes {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Job '%s' in '%s' has timeout-minutes (%d) exceeding maximum (%d)", jobName, wfPath, job.TimeoutMinutes, *c.config.MaxTimeoutMinutes),
				Fixable: false,
			})
		}
	}

	return issues
}

func (c *ActionsCheck) checkPermissions(wfPath string, wf *github.Workflow) []Issue {
	var issues []Issue

	// Check if workflow-level permissions is set
	if wf.Permissions == nil {
		hasJobPermissions := true
		for _, job := range wf.Jobs {
			if job.Permissions == nil {
				hasJobPermissions = false
				break
			}
		}
		if !hasJobPermissions {
			issues = append(issues, Issue{
				Type:    c.Type(),
				Name:    c.Name(),
				Message: fmt.Sprintf("Workflow '%s' does not declare permissions at workflow or job level", wfPath),
				Fixable: false,
			})
		}
	}

	return issues
}

func yamlEqual(a, b string) bool {
	var aData, bData any
	if err := yaml.Unmarshal([]byte(a), &aData); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(b), &bData); err != nil {
		return false
	}

	aBytes, _ := yaml.Marshal(aData)
	bBytes, _ := yaml.Marshal(bData)

	return string(aBytes) == string(bBytes)
}

func isSHA(version string) bool {
	if len(version) != 40 {
		return false
	}
	for _, c := range version {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

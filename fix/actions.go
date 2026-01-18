package fix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// ActionsFixer fixes actions/workflow issues
type ActionsFixer struct {
	client  *github.Client
	config  *config.ActionsConfig
	verbose bool
}

// NewActionsFixer creates a new actions fixer
func NewActionsFixer(client *github.Client, cfg *config.ActionsConfig, verbose bool) *ActionsFixer {
	return &ActionsFixer{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Name returns the fixer name
func (f *ActionsFixer) Name() string {
	return "actions"
}

// Fix attempts to fix an actions issue
func (f *ActionsFixer) Fix(ctx context.Context, issue checks.Issue) (*Result, error) {
	// Get workflow path from issue data
	workflowPath := issue.Data[checks.DataKeyFileName]
	if workflowPath == "" {
		return failedResult(issue, errors.New("issue data missing file_name"))
	}

	// Find the workflow config
	var wfConfig *config.WorkflowConfig
	for i := range f.config.RequiredWorkflows {
		if f.config.RequiredWorkflows[i].Path == workflowPath {
			wfConfig = &f.config.RequiredWorkflows[i]
			break
		}
	}

	if wfConfig == nil || wfConfig.Reference == "" {
		return failedResult(issue, fmt.Errorf("no reference specified for workflow %s", workflowPath))
	}

	// Fetch and write the reference content
	content, err := f.fetchAndInterpolateReference(wfConfig.Reference)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to fetch reference: %w", err))
	}

	if err := f.client.WriteFile(workflowPath, content); err != nil {
		return failedResult(issue, fmt.Errorf("failed to write workflow file: %w", err))
	}

	return successResult(issue)
}

func (f *ActionsFixer) fetchAndInterpolateReference(reference string) ([]byte, error) {
	// Parse reference: owner/repo/path
	parts := strings.SplitN(reference, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid reference format: %s", reference)
	}

	refOwner, refRepo, refPath := parts[0], parts[1], parts[2]

	// Fetch reference content
	content, err := f.client.GetRemoteFileContent(refOwner, refRepo, refPath)
	if err != nil {
		return nil, err
	}

	// Get repository info for template variables
	repo, err := f.client.GetRepository()
	if err != nil {
		return nil, err
	}

	// Interpolate template variables.
	// Uses simple string replacement instead of Go's text/template to avoid
	// conflicts with GitHub Actions expression syntax (${{ }}), which uses
	// similar curly-brace delimiters.
	tmplData := map[string]string{
		"owner":          f.client.Owner(),
		"repo":           f.client.Repo(),
		"default_branch": repo.DefaultBranch,
	}

	result := string(content)
	for key, value := range tmplData {
		result = strings.ReplaceAll(result, "{{ ."+key+" }}", value)
		result = strings.ReplaceAll(result, "{{."+key+"}}", value)
	}

	return []byte(result), nil
}

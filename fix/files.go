package fix

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethrylan/gh-repolint/checks"
	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// FilesFixer fixes file configuration issues
type FilesFixer struct {
	client  *github.Client
	configs []config.FileConfig
	verbose bool
}

// NewFilesFixer creates a new files fixer
func NewFilesFixer(client *github.Client, cfgs []config.FileConfig, verbose bool) *FilesFixer {
	return &FilesFixer{
		client:  client,
		configs: cfgs,
		verbose: verbose,
	}
}

// Name returns the fixer name
func (f *FilesFixer) Name() string {
	return "files"
}

// Fix attempts to fix a file issue
func (f *FilesFixer) Fix(ctx context.Context, issue checks.Issue) (*Result, error) {
	// Get file name from issue data
	fileName := issue.Data[checks.DataKeyFileName]
	if fileName == "" {
		return failedResult(issue, errors.New("issue data missing file_name"))
	}

	// Find the config for this file
	var cfg *config.FileConfig
	for i := range f.configs {
		if f.configs[i].Name == fileName {
			cfg = &f.configs[i]
			break
		}
	}

	if cfg == nil {
		return failedResult(issue, fmt.Errorf("no config found for file '%s'", fileName))
	}

	if cfg.Reference == "" {
		return failedResult(issue, fmt.Errorf("file '%s' has no reference specified", fileName))
	}

	// Fetch the reference file content
	refContent, err := github.ResolveReferenceFile(cfg.Reference, f.client)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to fetch reference file: %w", err))
	}

	// Hydrate reference file with template variables
	hydratedContent, err := f.client.HydrateTemplate(refContent)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to hydrate reference template: %w", err))
	}

	return f.writeFile(issue, cfg, hydratedContent)
}

func (f *FilesFixer) writeFile(issue checks.Issue, cfg *config.FileConfig, content []byte) (*Result, error) {
	err := f.client.WriteFile(cfg.Name, content)
	if err != nil {
		return failedResult(issue, fmt.Errorf("failed to write file: %w", err))
	}

	return successResult(issue)
}

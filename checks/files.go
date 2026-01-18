package checks

import (
	"bytes"
	"context"
	"fmt"

	"github.com/sethrylan/gh-repolint/config"
	"github.com/sethrylan/gh-repolint/github"
)

// FilesCheck validates that a file matches a reference
type FilesCheck struct {
	client  *github.Client
	config  *config.FileConfig
	verbose bool
}

// NewFilesCheck creates a new files check
func NewFilesCheck(client *github.Client, cfg *config.FileConfig, verbose bool) *FilesCheck {
	return &FilesCheck{
		client:  client,
		config:  cfg,
		verbose: verbose,
	}
}

// Type returns the check type
func (c *FilesCheck) Type() CheckType {
	return CheckTypeFiles
}

// Name returns the check name
func (c *FilesCheck) Name() string {
	return "files(" + c.config.Name + ")"
}

// Run executes the files check
func (c *FilesCheck) Run(ctx context.Context) ([]Issue, error) {
	if c.config == nil {
		return nil, nil
	}

	if c.config.Reference == "" {
		return nil, fmt.Errorf("file '%s' missing required reference field", c.config.Name)
	}

	var issues []Issue

	// Fetch the expected file content from reference
	expectedContent, err := github.ResolveReferenceFile(c.config.Reference, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reference file: %w", err)
	}

	// Hydrate reference file with template variables
	hydratedContent, err := c.client.HydrateTemplate(expectedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to hydrate reference template: %w", err)
	}

	// Fetch the actual file content from the working directory
	actualContent, err := c.client.GetLocalFileContent(c.config.Name)
	if err != nil {
		// File doesn't exist - this is an issue to report, not an error
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("File '%s' does not exist", c.config.Name),
			Fixable: true,
			Data: map[string]string{
				DataKeyFileName:  c.config.Name,
				DataKeyReference: c.config.Reference,
			},
		})
		return issues, nil //nolint:nilerr // Intentional: missing file is a reportable issue, not an error
	}

	// Compare the contents, ignoring trailing whitespace
	if !bytes.Equal(bytes.TrimSpace(actualContent), bytes.TrimSpace(hydratedContent)) {
		issues = append(issues, Issue{
			Type:    c.Type(),
			Name:    c.Name(),
			Message: fmt.Sprintf("File '%s' does not match reference '%s'", c.config.Name, c.config.Reference),
			Fixable: true,
			Data: map[string]string{
				DataKeyFileName:  c.config.Name,
				DataKeyReference: c.config.Reference,
			},
		})
	}

	return issues, nil
}

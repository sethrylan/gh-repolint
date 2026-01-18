package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a local file exists
func (c *Client) FileExists(filePath string) bool {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return false
		}
		fullPath = filepath.Join(cwd, filePath)
	}

	_, err := os.Stat(fullPath)
	return err == nil
}

// WriteFile writes content to a file in the repository (for fixes)
func (c *Client) WriteFile(filePath string, content []byte) error {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		fullPath = filepath.Join(cwd, filePath)
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(fullPath, content, 0600)
}

// GetLocalFileContent reads a file from the local repository
func (c *Client) GetLocalFileContent(filePath string) ([]byte, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fullPath := filepath.Join(cwd, filePath)
	return os.ReadFile(fullPath) //nolint:gosec // Reading user-specified files is intentional
}

// HydrateTemplate interpolates template variables in the content
// using the client's owner and repo as template data.
//
// Uses simple string replacement instead of Go's text/template to avoid
// conflicts with GitHub Actions expression syntax (${{ }}), which uses
// similar curly-brace delimiters.
func (c *Client) HydrateTemplate(content []byte) ([]byte, error) {
	result := string(content)
	result = strings.ReplaceAll(result, "{{ .owner }}", c.owner)
	result = strings.ReplaceAll(result, "{{.owner}}", c.owner)
	result = strings.ReplaceAll(result, "{{ .repo }}", c.repo)
	result = strings.ReplaceAll(result, "{{.repo}}", c.repo)
	return []byte(result), nil
}

// FetchReferenceRuleset fetches and parses a JSON ruleset from a reference file
// It first tries to read from the local filesystem, then falls back to remote repository lookup
func FetchReferenceRuleset(reference string, client *Client) (*Ruleset, error) {
	content, err := ResolveReferenceFile(reference, client)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var ruleset Ruleset
	if err := json.Unmarshal(content, &ruleset); err != nil {
		return nil, fmt.Errorf("failed to parse reference JSON: %w", err)
	}

	return &ruleset, nil
}

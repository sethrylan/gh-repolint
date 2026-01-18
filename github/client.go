// Package github provides GitHub API client functionality for repolint.
package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"gopkg.in/yaml.v3"
)

const (
	maxBackoffDuration = 1 * time.Minute
	initialBackoff     = 1 * time.Second
)

// Client provides cached GitHub API access with rate limiting
type Client struct {
	rest    *api.RESTClient
	owner   string
	repo    string
	verbose bool

	cacheMu sync.RWMutex
	cache   map[string]any
}

// NewClient creates a new GitHub client
func NewClient(owner, repo string, verbose bool) (*Client, error) {
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &Client{
		rest:    restClient,
		owner:   owner,
		repo:    repo,
		verbose: verbose,
		cache:   make(map[string]any),
	}, nil
}

// RESTClient returns the underlying REST client
func (c *Client) RESTClient() *api.RESTClient {
	return c.rest
}

// Owner returns the repository owner
func (c *Client) Owner() string {
	return c.owner
}

// Repo returns the repository name
func (c *Client) Repo() string {
	return c.repo
}

// GetRepository fetches repository information
func (c *Client) GetRepository() (*Repository, error) {
	cacheKey := fmt.Sprintf("repo:%s/%s", c.owner, c.repo)

	if cached := c.getFromCache(cacheKey); cached != nil {
		if repo, ok := cached.(*Repository); ok {
			return repo, nil
		}
	}

	var repo Repository
	path := fmt.Sprintf("repos/%s/%s", c.owner, c.repo)

	if err := c.doWithRetry("GET", path, nil, &repo); err != nil {
		return nil, err
	}

	if repo.Archived {
		return nil, fmt.Errorf("repository %s/%s is archived", c.owner, c.repo)
	}

	c.setCache(cacheKey, &repo)
	return &repo, nil
}

// GetWorkflowPermissions fetches workflow permissions for the repository
func (c *Client) GetWorkflowPermissions() (*WorkflowPermissions, error) {
	var perms WorkflowPermissions
	path := fmt.Sprintf("repos/%s/%s/actions/permissions/workflow", c.owner, c.repo)

	if err := c.doWithRetry("GET", path, nil, &perms); err != nil {
		return nil, err
	}

	return &perms, nil
}

// GetRulesets fetches repository rulesets
func (c *Client) GetRulesets() ([]Ruleset, error) {
	cacheKey := fmt.Sprintf("rulesets:%s/%s", c.owner, c.repo)

	if cached := c.getFromCache(cacheKey); cached != nil {
		if rulesets, ok := cached.([]Ruleset); ok {
			return rulesets, nil
		}
	}

	var rulesets []Ruleset
	path := fmt.Sprintf("repos/%s/%s/rulesets", c.owner, c.repo)

	if err := c.doWithRetry("GET", path, nil, &rulesets); err != nil {
		return nil, err
	}

	c.setCache(cacheKey, rulesets)
	return rulesets, nil
}

// GetRuleset fetches a specific ruleset by ID
func (c *Client) GetRuleset(id int) (*Ruleset, error) {
	cacheKey := fmt.Sprintf("ruleset:%s/%s/%d", c.owner, c.repo, id)

	if cached := c.getFromCache(cacheKey); cached != nil {
		if ruleset, ok := cached.(*Ruleset); ok {
			return ruleset, nil
		}
	}

	var ruleset Ruleset
	path := fmt.Sprintf("repos/%s/%s/rulesets/%d", c.owner, c.repo, id)

	if err := c.doWithRetry("GET", path, nil, &ruleset); err != nil {
		return nil, err
	}

	c.setCache(cacheKey, &ruleset)
	return &ruleset, nil
}

// GetFileContent fetches a file's content from the repository
func (c *Client) GetFileContent(filePath string) ([]byte, error) {
	cacheKey := fmt.Sprintf("file:%s/%s/%s", c.owner, c.repo, filePath)

	if cached := c.getFromCache(cacheKey); cached != nil {
		if content, ok := cached.([]byte); ok {
			return content, nil
		}
	}

	var content FileContent
	path := fmt.Sprintf("repos/%s/%s/contents/%s", c.owner, c.repo, filePath)

	if err := c.doWithRetry("GET", path, nil, &content); err != nil {
		return nil, err
	}

	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	c.setCache(cacheKey, decoded)
	return decoded, nil
}

// GetRemoteFileContent fetches a file from another repository
func (c *Client) GetRemoteFileContent(owner, repo, filePath string) ([]byte, error) {
	cacheKey := fmt.Sprintf("remote-file:%s/%s/%s", owner, repo, filePath)

	if cached := c.getFromCache(cacheKey); cached != nil {
		if content, ok := cached.([]byte); ok {
			return content, nil
		}
	}

	var content FileContent
	path := fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, filePath)

	if err := c.doWithRetry("GET", path, nil, &content); err != nil {
		return nil, err
	}

	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	c.setCache(cacheKey, decoded)
	return decoded, nil
}

// GetVulnerabilityAlertsEnabled checks if Dependabot alerts (vulnerability alerts) are enabled
// Returns true if enabled, false if disabled
func (c *Client) GetVulnerabilityAlertsEnabled() (bool, error) {
	path := fmt.Sprintf("repos/%s/%s/vulnerability-alerts", c.owner, c.repo)

	// This endpoint returns 204 if enabled, 404 if disabled
	err := c.doWithRetry("GET", path, nil, nil)
	if err != nil {
		// 404 means vulnerability alerts are disabled
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// EnableVulnerabilityAlerts enables Dependabot alerts (vulnerability alerts)
func (c *Client) EnableVulnerabilityAlerts() error {
	path := fmt.Sprintf("repos/%s/%s/vulnerability-alerts", c.owner, c.repo)
	return c.doWithRetry("PUT", path, nil, nil)
}

// DisableVulnerabilityAlerts disables Dependabot alerts (vulnerability alerts)
func (c *Client) DisableVulnerabilityAlerts() error {
	path := fmt.Sprintf("repos/%s/%s/vulnerability-alerts", c.owner, c.repo)
	return c.doWithRetry("DELETE", path, nil, nil)
}

// GetAutomatedSecurityFixes checks if Dependabot security updates are enabled
func (c *Client) GetAutomatedSecurityFixes() (*AutomatedSecurityFixes, error) {
	var result AutomatedSecurityFixes
	path := fmt.Sprintf("repos/%s/%s/automated-security-fixes", c.owner, c.repo)

	if err := c.doWithRetry("GET", path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// EnableAutomatedSecurityFixes enables Dependabot security updates
func (c *Client) EnableAutomatedSecurityFixes() error {
	path := fmt.Sprintf("repos/%s/%s/automated-security-fixes", c.owner, c.repo)
	return c.doWithRetry("PUT", path, nil, nil)
}

// DisableAutomatedSecurityFixes disables Dependabot security updates
func (c *Client) DisableAutomatedSecurityFixes() error {
	path := fmt.Sprintf("repos/%s/%s/automated-security-fixes", c.owner, c.repo)
	return c.doWithRetry("DELETE", path, nil, nil)
}

// GetWorkflow fetches and parses a workflow file
func (c *Client) GetWorkflow(path string) (*Workflow, error) {
	content, err := c.GetLocalFileContent(path)
	if err != nil {
		return nil, err
	}

	var wf Workflow
	if err := yaml.Unmarshal(content, &wf); err != nil {
		return nil, fmt.Errorf("invalid workflow file %s: %w", path, err)
	}

	return &wf, nil
}

// UpdateRepository updates repository settings
func (c *Client) UpdateRepository(req *RepoUpdateRequest) error {
	path := fmt.Sprintf("repos/%s/%s", c.owner, c.repo)
	return c.doWithRetry("PATCH", path, req, nil)
}

// UpdateWorkflowPermissions updates workflow permissions
func (c *Client) UpdateWorkflowPermissions(canApprove bool) error {
	path := fmt.Sprintf("repos/%s/%s/actions/permissions/workflow", c.owner, c.repo)
	req := map[string]any{
		"can_approve_pull_request_reviews": canApprove,
	}
	return c.doWithRetry("PUT", path, req, nil)
}

// CreateRuleset creates a new ruleset
func (c *Client) CreateRuleset(req *RulesetCreateRequest) (*Ruleset, error) {
	path := fmt.Sprintf("repos/%s/%s/rulesets", c.owner, c.repo)
	var ruleset Ruleset
	if err := c.doWithRetry("POST", path, req, &ruleset); err != nil {
		return nil, err
	}
	return &ruleset, nil
}

// UpdateRuleset updates an existing ruleset
func (c *Client) UpdateRuleset(id int, req *RulesetCreateRequest) error {
	path := fmt.Sprintf("repos/%s/%s/rulesets/%d", c.owner, c.repo, id)
	return c.doWithRetry("PUT", path, req, nil)
}

// doWithRetry performs an API request with exponential backoff for rate limiting
func (c *Client) doWithRetry(method, path string, body, result any) error {
	backoff := initialBackoff
	totalWait := time.Duration(0)

	for {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[API] %s %s\n", method, path)
		}

		var err error
		switch method {
		case "GET":
			err = c.rest.Get(path, result)
		case "POST":
			bodyReader, encErr := encodeBody(body)
			if encErr != nil {
				return encErr
			}
			err = c.rest.Post(path, bodyReader, result)
		case "PATCH":
			bodyReader, encErr := encodeBody(body)
			if encErr != nil {
				return encErr
			}
			err = c.rest.Patch(path, bodyReader, result)
		case "PUT":
			bodyReader, encErr := encodeBody(body)
			if encErr != nil {
				return encErr
			}
			err = c.rest.Put(path, bodyReader, result)
		case "DELETE":
			err = c.rest.Delete(path, result)
		default:
			return fmt.Errorf("unsupported method: %s", method)
		}

		if err == nil {
			return nil
		}

		// Check if this is a rate limit error
		if !isRateLimitError(err) {
			return err
		}

		if totalWait >= maxBackoffDuration {
			return fmt.Errorf("rate limit exceeded, waited %v: %w", totalWait, err)
		}

		fmt.Fprintf(os.Stderr, "Rate limited, waiting %v before retry...\n", backoff)
		time.Sleep(backoff)
		totalWait += backoff
		backoff *= 2
		if backoff > maxBackoffDuration-totalWait {
			backoff = maxBackoffDuration - totalWait
		}
	}
}

// encodeBody encodes the body as JSON
func encodeBody(body any) (*bytes.Buffer, error) {
	if body == nil {
		// Return empty buffer instead of nil - go-gh REST client requires non-nil body
		return new(bytes.Buffer), nil
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}
	return buf, nil
}

// isRateLimitError checks if the error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "secondary rate limit")
}

// Cache methods

func (c *Client) getFromCache(key string) any {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return c.cache[key]
}

func (c *Client) setCache(key string, value any) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[key] = value
}

// CheckPermissions verifies the client has necessary permissions
func (c *Client) CheckPermissions() error {
	// Try to fetch repository to check basic access
	_, err := c.GetRepository()
	if err != nil {
		return fmt.Errorf("insufficient permissions to access repository: %w", err)
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsNotFound checks if an error is a 404 not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	httpErr := &HTTPError{}
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusNotFound
	}
	return strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found")
}

// IsForbidden checks if an error is a 403 forbidden error
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	httpErr := &HTTPError{}
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == http.StatusForbidden
	}
	return strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden")
}

// ReadLocalWorkflowFile reads a workflow file from the local filesystem
func ReadLocalWorkflowFile(path string) (*Workflow, []byte, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Reading user-specified workflow files is intentional
	if err != nil {
		return nil, nil, err
	}

	var wf Workflow
	if err := yaml.Unmarshal(content, &wf); err != nil {
		return nil, content, fmt.Errorf("invalid workflow file %s: %w", path, err)
	}

	return &wf, content, nil
}

// ResolveReferenceFile resolves a reference file from local filesystem or remote repository
func ResolveReferenceFile(reference string, client *Client) ([]byte, error) {
	var content []byte
	var err error

	// First, try to read from local filesystem
	content, err = os.ReadFile(reference) //nolint:gosec // Reading user-specified reference files is intentional
	if err == nil {
		// Successfully read from local file
		return content, nil
	}

	// If local file doesn't exist, try remote repository lookup
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read local reference file: %w", err)
	}

	// Parse reference as remote: owner/repo/path
	parts := strings.SplitN(reference, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("reference file '%s' not found locally and invalid remote format (expected owner/repo/path)", reference)
	}

	refOwner, refRepo, refPath := parts[0], parts[1], parts[2]

	// Fetch reference content from remote
	content, err = client.GetRemoteFileContent(refOwner, refRepo, refPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote reference file: %w", err)
	}
	return content, nil
}

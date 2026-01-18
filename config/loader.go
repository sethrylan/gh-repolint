package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/sethrylan/gh-repolint/github"
	"gopkg.in/yaml.v3"
)

// ConfigFileNames contains the candidate config file names in priority order
var ConfigFileNames = []string{".repolint.yaml", ".repolint.yml"}

// Source indicates where a config was loaded from
type Source int

// Source values for configuration origin tracking
const (
	SourceNone Source = iota
	SourceRepo
	SourceOwner
)

func (s Source) String() string {
	switch s {
	case SourceRepo:
		return "repo"
	case SourceOwner:
		return "owner"
	default:
		return "none"
	}
}

// LoadedConfig contains the config and its source information
type LoadedConfig struct {
	Config      *Config
	RepoConfig  *Config
	OwnerConfig *Config
	RepoSource  string
	OwnerSource string
}

// Loader handles configuration discovery and loading
type Loader struct {
	client *api.RESTClient
	owner  string
	repo   string
}

// NewLoader creates a new config loader
func NewLoader(client *github.Client) *Loader {
	return &Loader{
		client: client.RESTClient(),
		owner:  client.Owner(),
		repo:   client.Repo(),
	}
}

// Load discovers and loads configuration files
// Returns the merged config, or an error if no config is found
func (l *Loader) Load() (*LoadedConfig, error) {
	result := &LoadedConfig{}

	// Try to load repo-level config from local filesystem first
	repoConfig, repoFileName, err := l.loadLocalConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading repo config: %w", err)
	}
	if repoConfig != nil {
		result.RepoConfig = repoConfig
		result.RepoSource = fmt.Sprintf("%s/%s/%s", l.owner, l.repo, repoFileName)
	}

	// Try to load owner-level config from <owner>/<owner> repo
	ownerConfig, ownerFileName, err := l.loadOwnerConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading owner config: %w", err)
	}
	if ownerConfig != nil {
		result.OwnerConfig = ownerConfig
		result.OwnerSource = fmt.Sprintf("%s/%s/%s", l.owner, l.owner, ownerFileName)
	}

	// If neither exists, return error
	if result.RepoConfig == nil && result.OwnerConfig == nil {
		return nil, fmt.Errorf("no configuration found: checked %s/%s/{%s} and %s/%s/{%s}. To get started, run 'gh repolint init'",
			l.owner, l.repo, strings.Join(ConfigFileNames, ","), l.owner, l.owner, strings.Join(ConfigFileNames, ","))
	}

	// Merge configs (repo takes precedence over owner)
	result.Config = MergeConfigs(result.OwnerConfig, result.RepoConfig)

	return result, nil
}

// LoadFromFile loads configuration from a specific file path
// This bypasses normal config discovery and uses only the specified file
func (l *Loader) LoadFromFile(path string) (*LoadedConfig, error) {
	file, err := os.Open(path) //nolint:gosec // Reading config from user-specified path is intentional
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	cfg, err := parseConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &LoadedConfig{
		Config:     cfg,
		RepoConfig: cfg,
		RepoSource: path,
		// OwnerConfig and OwnerSource are intentionally left nil/empty
	}, nil
}

// findConfigFile returns the path of the first existing config file from candidates,
// or empty string if none exist
func findConfigFile(dir string) string {
	for _, name := range ConfigFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// loadLocalConfig loads config from the local repository root
func (l *Loader) loadLocalConfig() (*Config, string, error) {
	// First try current directory
	configPath := findConfigFile(".")
	if configPath == "" {
		// Try to find git root
		gitRoot, err := findGitRoot()
		if err != nil {
			return nil, "", nil
		}
		configPath = findConfigFile(gitRoot)
		if configPath == "" {
			return nil, "", nil
		}
	}

	file, err := os.Open(configPath) //nolint:gosec // Reading config from known paths is intentional
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", err
	}
	defer func() { _ = file.Close() }()

	cfg, err := parseConfig(file)
	if err != nil {
		return nil, "", err
	}
	return cfg, filepath.Base(configPath), nil
}

// loadOwnerConfig loads config from the owner's org-level repo
func (l *Loader) loadOwnerConfig() (*Config, string, error) {
	// If repo is the same as owner (org-level repo), skip owner config
	if l.repo == l.owner {
		return nil, "", nil
	}

	var content struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	// Try each config file name in priority order
	for _, name := range ConfigFileNames {
		// Fetch from GitHub API: GET /repos/{owner}/{owner}/contents/{path}
		path := fmt.Sprintf("repos/%s/%s/contents/%s", l.owner, l.owner, name)

		err := l.client.Get(path, &content)
		if err != nil {
			// If 404, try next filename
			continue
		}

		if content.Encoding != "base64" {
			return nil, "", fmt.Errorf("unexpected encoding: %s", content.Encoding)
		}

		// Decode base64 content
		decoded, err := decodeBase64(content.Content)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode content: %w", err)
		}

		cfg, err := parseConfigBytes(decoded)
		if err != nil {
			return nil, "", err
		}
		return cfg, name, nil
	}

	// No config file found - not an error
	return nil, "", nil
}

// parseConfig parses a config from a reader
func parseConfig(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parseConfigBytes(data)
}

// parseConfigBytes parses a config from bytes
func parseConfigBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	return &cfg, nil
}

// findGitRoot finds the root of the git repository
func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not in a git repository")
		}
		dir = parent
	}
}

// decodeBase64 decodes a base64 string (handling newlines in GitHub's response)
func decodeBase64(s string) ([]byte, error) {
	// Remove any whitespace/newlines from base64 string
	var clean strings.Builder
	clean.Grow(len(s))
	for _, c := range s {
		if c != '\n' && c != '\r' && c != ' ' {
			clean.WriteRune(c)
		}
	}
	return base64.StdEncoding.DecodeString(clean.String())
}

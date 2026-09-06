// Package agentcatalog loads and validates the catalog of supported AI coding agents.
package agentcatalog

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.yaml
var embeddedFS embed.FS

// AgentDefinition represents one supported AI coding agent listed in the catalog.
type AgentDefinition struct {
	ID          string   `yaml:"id"`
	DisplayName string   `yaml:"display_name"`
	Files       []string `yaml:"files"`
	Folders     []string `yaml:"folders"`
}

// GitConfig is the single, tool-wide git repository and commit retrieved after
// agent selection, independent of any specific AgentDefinition.
type GitConfig struct {
	Repository string        `yaml:"repository"`
	Ref        string        `yaml:"ref"`
	Managed    ManagedConfig `yaml:"managed"`
	Seeded     SeededConfig  `yaml:"seeded"`
}

// ManagedConfig is the set of files and folders that apply on every init run,
// independent of which agent is selected (feature 005). Managed files behave like an
// agent's declared files (create-or-overwrite, never deleted); managed folders are
// additionally mirrored exactly, including removal of destination files the repository
// no longer provides. A zero-value ManagedConfig (both fields nil) is valid and means
// no managed files/folders are declared.
type ManagedConfig struct {
	Files   []string `yaml:"files"`
	Folders []string `yaml:"folders"`
}

// SeededConfig is the set of files and folders that are created at the destination on
// every init run, independent of which agent is selected (feature 006), but only the
// first time nothing already exists at their exact destination path. Unlike
// ManagedConfig, seeded content is never overwritten or removed once it exists. A
// zero-value SeededConfig (both fields nil) is valid and means no seeded files/folders
// are declared.
type SeededConfig struct {
	Files   []string `yaml:"files"`
	Folders []string `yaml:"folders"`
}

// AgentCatalog is the full set of agent definitions and git configuration loaded from the catalog file.
type AgentCatalog struct {
	Agents []AgentDefinition `yaml:"agents"`
	Git    GitConfig         `yaml:"git"`
}

// ErrCatalogNotFound indicates the catalog file could not be found (FR-006).
var ErrCatalogNotFound = errors.New("agent catalog configuration not found")

// ErrCatalogEmpty indicates the catalog has no valid supported agents (FR-008).
var ErrCatalogEmpty = errors.New("agent catalog configuration has no supported agents")

// ErrGitConfigMissing indicates no git repository/commit is configured (FR-006).
var ErrGitConfigMissing = errors.New("agent catalog configuration has no git repository configured")

// ErrInvalidDeclaredPath indicates an agent's declared files/folders list contains an
// empty, absolute, or path-traversing (`..`) entry (FR-006 of feature 003).
var ErrInvalidDeclaredPath = errors.New("agent catalog configuration has an invalid declared file or folder path")

// Load reads and validates the agent catalog embedded in the binary.
func Load() (*AgentCatalog, error) {
	return LoadFS(embeddedFS, "config.yaml")
}

// LoadFS reads and validates an agent catalog at name within fsys, allowing
// tests to inject fixture catalogs independent of the embedded production file.
func LoadFS(fsys fs.FS, name string) (*AgentCatalog, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCatalogNotFound, name)
	}

	var catalog AgentCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse agent catalog configuration: %w", err)
	}

	catalog.Agents = dedupeAndValidate(catalog.Agents)
	if len(catalog.Agents) == 0 {
		return nil, ErrCatalogEmpty
	}

	if err := validateDeclaredPaths(catalog.Agents); err != nil {
		return nil, err
	}

	if err := validateManagedPaths(catalog.Git.Managed); err != nil {
		return nil, err
	}

	if err := validateSeededPaths(catalog.Git.Seeded); err != nil {
		return nil, err
	}

	if catalog.Git.Repository == "" || catalog.Git.Ref == "" {
		return nil, ErrGitConfigMissing
	}

	return &catalog, nil
}

// dedupeAndValidate drops entries with an empty id or display_name and removes
// duplicate ids, keeping the first occurrence (FR-009).
func dedupeAndValidate(agents []AgentDefinition) []AgentDefinition {
	seen := make(map[string]bool, len(agents))
	result := make([]AgentDefinition, 0, len(agents))
	for _, a := range agents {
		if a.ID == "" || a.DisplayName == "" {
			continue
		}
		if seen[a.ID] {
			continue
		}
		seen[a.ID] = true
		result = append(result, a)
	}
	return result
}

// validateDeclaredPaths rejects an empty, absolute, or path-traversing (`..`) entry in
// any agent's declared Files/Folders lists (FR-006 of feature 003).
func validateDeclaredPaths(agents []AgentDefinition) error {
	for _, a := range agents {
		for _, p := range a.Files {
			if !isValidDeclaredPath(p) {
				return fmt.Errorf("%w: agent %q declares file %q", ErrInvalidDeclaredPath, a.ID, p)
			}
		}
		for _, p := range a.Folders {
			if !isValidDeclaredPath(p) {
				return fmt.Errorf("%w: agent %q declares folder %q", ErrInvalidDeclaredPath, a.ID, p)
			}
		}
	}
	return nil
}

// validateManagedPaths rejects an empty, absolute, or path-traversing (`..`) entry in
// the git config's managed Files/Folders lists (FR-001 of feature 005).
func validateManagedPaths(managed ManagedConfig) error {
	for _, p := range managed.Files {
		if !isValidDeclaredPath(p) {
			return fmt.Errorf("%w: managed file %q", ErrInvalidDeclaredPath, p)
		}
	}
	for _, p := range managed.Folders {
		if !isValidDeclaredPath(p) {
			return fmt.Errorf("%w: managed folder %q", ErrInvalidDeclaredPath, p)
		}
	}
	return nil
}

// validateSeededPaths rejects an empty, absolute, or path-traversing (`..`) entry in
// the git config's seeded Files/Folders lists (FR-001 of feature 006).
func validateSeededPaths(seeded SeededConfig) error {
	for _, p := range seeded.Files {
		if !isValidDeclaredPath(p) {
			return fmt.Errorf("%w: seeded file %q", ErrInvalidDeclaredPath, p)
		}
	}
	for _, p := range seeded.Folders {
		if !isValidDeclaredPath(p) {
			return fmt.Errorf("%w: seeded folder %q", ErrInvalidDeclaredPath, p)
		}
	}
	return nil
}

// isValidDeclaredPath reports whether p is a non-empty, relative, repository-root-scoped
// path with no `..` segment, using the repository's forward-slash convention.
func isValidDeclaredPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") {
		return false
	}
	for _, segment := range strings.Split(p, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}

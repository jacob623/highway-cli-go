// Package agentcatalog loads and validates the catalog of supported AI coding agents.
package agentcatalog

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

//go:embed agents.yaml
var embeddedFS embed.FS

// AgentDefinition represents one supported AI coding agent listed in the catalog.
type AgentDefinition struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
}

// AgentCatalog is the full set of agent definitions loaded from the catalog file.
type AgentCatalog struct {
	Agents []AgentDefinition `yaml:"agents"`
}

// ErrCatalogNotFound indicates the catalog file could not be found (FR-006).
var ErrCatalogNotFound = errors.New("agent catalog configuration not found")

// ErrCatalogEmpty indicates the catalog has no valid supported agents (FR-008).
var ErrCatalogEmpty = errors.New("agent catalog configuration has no supported agents")

// Load reads and validates the agent catalog embedded in the binary.
func Load() (*AgentCatalog, error) {
	return LoadFS(embeddedFS, "agents.yaml")
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

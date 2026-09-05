package agentcatalog

import (
	"errors"
	"testing"
	"testing/fstest"
)

func TestLoadFS_Success(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []AgentDefinition
	}{
		{
			name: "multiple distinct agents",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
  - id: claude-code
    display_name: Claude Code
`,
			want: []AgentDefinition{
				{ID: "vscode", DisplayName: "GitHub Copilot"},
				{ID: "claude-code", DisplayName: "Claude Code"},
			},
		},
		{
			name: "duplicate id keeps first occurrence",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
  - id: vscode
    display_name: Duplicate Copilot
  - id: cursor
    display_name: Cursor
`,
			want: []AgentDefinition{
				{ID: "vscode", DisplayName: "GitHub Copilot"},
				{ID: "cursor", DisplayName: "Cursor"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte(tt.content)},
			}

			got, err := LoadFS(fsys, "config.yaml")
			if err != nil {
				t.Fatalf("LoadFS() unexpected error: %v", err)
			}
			if len(got.Agents) != len(tt.want) {
				t.Fatalf("LoadFS() got %d agents, want %d: %+v", len(got.Agents), len(tt.want), got.Agents)
			}
			for i, agent := range got.Agents {
				if agent != tt.want[i] {
					t.Errorf("LoadFS() agent[%d] = %+v, want %+v", i, agent, tt.want[i])
				}
			}
		})
	}
}

func TestLoadFS_GitConfig(t *testing.T) {
	fsys := fstest.MapFS{
		"config.yaml": &fstest.MapFile{Data: []byte(`git:
  repository: https://example.com/example/repo.git
  ref: abc123def456
agents:
  - id: vscode
    display_name: GitHub Copilot
`)},
	}

	got, err := LoadFS(fsys, "config.yaml")
	if err != nil {
		t.Fatalf("LoadFS() unexpected error: %v", err)
	}
	want := GitConfig{Repository: "https://example.com/example/repo.git", Ref: "abc123def456"}
	if got.Git != want {
		t.Errorf("LoadFS() Git = %+v, want %+v", got.Git, want)
	}
}

func TestLoadFS_Errors(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fstest.MapFS
		wantErr error
	}{
		{
			name:    "missing file",
			fsys:    fstest.MapFS{},
			wantErr: ErrCatalogNotFound,
		},
		{
			name: "empty agents list",
			fsys: fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte("agents: []\n")},
			},
			wantErr: ErrCatalogEmpty,
		},
		{
			name: "absent agents key",
			fsys: fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte("other: value\n")},
			},
			wantErr: ErrCatalogEmpty,
		},
		{
			name: "malformed yaml",
			fsys: fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte("agents: [\n  - id: vscode\n")},
			},
			wantErr: nil, // malformed YAML returns a wrapped parse error, not a sentinel
		},
		{
			name: "missing git config",
			fsys: fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte(`agents:
  - id: vscode
    display_name: GitHub Copilot
`)},
			},
			wantErr: ErrGitConfigMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFS(tt.fsys, "config.yaml")
			if err == nil {
				t.Fatal("LoadFS() expected an error, got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("LoadFS() error = %v, want it to wrap %v", err, tt.wantErr)
			}
		})
	}
}

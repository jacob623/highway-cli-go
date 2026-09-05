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
				if agent.ID != tt.want[i].ID || agent.DisplayName != tt.want[i].DisplayName {
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
	if got.Git.Repository != want.Repository || got.Git.Ref != want.Ref {
		t.Errorf("LoadFS() Git = %+v, want %+v", got.Git, want)
	}
	if len(got.Git.Managed.Files) != 0 || len(got.Git.Managed.Folders) != 0 {
		t.Errorf("LoadFS() Git.Managed = %+v, want empty", got.Git.Managed)
	}
}

func TestLoadFS_DeclaredFilesAndFolders(t *testing.T) {
	fsys := fstest.MapFS{
		"config.yaml": &fstest.MapFile{Data: []byte(`git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
    files:
      - file1.md
      - file2.md
    folders:
      - .github/skills
      - .idea/library
  - id: cursor
    display_name: Cursor
`)},
	}

	got, err := LoadFS(fsys, "config.yaml")
	if err != nil {
		t.Fatalf("LoadFS() unexpected error: %v", err)
	}

	wantVscode := AgentDefinition{
		ID:          "vscode",
		DisplayName: "GitHub Copilot",
		Files:       []string{"file1.md", "file2.md"},
		Folders:     []string{".github/skills", ".idea/library"},
	}
	if got.Agents[0].ID != wantVscode.ID ||
		got.Agents[0].DisplayName != wantVscode.DisplayName ||
		len(got.Agents[0].Files) != len(wantVscode.Files) ||
		len(got.Agents[0].Folders) != len(wantVscode.Folders) {
		t.Errorf("LoadFS() agents[0] = %+v, want %+v", got.Agents[0], wantVscode)
	}
	for i, f := range wantVscode.Files {
		if got.Agents[0].Files[i] != f {
			t.Errorf("LoadFS() agents[0].Files[%d] = %q, want %q", i, got.Agents[0].Files[i], f)
		}
	}
	for i, f := range wantVscode.Folders {
		if got.Agents[0].Folders[i] != f {
			t.Errorf("LoadFS() agents[0].Folders[%d] = %q, want %q", i, got.Agents[0].Folders[i], f)
		}
	}

	if len(got.Agents[1].Files) != 0 || len(got.Agents[1].Folders) != 0 {
		t.Errorf("LoadFS() agents[1] (cursor) = %+v, want no declared files/folders", got.Agents[1])
	}
}

func TestLoadFS_ManagedConfigParsesFromYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"config.yaml": &fstest.MapFile{Data: []byte(`git:
  repository: https://example.com/example/repo.git
  ref: abc123
  managed:
    files:
      - file1.md
    folders:
      - .highway
agents:
  - id: vscode
    display_name: GitHub Copilot
`)},
	}

	got, err := LoadFS(fsys, "config.yaml")
	if err != nil {
		t.Fatalf("LoadFS() unexpected error: %v", err)
	}

	wantFiles := []string{"file1.md"}
	wantFolders := []string{".highway"}
	if len(got.Git.Managed.Files) != len(wantFiles) || got.Git.Managed.Files[0] != wantFiles[0] {
		t.Errorf("LoadFS() Git.Managed.Files = %v, want %v", got.Git.Managed.Files, wantFiles)
	}
	if len(got.Git.Managed.Folders) != len(wantFolders) || got.Git.Managed.Folders[0] != wantFolders[0] {
		t.Errorf("LoadFS() Git.Managed.Folders = %v, want %v", got.Git.Managed.Folders, wantFolders)
	}
}

func TestLoadFS_AbsentManagedConfigIsBackwardCompatible(t *testing.T) {
	fsys := fstest.MapFS{
		"config.yaml": &fstest.MapFile{Data: []byte(`git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
`)},
	}

	got, err := LoadFS(fsys, "config.yaml")
	if err != nil {
		t.Fatalf("LoadFS() unexpected error: %v", err)
	}
	if len(got.Git.Managed.Files) != 0 || len(got.Git.Managed.Folders) != 0 {
		t.Errorf("LoadFS() Git.Managed = %+v, want empty", got.Git.Managed)
	}
}

func TestLoadFS_InvalidManagedPathFails(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "empty managed file entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
  managed:
    files:
      - ""
agents:
  - id: vscode
    display_name: GitHub Copilot
`,
		},
		{
			name: "absolute managed folder entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
  managed:
    folders:
      - /etc/passwd
agents:
  - id: vscode
    display_name: GitHub Copilot
`,
		},
		{
			name: "path-traversing managed file entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
  managed:
    files:
      - ../outside-repo.md
agents:
  - id: vscode
    display_name: GitHub Copilot
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte(tt.content)},
			}

			_, err := LoadFS(fsys, "config.yaml")
			if !errors.Is(err, ErrInvalidDeclaredPath) {
				t.Errorf("LoadFS() error = %v, want it to wrap ErrInvalidDeclaredPath", err)
			}
		})
	}
}

func TestLoadFS_InvalidDeclaredPath(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "empty file entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
    files:
      - ""
`,
		},
		{
			name: "absolute folder entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
    folders:
      - /etc/passwd
`,
		},
		{
			name: "path-traversing file entry",
			content: `git:
  repository: https://example.com/example/repo.git
  ref: abc123
agents:
  - id: vscode
    display_name: GitHub Copilot
    files:
      - ../outside-repo.md
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"config.yaml": &fstest.MapFile{Data: []byte(tt.content)},
			}

			_, err := LoadFS(fsys, "config.yaml")
			if !errors.Is(err, ErrInvalidDeclaredPath) {
				t.Errorf("LoadFS() error = %v, want it to wrap ErrInvalidDeclaredPath", err)
			}
		})
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

# Research: CLI Init Agent Selection

All items below were resolved from the user-provided technical context; no
`NEEDS CLARIFICATION` markers remain in the plan.

## Language & toolchain

- **Decision**: Go 1.23.
- **Rationale**: Current stable Go release; `go:embed` (used to bundle the agent
  catalog into the binary) has been stable since Go 1.16, so any recent Go 1.x
  version works — 1.23 simply tracks current tooling.
- **Alternatives considered**: Pinning to an older Go version — rejected, no
  requirement drives an older minimum.

## CLI command framework

- **Decision**: `github.com/spf13/cobra`.
- **Rationale**: De facto standard for multi-command Go CLIs; the user noted
  the configuration (and, by extension, the command surface) will grow over
  future iterations, and cobra scales cleanly to additional subcommands and
  flags without restructuring.
- **Alternatives considered**: Standard library `flag` package (rejected —
  harder to grow into subcommands cleanly); `urfave/cli` (comparable, but
  cobra is more widely used in the Go ecosystem and pairs well with Viper if
  configuration needs grow later).

## YAML parsing

- **Decision**: `gopkg.in/yaml.v3`.
- **Rationale**: De facto standard YAML library for Go; supports the struct
  tags needed to map the `agents:` list (with `id` / `display_name`).
- **Alternatives considered**: `sigs.k8s.io/yaml` (round-trips through JSON,
  unnecessary indirection for this use case).

## Embedding the agent catalog

- **Decision**: Store the catalog as `internal/agentcatalog/agents.yaml` and
  embed it into the binary at compile time with `go:embed`.
- **Rationale**: The user specified the configuration file is "contained
  within the CLI" and should be kept as an editable YAML config that grows
  over time — `go:embed` is the standard Go mechanism for bundling a real,
  human-editable file into the binary so there is no runtime dependency on an
  external file path.
- **Alternatives considered**: Hardcoding agents as Go struct literals
  (rejected — user explicitly wants a YAML file that maintainers edit as the
  catalog grows); reading the file from disk relative to the binary at
  runtime (rejected — fragile if the binary is moved or invoked from another
  directory, and reintroduces the "missing config file" failure mode for a
  file that should always ship with the tool).

## Interactive selection UX

- **Decision**: `github.com/charmbracelet/huh`'s `Select` field for the
  single-select list.
- **Rationale**: Actively maintained (commits within the last month, regular
  tagged releases) and satisfies the constitution's Technology & Dependency
  Constraints requirement that new dependencies be actively maintained.
  Provides an arrow-key list selection UX with built-in Ctrl+C cancellation
  (no confirmation shown, exit non-zero) and re-prompting on invalid input,
  directly satisfying FR-003 and FR-004 without hand-rolled input-parsing
  edge cases.
- **Alternatives considered**: `manifoldco/promptui` (rejected — no commits
  or releases in 5 years, effectively unmaintained); `AlecAivazis/survey`
  (rejected — officially archived by its maintainer, README explicitly says
  it is no longer maintained); hand-rolled numbered stdin prompt (rejected —
  requires bespoke validation/re-prompt/cancel handling).

## Non-interactive / no-TTY handling

- **Decision**: Detect whether stdin is a terminal using
  `golang.org/x/term.IsTerminal` before invoking the prompt; if not a TTY,
  print a clear error and exit non-zero instead of prompting.
- **Rationale**: Directly satisfies FR-010 ("must exit with a clear error,
  rather than hang, when an interactive prompt cannot be presented").
- **Alternatives considered**: Attempting the prompt unconditionally and
  relying on a read timeout — rejected, causes CI/scripted invocations to
  hang rather than fail fast.

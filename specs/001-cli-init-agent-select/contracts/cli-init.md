# CLI Contract: `init` command

## Invocation

```text
highway init
```

No required arguments or flags for the scenarios covered by this feature.

## Preconditions

- The binary has a valid embedded agent catalog (built into the binary; not
  user-supplied at runtime).

## Behavior

1. Load and validate the embedded agent catalog.
   - Missing/unparsable/empty catalog → print an actionable error to stderr,
     exit non-zero. **No prompt is shown.**
2. Verify stdin is an interactive terminal.
   - If not a TTY → print an actionable error to stderr, exit non-zero.
     **No prompt is shown, process does not hang.**
3. Present the de-duplicated list of agents (`display_name`, ordered as in
   the catalog) as an interactive single-select prompt.
4. On developer confirmation of a choice:
   - Print a confirmation message naming the chosen agent's `display_name`.
   - Exit 0.
5. On cancellation (e.g., Ctrl+C):
   - Print no confirmation message.
   - Exit non-zero.
6. On invalid selection input that is not a cancellation (e.g., out-of-range
   entry in a non-arrow-key input mode):
   - Re-prompt; do not exit or crash.

## Outputs

| Channel | Content |
|---------|---------|
| stdout  | Interactive prompt UI; final confirmation message naming the selected agent |
| stderr  | Actionable error messages for missing/empty/malformed catalog, no-TTY environment |
| exit code | `0` on a completed selection; non-zero on any error or cancellation |

The selection is never written to disk (FR-011); the outputs above are the
only effects of running `init`.

## Out of scope for this contract

- Flags for non-interactive/unattended selection (explicitly out of scope
  per spec Assumptions).
- Any subcommand other than `init`.

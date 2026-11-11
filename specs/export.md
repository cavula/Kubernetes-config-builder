# Export

Approved November 11, 2026.

## Inputs

The active format, its current generated text, and a copy or download action.

## Outputs

Clipboard text or one YAML file containing the displayed output, including explanatory comments.

## Acceptance criteria

- [ ] One click copies the active text and reports success or clipboard failure.
- [ ] One click downloads the active text as `<tool>-manifests.yaml` in YAML mode or `<tool>-values.yaml` in Helm mode.
- [ ] Both actions use only the current active format, including while the comparison is open.
- [ ] Validation warnings and errors leave both actions available whenever generated content exists.
- [ ] Both actions work without a live cluster or Git credentials.

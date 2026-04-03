# Platform Contract Verification

## Pattern

When dot-agents renders a vendor-native config format, assumptions about field names or file shapes can drift from the current platform contract and still pass local tests if the tests only mirror the implementation.

## Guardrail

- Verify the current platform schema against the primary vendor docs before shipping a new native transform or changing an existing one.
- Add tests that assert exact required keys and reject known-invalid legacy keys.
- Prefer one lightweight runtime smoke check after refresh when the vendor CLI is installed locally.
- When refreshing vendor-doc links, keep the exact URL form that was runtime-verified in the target workflow. Do not normalize links by assumption.

## Applied Here

- Codex subagent TOML requires `developer_instructions`, but the renderer emitted `instructions`.
- The test encoded the same mistake, so the bug escaped until the real Codex CLI loaded the generated files.
- During docs refresh work, a link was rewritten into a different URL form because it looked canonical, even though the working form had not been re-verified in the target markdown workflow.

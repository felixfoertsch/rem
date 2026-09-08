# Bundled go-eventkit

- Source: https://github.com/BRO3886/go-eventkit
- Tag: `v0.13.0`
- Commit: `53ca4713fa89327a64eaecf301fa2ff90c76d82a`
- License: MIT; original `LICENSE` retained.

This directory is a source snapshot tracked by the rem repository, not a submodule or a Git subtree merge. It retains its own `go.mod` and upstream import path. The parent module uses a local `replace`; no separate checkout or `go.work` is required. Upstream has no external module dependencies at this revision, so no `go.sum` is needed here.

The import preserves upstream source, tests, scripts, and docs. Upstream `CLAUDE.md` and `.claude/` agent configuration were omitted; the parent `AGENTS.md` governs this checkout. No native source changes were made during import. The rem collaboration bridge remains in the parent `internal/reminderkit/` pending a separate migration.

Run `go test ./...` and `go vet ./...` from this directory to test it independently. Root Go commands do not traverse nested modules. Upstream integration/demo scripts access real Calendar/Reminders data: do not execute them without explicit approval.

For an upstream update, fetch an exact tag/commit into a separate checkout, compare against the recorded revision, review and apply the delta to this directory while preserving local changes and these import exclusions, update this record and the root required version, then test both modules. Do not overwrite the directory blindly or introduce merge commits into the parent history.

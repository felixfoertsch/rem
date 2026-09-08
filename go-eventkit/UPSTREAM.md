# Source provenance

Originally imported from https://github.com/BRO3886/go-eventkit at `v0.13.0`, commit `53ca4713fa89327a64eaecf301fa2ff90c76d82a`. Original MIT license and copyright notices remain. Upstream agent configuration was omitted.

`go-eventkit` is now maintained here as an independent module, not an upstream mirror, submodule, or subtree merge. No upstream compatibility or synchronization requirement applies. Its canonical module path is `github.com/felixfoertsch/rem/go-eventkit`; the parent uses a local `replace` and `v0.0.0` placeholder. No separate checkout, library release tag, or `go.work` is required.

The `reminderkit/` package owns native collaboration types, roster lookup, assignment transactions, sections, diagnostics, and save/readback verification. Synthetic native tests live in `scripts/reminderkit-tests.m`. CLI participant text resolution remains in rem; this module does not depend on rem.

Run `go test -race ./...` and `go vet ./...` here to check the library independently. Root Go commands do not traverse nested modules. Integration/demo scripts access real Calendar/Reminders data: do not execute them without explicit approval.

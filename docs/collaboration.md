# Shared-reminder assignments and sections (experimental)

This branch adds a native collaboration bridge, CLI commands, and assignment display. It is not a claim that all upstream issues are solved. New **write commands require `--experimental`** until live-account validation is complete. Read-only metadata and `doctor` do not require this flag.

## Build and try

On macOS with Go (the version in `go.mod`) and Xcode Command Line Tools:

```sh
git fetch origin
git switch feat/shared-reminders-and-upstream-fixes
go build -o bin/rem ./cmd/rem
bin/rem doctor
bin/rem participants --list "Family" --output json
bin/rem list --list "Family" --output json
```

Substitute an actual list name or its exact ID. Use a disposable reminder in a test shared list for writes, not a production task:

```sh
bin/rem assign REMINDER_ID existing-participant@example.com --experimental
bin/rem assign REMINDER_ID me --experimental
bin/rem show REMINDER_ID --output json
bin/rem assign REMINDER_ID --none --experimental
```

These are separate operations, not a script to run blindly. Inspect the result in Reminders.app and on the other participant's device after each write. `me` requires a positively resolved native current-user identity. An unresolved or ambiguous identity fails before saving; the implementation never guesses from the shell username or a matching contact name.

Participant resolution uses exact ID, exact case-insensitive email (including `mailto:`), or exact case-insensitive display name. Duplicate names are rejected. No fuzzy person matching, invitations, or sharing-permission changes are performed. Reminder prefixes must uniquely identify a reminder; the resolved full reminder ID and list ID are pinned before the write. Native participant IDs are rechecked against that list immediately before preparing the transaction.

## Read and write contracts

`rem show`, `rem list`, their table/plain output, and JSON export include assignment metadata. JSON has these additional fields:

```json
{
  "assigned_to": {
    "id": "LIST-SCOPED-PARTICIPANT-ID",
    "name": "Example Participant",
    "address": "person@example.com",
    "access_level": 0,
    "is_me": false
  },
  "assignment_available": true
}
```

The sample `access_level` is an opaque native numeric value, not a stable permission enum. `assigned_to: null` means **unassigned only when `assignment_available` is true**. If a private API cannot be read, availability is false and `assignment_error` explains why. Core reads still succeed and emit a warning on stderr; JSON stdout remains parseable. An orphaned assignee ID remains visible instead of being mislabeled unassigned. Participant names are escaped in terminal output.

Assignment and section writes are different: an unavailable API, read-only list, missing participant, unresolved caller, save error, or failed verification returns a nonzero exit status. Assigning the current assignee again, or clearing an already empty assignment, does not prepare a new save. The old assignment is removed and the replacement prepared in **one** REMSaveRequest transaction, not two independently committed writes.

After a successful native save the bridge fetches the object again and verifies the requested assignment/name. A verification failure explicitly says the save succeeded and advises inspection before retrying. It is not reported as a rollback. Readback checks the local native store; it does **not** prove remote iCloud synchronization or correct notification delivery.

JSON import intentionally does not replay collaboration metadata: participant IDs belong to the original shared list. The existing cross-list copy/delete move path also does not preserve assignments. Inspect and explicitly reassign a moved/imported task in its destination list.

## Sections: limited scope

```sh
bin/rem sections --list "Family" --output json
bin/rem section create "Planning" --list "Family" --experimental
bin/rem section rename "Planning" "Next" --list "Family" --experimental
```

Listing, creation, and renaming are implemented. **Per-reminder section membership, `update --section`, filtering by section, and section deletion are not implemented.** The available native change context exposes unsaved membership/order structures; treating those as the complete persisted membership would risk overwriting list organization. No fabricated `section` JSON field or ineffective move/delete flag is provided.

## Architecture and verification limits

Core reminder operations remain in `go-eventkit`. The narrow `internal/reminderkit` extension is an explicit exception to the repository's usual all-capabilities-through-go-eventkit architecture, because the pinned dependency does not expose assignments/sections. It should be extracted into that library once the API and live-account behavior are validated. It follows the existing backing-object/ReminderKit approach; it does not use AppleScript, spawn helper commands, edit SQLite, or bypass macOS privacy controls.

All native operations are serialized. The bridge checks selectors and method signatures, uses KVC for dynamic properties, validates backing-object types, and propagates exceptions/errors. `rem doctor` reads authorization status and selector availability without creating an event store or requesting access. Presence of a selector is not evidence that a particular account supports the operation.

Important remaining validation:

- The assignment status argument is provisionally `0`. Synthetic native storage accepts it, but the same experiment accepts other statuses too. The semantic value used by Reminders.app, notification behavior, and cloud propagation must be checked against a real assignment before removing the experimental gate.
- Current-user identity must be verified for both lists owned by the caller and lists shared to the caller. A native participant string that cannot be mapped to a known participant causes a safe error; shared-to-me support is not yet established end to end.
- No automated test in this PR logs into iCloud, reads a user's reminders, sends invitations, or mutates a real shared list. Tests cover parsing/resolution, failure propagation, availability/JSON/UI contracts, permission-free commands, native calling conventions, and synthetic native storage only.
- Private APIs may change on newer macOS releases. This is not an App Store-compatible/public-API guarantee.

## Live acceptance checklist before release

Create a disposable shared list manually in Reminders.app. Record the macOS version and `rem doctor` output without publishing private participant addresses. Verify participant enumeration and `me` on both owner and invitee accounts. Assign an existing test task to self, another participant, a different participant, and then nobody. After each operation check local app display, a fresh CLI process, the other participant's device, and unchanged title, notes, due date, alarms, tags, priority, and completion state. Repeat the same assignment to confirm idempotence. Verify read-only/unshared lists and unknown/ambiguous names fail without changes. Create/rename a disposable section and confirm other sections and reminder placement remain unchanged. Record assignment status semantics from an app-created assignment before enabling writes by default.

Run the automated checks on macOS:

```sh
go test -race ./...
go vet ./...
mkdir -p bin
xcrun clang -fobjc-arc -fblocks -fsanitize=address,undefined \
  -framework Foundation -framework EventKit \
  scripts/reminderkit-tests.m -o bin/reminderkit-tests
bin/reminderkit-tests
```

## Upstream issue and PR disposition

| Upstream item | Work in this branch | Still needed |
|---|---|---|
| [#56: assignments](https://github.com/BRO3886/rem/issues/56) | Native read/write bridge, participants/assign commands, show/list/JSON, explicit unassignment, exact identity resolution and save verification. | Live owner/invitee testing, status semantics and iCloud notification/sync verification. Experimental, not production-verified. |
| [#41: permission failure](https://github.com/BRO3886/rem/issues/41) | Removed package-load EventKit initialization; help/version/skills/completions/doctor run without access. Added diagnostics and troubleshooting. | A third-party GUI host's missing entitlements or macOS permission attribution cannot be repaired by the CLI. No signing/notarization change is claimed. |
| [PR #67: date commands](https://github.com/BRO3886/rem/pull/67) | Ported the exact skill change from @itaysk, with credit in the commit. | Upstream maintainer review/merge remains separate from this fork. |
| [#38: sections](https://github.com/BRO3886/rem/issues/38) | Native listing plus experimental create/rename. | Membership, move/filter/delete and live validation. Partially implemented. |
| [#68: section in output](https://github.com/BRO3886/rem/issues/68) | Investigated native membership storage; no inaccurate output field added. | Actual per-reminder section resolution and JSON/table integration. **Not fixed.** |
| [#27: feature wishlist](https://github.com/BRO3886/rem/issues/27) | Assignment work covers one part of the wishlist; existing features are not reimplemented. | Remaining wishlist features are outside this change and are **not claimed fixed**. |

No upstream issue is automatically closed by this branch. See [troubleshooting](troubleshooting.md) for the permission setup and host limitations.

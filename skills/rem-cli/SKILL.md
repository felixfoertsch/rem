---
name: rem-cli
description: Manage macOS Reminders with the felixfoertsch/rem fork. Use for reminder CRUD, assignment to shared-list participants, unassignment, native list sections, permission diagnostics, and reminder automation.
license: MIT
compatibility: Requires macOS with felixfoertsch/rem installed
metadata:
  author: BRO3886, felixfoertsch
  homepage: https://github.com/felixfoertsch/rem
---

# rem — macOS Reminders

Use this fork's CLI, not upstream binaries. Inspect `rem version` and unfamiliar command `--help` before acting. Main snapshots enable collaboration writes by default; no `--experimental` flag. Private APIs remain experimental and account-dependent.

## Install and permissions

Download a successful main artifact from [Build binaries](https://github.com/felixfoertsch/rem/actions/workflows/build.yml), following [build instructions](https://github.com/felixfoertsch/rem/blob/main/docs/builds.md). Artifacts include both Mac architectures and checksums, expire after 90 days, and are not Developer ID signed or notarized. Releases are manual.

Help, version, completions, skills, and `rem doctor` need no Reminders access. Data commands initialize access lazily. Doctor reports authorization and native capability presence, not proof of live account support.

If denied, ask the user to run `rem lists` in a supported terminal and approve the system prompt. Check System Settings > Privacy & Security > Reminders for the launching terminal/IDE/agent host. Never edit TCC, disable SIP, reset unrelated permissions, or claim the CLI can repair another host's entitlements.

## Commands

| Intent | Command |
|---|---|
| Create | `rem add "TITLE" --list "LIST" -o json` |
| Read | `rem list --list "LIST" --incomplete -o json`; `rem show <id> -o json` |
| Search | `rem search "QUERY" --list "LIST" -o json` |
| Due work | `rem today`; `rem overdue`; `rem upcoming --days 7` |
| Edit | `rem update <id> --title "TITLE" --notes "NOTES"` |
| Complete/undo | `rem complete <id>...`; `rem uncomplete <id>...` |
| Flag/undo | `rem flag <id>...`; `rem unflag <id>...` |
| Delete | `rem delete <id>...` (confirmation required) |
| Lists | `rem lists -o json`; `rem list-mgmt create "LIST"` |
| Participants | `rem participants --list "LIST" -o json` |
| Assign/clear | `rem assign <id> me -o json`; `rem assign <id> --none -o json` |
| Sections | `rem sections --list "LIST" -o json` |
| Create/rename section | `rem section create "NAME" --list "LIST"`; `rem section rename "OLD" "NEW" --list "LIST"` |
| Export/import | `rem export --list "LIST" --format json`; `rem import file.json --dry-run` |

Full flags: [references/commands.md](references/commands.md). Relative dates: [references/dates.md](references/dates.md). Pass natural language directly to date flags; read the current local clock when exact ISO boundaries are needed. Do not guess dates or execute illustrative mutation commands blindly.

## Assignment workflow

1. Resolve the reminder and list. Enumerate that list's participants, projecting only needed fields rather than exposing unrelated addresses.
2. Use native `is_me` and the selector `me` for the current user. Otherwise use exact list-scoped participant ID, case-insensitive email (including `mailto:`), or case-insensitive name. Ambiguous names and unresolved identity fail; never infer self from shell username or invite someone to make assignment work.
3. For create-and-assign, capture `id` from `rem add ... -o json`, then call `rem assign <id> me -o json`. Assignment is not an add/update flag. If assignment fails, report the created ID; do not create duplicates or delete it without approval.
4. Read `rem show <id> -o json` in a separate command. Require `assignment_available: true` and matching `assigned_to.id`, or `assigned_to.is_me: true` for self. For explicit unassignment, verify availability is true and `assigned_to` is null.

`assigned_to: null` means unassigned only when `assignment_available` is true. Otherwise report unknown/unavailable and inspect `assignment_error`. Core reads can succeed with warnings; assignment writes fail on permission, identity, save, or verification errors. `access_level` is opaque, not a stable permission enum.

A post-save verification failure is not a rollback: inspect before retrying. Repeating the same assignment is a no-op. Local readback does not prove iCloud synchronization, notification delivery, or owner/invitee compatibility; use approved disposable data and inspect other devices when testing those properties.

JSON export includes assignment metadata; import ignores it because participant IDs are list-scoped. Shared-list copy/delete moves do not preserve assignment. Resolve destination participants and explicitly reassign only when requested.

## Limits and data safety

- Sections support listing, creation, and renaming only. No per-reminder membership, `update --section`, section filtering/deletion, or per-reminder section JSON field exists.
- Shared-list moves copy/delete with a new reminder ID. Explain this and obtain user confirmation before passing `--force`/`--yes`/`-y`; re-resolve the new ID afterward. Ordinary moves preserve IDs.
- Use `-o json` for scripts. Core reads support table/JSON/plain; collaboration commands use JSON or command-specific text; doctor always emits JSON. `REM_NO_UPDATE_CHECK=1` disables release notices; `NO_COLOR=1` disables colors. Preserve stderr warnings while parsing stdout.
- Use one multi-ID command for complete/uncomplete/flag/unflag/delete, not a per-ID loop. Keep independent calls separate; do not hide failures behind chained header output.
- `--due` auto-attaches a due-time alarm; `--silent` disables it. Do not add `--remind-me 0m` to enable the default. Clear due dates with `--due none`.
- Time alarms and geofences are independent: `--remind-me none` preserves geofences; `--location none` preserves time alarms. Locations require coordinates and enabled Location Services to fire; ask for personal locations, never guess them.
- Use `--url` for the native URL field. Flags/tags use private APIs and can warn on partial failure. Title hashtags are additive; numeric `#42` is ignored.
- Prefer `--priority high|medium|low|none`. `-F` means flagged, `-f` force on deletion/shared moves, `-f` format on export, `-O` export output file, `-t` title on update and tags on add.
- Mutation confirmation remains required. Do not overwrite externally managed skills with `rem skills install`; use its dry-run to preview.

Report created short IDs and verified outcomes. Never infer remote synchronization from local success.

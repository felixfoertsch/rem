# rem — fork contributor instructions

## Scope and ownership

- This repository is `felixfoertsch/rem`, a fork of `BRO3886/rem`. Preserve upstream attribution and the MIT license.
- Both modules are maintained here: `github.com/felixfoertsch/rem` and `github.com/felixfoertsch/rem/go-eventkit`. The library builds independently in `go-eventkit/`, selected by the root local `replace`; `v0.0.0` is a local dependency placeholder, not a release. No upstream compatibility or synchronization requirement applies. Preserve MIT notices and source provenance in `go-eventkit/UPSTREAM.md`.
- Use neutral names (`rem`, `go-eventkit`, `reminderkit`) in APIs, commands, and prose. Repository-owner names belong only in canonical module/import paths, repository URLs, and attribution.
- `AGENTS.md` is the sole repository instruction file. Documentation lives in `README.md`, `docs/`, and `skills/rem-cli/`; no hosted website or external installer is maintained here.
- Keep patches small, follow existing Go conventions, and preserve unrelated work. Commit messages use Conventional Commits: `type(scope): description`.
- Do not push, tag, publish releases, or change repository hosting settings without explicit authorization.

## Architecture

- `cmd/rem/commands/`: Cobra commands; shared interactive helpers in `huh_helpers.go` and batch mutations in `batch.go`.
- `internal/service/`: core reminder/list operations via `go-eventkit`; no AppleScript runtime.
- `go-eventkit/reminderkit/`: native collaboration types, guarded Objective-C/cgo transactions, participants, sections, diagnostics, and save/readback verification.
- `internal/reminderkit/`: CLI participant selectors and assignment orchestration only; no native code or serialized bridge protocol.
- `internal/reminder/`: domain models and collaboration metadata; `internal/export/`: JSON/CSV; `internal/ui/`: terminal formatting.
- `internal/skills/` and root `skills.go`: embedded skill installation/status; `skills/rem-cli/` is the distributable instruction source.
- `internal/update/`: best-effort release notices, not an artifact installer. Keep notices scoped to this fork.
- Keep one executable. Do not add helper processes, direct SQLite writes, or privacy-control bypasses for collaboration.

## Native safety and permissions

- Initialize EventKit lazily for data commands. Help, version, completions, skills, and doctor must work without Reminders access.
- `doctor` reports authorization and selector availability without opening an event store or requesting access. Capability presence is not a live-account test.
- Preserve serialized native operations, selector/signature checks, backing-object validation, exception handling, and native save/readback verification.
- Permission may belong to the responsible terminal/IDE/agent host. A CLI cannot repair another application's entitlements. Never edit TCC, disable SIP, or reset unrelated permissions.
- Do not read or mutate real reminders in automated tests. Live tests require user-approved disposable data; project only necessary fields and never publish participant addresses or reminder contents.

## Collaboration contract

- `main` is experimental by definition. Assignment and section writes work by default; do not restore an `--experimental` gate.
- Participants are existing list members, not contacts to invite. Resolve exact list-scoped ID, case-insensitive email/name, or native `me`; reject ambiguity and unresolved identity. Never infer self from a shell username.
- Pin full reminder/list identities before writes and revalidate participants against that list. Assignment replacement is one transaction; repeating an existing assignment or clearing an empty assignment is a no-op.
- `assigned_to: null` means unassigned only when `assignment_available` is true. Preserve unavailable/orphaned metadata, `assignment_error`, stderr warnings, and parseable JSON stdout.
- Core reads may succeed when collaboration metadata is unavailable. Collaboration writes must fail on permission, identity, save, or verification errors. A successful save followed by failed verification is not a rollback; tell the caller to inspect before retrying.
- JSON export includes assignment metadata; import deliberately does not replay list-scoped participant IDs. Shared-list copy/delete moves do not preserve assignments.
- Sections support list/create/rename only. Membership, reminder moves into sections, section filtering/deletion, and per-reminder section output are not implemented. Never rewrite ordering structures to imitate them.
- Local native readback does not establish iCloud sync, notifications, assignment-status semantics, or owner/invitee compatibility. Follow `docs/collaboration.md` before claiming release validation.

## Existing reminder behavior to preserve

- `go-eventkit` fields are `Title`, `Notes`, `List`, and `URL`; CLI JSON uses its own domain tags. Verify the actual schema rather than mixing them.
- URLs use the native ReminderKit attachment path exposed by go-eventkit, not a notes-only workaround. Flags/tags use private APIs and warn on partial failure; genuine core errors remain fatal.
- Title hashtags are additive native tags; numeric fragments such as `#42` are ignored.
- Shared-list moves require confirmation and copy/delete with a new reminder ID. Preserve the original until the copy succeeds and report the new ID. Plain moves remain ID-stable.
- `--due` automatically adds a due-time alarm unless `--silent` is set. Time alarms and geofences are separate buckets; clearing one must preserve the other. Location input is coordinates, not geocoded text.
- Priority values: 0 none, 1–4 high, 5 medium, 6–9 low. Prefer named CLI priorities.
- Preserve multi-ID validation and per-item failure reporting for complete/uncomplete/flag/unflag/delete. Do not silently weaken confirmation boundaries.
- Preserve flag meanings: `-F` flagged; `-f` force on destructive/shared-move commands, format on export; `-t` title on update, tags on add; `-O` export output file. Inspect command help for exact options.
- Core reads support table/JSON/plain; collaboration commands use JSON or command-specific text; doctor emits JSON. Respect `NO_COLOR` and escape external names in terminal output.

## Build and verification

Use macOS, the Go version in `go.mod`, and Xcode Command Line Tools. Dependencies are pinned in `go.mod`/`go.sum`. Do not install alternate global runtimes for a test.

```fish
make build
make test
make lint
python3 -m unittest discover -s scripts/ci -p 'test_*.py' -v
```

- Run focused tests for changed behavior, then relevant neighboring tests. Native bridge changes also require the race/sanitizer checks documented in `docs/collaboration.md` and `.github/workflows/test.yml`.
- Use `olekukonko/tablewriter` v1 APIs (`NewTable`, `Header`, `Append`, `Render`), not the old `SetHeader` API.
- Keep embedded skills, help, README, and command references consistent. Test skill embedding after documentation changes. `rem skills install --dry-run` previews without overwriting managed user skills.
- `make completions` generates ignored shell completions locally; do not commit generated copies. Do not claim live account behavior from permission-free or synthetic tests.

## Artifacts and releases

- `.github/workflows/build.yml` builds arm64 and amd64 archives for every newly reachable main commit, including multi-commit pushes. PRs validate packaging but do not publish artifacts.
- Keep artifact names commit-specific, include checksums/build metadata, and preserve independent runs. Retention is 90 days; artifacts are not permanent releases.
- CI snapshots report `main-<short-sha>` plus the full commit. They are not Developer ID signed or notarized. Do not claim signing or hosting that has not been configured.
- Release/tag creation stays manual. For an authorized release: test first, push main, create/push the version tag, build with `make release`, verify archive architecture and embedded version, then explicitly publish to `felixfoertsch/rem`.
- Never relabel a main artifact as a tagged release. Use CalVer for future binary releases, retain `main-<short-sha>` for artifacts, and keep linear history. The locally replaced library needs no separate release tags. See `docs/builds.md` for download and validation details.

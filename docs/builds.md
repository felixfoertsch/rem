# Per-commit macOS binaries

The [Build binaries workflow](https://github.com/felixfoertsch/rem/actions/workflows/build.yml)
builds this fork automatically after every push to `main`. Pull requests validate
packaging but do not publish binary artifacts. **Tags and GitHub Releases remain
manual**; this workflow has only `contents: read` permission and no release step.

## Download and install

Open a successful **Build binaries** run, then download the artifact named
`rem-macos-<full-commit-sha>` from its **Artifacts** section or its job summary.
GitHub requires you to be signed in to download Actions artifacts. Each artifact
contains:

- `rem-darwin-arm64.tar.gz` for Apple Silicon.
- `rem-darwin-amd64.tar.gz` for Intel Macs.
- `SHA256SUMS` for both archives.

After unzipping the Actions download, verify and extract the appropriate tarball:

```sh
shasum -a 256 -c SHA256SUMS
# Apple Silicon; substitute amd64 on an Intel Mac.
tar -xzf rem-darwin-arm64.tar.gz
./rem version
./rem doctor
mkdir -p "$HOME/.local/bin"
install -m 755 rem "$HOME/.local/bin/rem"
```

Each tarball preserves executable permissions and contains `rem`, its MIT license,
and `BUILD-INFO.txt` with the full source commit, build time, architecture, workflow
run, Go toolchain and dependency information. `rem version` reports
`main-<12-character-sha>` and the full commit. No automatic version tags are made.
The CI build checks the Mach-O architecture of both executables, executes the
native-host `version` command, and verifies archive extraction, executable mode,
and checksums before upload. The Intel build is cross-compiled with cgo/clang;
its architecture is checked, but it is not runtime-tested on an Intel Mac.

These are development snapshots, **not Developer ID-signed or notarized releases**.
macOS Gatekeeper and Reminders permission requirements still apply. Shared-list
assignment writes still require `--experimental`; producing binaries does not
replace the live-account acceptance checklist in [collaboration.md](collaboration.md).
The upstream Homebrew/install-script instructions install upstream releases, not
these development snapshots.

Artifacts are retained for **90 days**. They are not permanent release storage.
Download copies you need to retain longer. You can re-run a historical workflow
run to rebuild its original commits; **Run workflow** on `main` rebuilds the
current tip without creating a release.

## Every commit, including multi-commit pushes

A push containing several commits produces one artifact per newly reachable
commit, not just for `github.sha`. The planner uses the complete Git range
`before..after`, including newly merged side-branch commits, rather than the
potentially truncated webhook commit list. Each matrix job checks out its exact
SHA and builds both architectures. The workflow code handles packaging directly
so earlier commits do not need to contain the newly introduced CI scripts.

There is deliberately **no concurrency group** on the artifact workflow: a newer
push cannot cancel or replace an older pending build. Matrix `fail-fast` is false,
so one broken commit does not cancel the others. Only successful builds produce
binary artifacts; failures remain visible instead of being packaged as successes.
The separate Tests workflow can still cancel superseded test runs.

This starts with commits introduced to `main` by the push that installs the
workflow, not a retrospective rebuild of the repository's entire history. Initial
branch creation establishes its tip as a baseline. Missing Git history fails
explicitly instead of silently dropping commits. GitHub permits at most 256
matrix jobs per workflow; larger pushes are rejected explicitly, not truncated.

## Local checks

The planner uses only Python's standard library and Git:

```sh
python3 -m unittest discover -s scripts/ci -p 'test_*.py' -v
```

Its disposable Git-history tests cover multi-commit pushes, merges, force pushes,
branch creation/deletion, incomplete webhook commit lists, input validation, and
matrix limits. The existing macOS Tests workflow separately runs the Go tests,
race detector, vet, and synthetic native API tests.

## Manual releases

Publishing a release remains a separate deliberate operation. Create the version
tag manually and build with that tagged version before uploading assets to a
manually created GitHub Release. Do not relabel a `main-<sha>` snapshot as a tagged
release: the embedded version would not match. No automatic tag, release,
prerelease, or moving `latest` download is created by `build.yml`.

References: [GitHub artifact downloads](https://docs.github.com/en/actions/managing-workflow-runs-and-deployments/managing-workflow-runs/downloading-workflow-artifacts),
[artifact retention](https://docs.github.com/en/actions/tutorials/store-and-share-data),
[matrix limits](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstrategymatrix).

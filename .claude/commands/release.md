Prepare a manual release of `felixfoertsch/rem`. Follow `AGENTS.md` and `docs/builds.md`; main CI artifacts are not releases.

1. Inspect status, tags, and the full proposed release diff. Preserve upstream SemVer; propose a version and obtain confirmation.
2. Run Go tests/vet and the collaboration acceptance checks appropriate to the change. Do not claim iCloud synchronization or notification validation from local readback.
3. With explicit publication approval, push main first, then create and push the version tag. Never tag an unpushed commit.
4. Run `make release` after tagging. Verify both archives, executable architectures, and the native executable's embedded version from a disposable extraction directory.
5. Publish both archives with `gh release create --repo felixfoertsch/rem`, naming the approved tag explicitly. Never relabel an existing main snapshot as a release.
6. Release notes lead with breaking changes, describe user-visible behavior and remaining validation limits, link the release assets and `docs/builds.md`, and include `https://github.com/felixfoertsch/rem/compare/<previous-tag>...<new-tag>`.

No website deployment or automatic installer is part of this workflow.

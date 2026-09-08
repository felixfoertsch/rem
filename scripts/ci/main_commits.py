#!/usr/bin/env python3
"""Select every commit introduced by a main push, not just its final SHA."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys

ZERO_SHA = "0" * 40
MAX_JOBS = 256  # GitHub Actions' matrix limit.


def checked_sha(value):
    if not isinstance(value, str) or not re.fullmatch(r"[0-9a-f]{40}", value):
        raise ValueError("Expected a full lowercase Git commit SHA")
    return value


def git(repo, *args):
    return subprocess.check_output(["git", "-C", str(repo), *args], text=True).strip()


def ensure_commit(repo, sha):
    checked_sha(sha)
    exists = subprocess.run(
        ["git", "-C", str(repo), "cat-file", "-e", sha + "^{commit}"],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    if exists.returncode:
        # A force push can leave the old tip outside the fetched refs. Never
        # silently fall back to building just HEAD if the range is unavailable.
        git(repo, "fetch", "--no-tags", "origin", sha)
        git(repo, "cat-file", "-e", sha + "^{commit}")


def select_commits(repo, event_name, event, head, ref):
    if event_name == "push":
        if event.get("ref") != "refs/heads/main" or event.get("deleted", False):
            return []
        before = checked_sha(event.get("before"))
        after = checked_sha(event.get("after"))
        ensure_commit(repo, after)
        if before == ZERO_SHA:
            # Initial branch creation establishes a baseline, not a backfill
            # of the repository's entire pre-existing history.
            commits = [after]
        else:
            ensure_commit(repo, before)
            commits = git(repo, "rev-list", "--reverse", "--topo-order",
                          after, "^" + before).splitlines()
    elif event_name == "pull_request":
        # Validate packaging at the PR merge SHA without publishing binaries.
        commits = [checked_sha(head)]
    elif event_name == "workflow_dispatch" and ref == "refs/heads/main":
        commits = [checked_sha(head)]
    else:
        raise ValueError("Manual builds must run on main")

    if len(commits) > MAX_JOBS:
        raise ValueError(
            f"Push contains {len(commits)} commits; GitHub permits {MAX_JOBS} "
            "matrix jobs. Split the range into smaller pushes. No commits "
            "were silently truncated."
        )
    return [checked_sha(sha) for sha in commits]


def main():
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text())
    commits = select_commits(
        Path.cwd(), os.environ["GITHUB_EVENT_NAME"], event,
        os.environ["GITHUB_SHA"], os.environ["GITHUB_REF"],
    )
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        output.write("commits=" + json.dumps(commits, separators=(",", ":")) + "\n")
        output.write("count=" + str(len(commits)) + "\n")
    print(f"Selected {len(commits)} commit(s):")
    print("\n".join(commits))


if __name__ == "__main__":
    try:
        main()
    except (KeyError, ValueError, OSError, subprocess.CalledProcessError) as error:
        print(f"Cannot plan binary builds: {error}", file=sys.stderr)
        sys.exit(1)

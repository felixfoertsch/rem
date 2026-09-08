"""Regression tests using real disposable Git histories; no network access."""

from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import main_commits as planner


class CommitSelectionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "CI Test")
        self.git("config", "user.email", "ci@example.invalid")
        self.base = self.commit("baseline")

    def git(self, *args):
        return planner.git(self.repo, *args)

    def commit(self, message):
        self.git("commit", "-q", "--allow-empty", "-m", message)
        return self.git("rev-parse", "HEAD")

    def push(self, before, after, **extra):
        event = {"before": before, "after": after, "ref": "refs/heads/main"}
        event.update(extra)
        return planner.select_commits(self.repo, "push", event, after, event["ref"])

    def test_single_commit(self):
        head = self.commit("one")
        self.assertEqual(self.push(self.base, head), [head])

    def test_every_commit_in_one_push_even_if_event_list_is_incomplete(self):
        commits = [self.commit(f"change {i}") for i in range(3)]
        self.assertEqual(self.push(self.base, commits[-1], commits=[]), commits)

    def test_merge_includes_new_side_branch_commits(self):
        self.git("checkout", "-q", "-b", "feature")
        feature = self.commit("feature work")
        self.git("checkout", "-q", "main")
        main = self.commit("main work")
        self.git("merge", "-q", "--no-ff", "feature", "-m", "merge feature")
        merge = self.git("rev-parse", "HEAD")
        commits = self.push(self.base, merge)
        self.assertEqual(set(commits), {main, feature, merge})
        self.assertEqual(commits[-1], merge)

    def test_force_push_excludes_previously_reachable_history(self):
        old = self.commit("superseded")
        self.git("checkout", "-q", "-b", "replacement", self.base)
        new = self.commit("replacement")
        self.assertEqual(self.push(old, new), [new])

    def test_reset_to_old_tip_has_no_new_commits(self):
        old = self.commit("discarded")
        self.assertEqual(self.push(old, self.base), [])

    def test_new_branch_starts_at_tip_without_historical_backfill(self):
        head = self.commit("new tip")
        self.assertEqual(self.push(planner.ZERO_SHA, head), [head])

    def test_branch_deletion_builds_nothing(self):
        self.assertEqual(self.push(self.base, planner.ZERO_SHA, deleted=True), [])

    def test_non_main_push_builds_nothing(self):
        self.assertEqual(self.push(self.base, self.base, ref="refs/heads/topic"), [])

    def test_pull_request_builds_only_merge_commit(self):
        self.assertEqual(planner.select_commits(
            self.repo, "pull_request", {}, self.base, "refs/pull/1/merge"), [self.base])

    def test_manual_rebuild_on_main(self):
        self.assertEqual(planner.select_commits(
            self.repo, "workflow_dispatch", {}, self.base, "refs/heads/main"), [self.base])

    def test_manual_rebuild_on_feature_branch_is_rejected(self):
        with self.assertRaises(ValueError):
            planner.select_commits(self.repo, "workflow_dispatch", {}, self.base,
                                   "refs/heads/topic")

    def test_invalid_sha_is_rejected_before_git(self):
        with self.assertRaises(ValueError):
            self.push("--all", self.base)
        with self.assertRaises(ValueError):
            planner.checked_sha("$(touch unwanted)")

    def test_missing_range_is_fatal_not_a_tip_only_fallback(self):
        with patch.object(planner, "ensure_commit", side_effect=RuntimeError("missing")):
            with self.assertRaises(RuntimeError):
                self.push(self.base, self.base)

    def test_matrix_limit_is_not_silently_truncated(self):
        commits = [self.commit(f"change {i}") for i in range(3)]
        with patch.object(planner, "MAX_JOBS", 2):
            with self.assertRaisesRegex(ValueError, "No commits were silently truncated"):
                self.push(self.base, commits[-1])


if __name__ == "__main__":
    unittest.main()

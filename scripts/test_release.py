import os
from pathlib import Path
import subprocess
import tempfile
import unittest


VALIDATOR = Path(__file__).with_name("validate_release.py")


class ReleaseValidationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.repo = Path(self.directory.name)
        self.git("init", "-b", "master")
        self.git("config", "user.name", "Release Test")
        self.git("config", "user.email", "release@example.test")
        self.git("commit", "--allow-empty", "-m", "initial")
        self.sha = self.git("rev-parse", "HEAD")
        self.git("update-ref", "refs/remotes/origin/master", self.sha)
        self.git("update-ref", "refs/remotes/origin/release/v0.1", self.sha)

    def git(self, *args):
        return subprocess.check_output(
            ["git", *args], cwd=self.repo, text=True, stderr=subprocess.STDOUT
        ).strip()

    def validate(self, version="v0.1.0", commit=None, retry=False):
        return subprocess.run(
            ["python3", str(VALIDATOR), version, commit or self.sha]
            + (["--retry"] if retry else []),
            cwd=self.repo, text=True, capture_output=True,
            env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"},
        )

    def test_first_release_resolves_exact_commit_and_branch(self):
        result = self.validate()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(f"commit={self.sha}\n", result.stdout)
        self.assertIn("branch=release/v0.1\n", result.stdout)

    def test_existing_version_requires_explicit_retry_and_same_commit(self):
        self.git("tag", "-a", "v0.1.0", "-m", "release")
        self.assertNotEqual(self.validate().returncode, 0)
        self.assertEqual(self.validate(retry=True).returncode, 0)
        self.git("commit", "--allow-empty", "-m", "fix")
        new_sha = self.git("rev-parse", "HEAD")
        self.git("update-ref", "refs/remotes/origin/release/v0.1", new_sha)
        self.assertNotEqual(self.validate(commit=new_sha, retry=True).returncode, 0)

    def test_only_latest_line_receives_patches_and_new_lines_start_at_zero(self):
        self.assertNotEqual(self.validate("v0.1.1").returncode, 0)
        self.git("tag", "v0.1.0")
        self.assertEqual(self.validate("v0.1.1").returncode, 0)
        self.git("update-ref", "refs/remotes/origin/release/v0.2", self.sha)
        self.assertNotEqual(self.validate("v0.2.1").returncode, 0)
        self.assertEqual(self.validate("v0.2.0").returncode, 0)
        self.git("tag", "v0.2.0")
        self.assertNotEqual(self.validate("v0.1.1").returncode, 0)
        self.assertNotEqual(self.validate("v0.1.0", retry=True).returncode, 0)

    def test_rejects_invalid_versions_and_symbolic_commit_inputs(self):
        for version in ("v0.01.0", "v0.1.0-rc.1", "v1.0.0", "0.1.0", "v0.1.0\n"):
            with self.subTest(version=version):
                self.assertNotEqual(self.validate(version).returncode, 0)
        self.assertNotEqual(self.validate(commit="master").returncode, 0)
        self.assertNotEqual(self.validate(retry=True).returncode, 0)

    def test_selected_commit_must_belong_to_remote_release_branch(self):
        self.git("commit", "--allow-empty", "-m", "development only")
        development_sha = self.git("rev-parse", "HEAD")
        self.assertNotEqual(self.validate(commit=development_sha).returncode, 0)
        self.git("update-ref", "-d", "refs/remotes/origin/release/v0.1")
        self.assertNotEqual(self.validate().returncode, 0)

    def test_release_must_include_previous_release_history(self):
        self.git("tag", "v0.1.0")
        self.git("switch", "--orphan", "unrelated")
        self.git("commit", "--allow-empty", "-m", "unrelated history")
        unrelated_sha = self.git("rev-parse", "HEAD")
        self.git("update-ref", "refs/remotes/origin/release/v0.2", unrelated_sha)
        self.assertNotEqual(self.validate("v0.2.0", commit=unrelated_sha).returncode, 0)


if __name__ == "__main__":
    unittest.main()

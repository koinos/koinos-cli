import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from draft_release import prepare_draft
from package_release import SUPPORT_FILES, TARGETS, package


class DraftReleaseTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.folder = self.root / "packages"
        self.version = "v2.1.0-rc.1"
        self.commit = "a" * 40
        self.binary = self.root / "binary"
        self.binary.write_bytes(b"test executable")
        for name in SUPPORT_FILES:
            (self.root / name).write_text("public support file")
        for target in sorted(TARGETS):
            package(self.root, self.binary, self.folder, self.version, *target, self.commit, "test Go")
        self.gh = patch("draft_release.gh").start()
        self.addCleanup(patch.stopall)
        self.gh.side_effect = [
            json.dumps([[]]),
            json.dumps({"ref": f"refs/tags/{self.version}", "object": {"sha": self.commit}}),
            "draft created",
        ]

    def prepare(self, commit=None):
        return prepare_draft(self.folder, self.version, commit or self.commit, "koinos/koinos-cli")

    def test_reserves_exact_commit_then_creates_unpublished_prerelease_with_all_assets(self):
        self.assertEqual(self.prepare(), "draft created")
        lookup, reserve, create = self.gh.call_args_list
        self.assertIn("--paginate", lookup.args[0])
        self.assertIn("--slurp", lookup.args[0])
        self.assertEqual(json.loads(reserve.args[1]), {"ref": f"refs/tags/{self.version}", "sha": self.commit})
        self.assertIn("POST", reserve.args[0])
        command = create.args[0]
        for flag in ("--draft", "--verify-tag", "--prerelease"):
            self.assertIn(flag, command)
        self.assertEqual(len([arg for arg in command if arg.endswith((".zip", ".tar.gz"))]), len(TARGETS))
        self.assertIn(str(self.folder / "SHA256SUMS"), command)
        self.assertEqual(len((self.folder / "SHA256SUMS").read_text().splitlines()), len(TARGETS))

    def test_existing_draft_without_tag_on_later_page_stops_before_writes(self):
        self.gh.side_effect = [json.dumps([[{"tag_name": "v1.0.0"}], [{"tag_name": self.version, "draft": True}]])]
        with self.assertRaisesRegex(ValueError, "already exists"):
            self.prepare()
        self.assertEqual(self.gh.call_count, 1)

    def test_failed_release_lookup_stops_before_writes(self):
        self.gh.side_effect = subprocess.CalledProcessError(1, "gh api")
        with self.assertRaises(subprocess.CalledProcessError):
            self.prepare()
        self.assertEqual(self.gh.call_count, 1)

    def test_existing_tag_rejection_stops_before_draft_creation(self):
        self.gh.side_effect = [json.dumps([[]]), subprocess.CalledProcessError(1, "gh api POST refs")]
        with self.assertRaises(subprocess.CalledProcessError):
            self.prepare()
        self.assertEqual(self.gh.call_count, 2)
        self.assertFalse((self.folder / "SHA256SUMS").exists())

    def test_wrong_tag_response_stops_before_draft_creation(self):
        self.gh.side_effect = [json.dumps([[]]), json.dumps({"ref": f"refs/tags/{self.version}", "object": {"sha": "b" * 40}})]
        with self.assertRaisesRegex(ValueError, "tested commit"):
            self.prepare()
        self.assertEqual(self.gh.call_count, 2)

    def test_changed_archive_stops_before_api_calls(self):
        archive = next(self.folder.glob("*.zip"))
        archive.write_bytes(archive.read_bytes() + b"tampered")
        with self.assertRaisesRegex(ValueError, "Checksum mismatch"):
            self.prepare()
        self.gh.assert_not_called()

    def test_wrong_source_commit_stops_before_api_calls(self):
        with self.assertRaisesRegex(ValueError, "requested source"):
            self.prepare(commit="b" * 40)
        self.gh.assert_not_called()

    def test_missing_target_stops_before_api_calls(self):
        next(self.folder.glob("*.zip")).unlink()
        with self.assertRaisesRegex(ValueError, "each release target"):
            self.prepare()
        self.gh.assert_not_called()


if __name__ == "__main__":
    unittest.main()

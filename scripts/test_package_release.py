import hashlib
from pathlib import Path
import stat
import tarfile
import tempfile
import unittest
import zipfile

from package_release import SUPPORT_FILES, TARGETS, package, validate_version


class PackageReleaseTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.binary = self.root / "input-binary"
        self.binary.write_bytes(b"binary fixture")
        for name in SUPPORT_FILES:
            (self.root / name).write_bytes(f"public content of {name}\n".encode())
        (self.root / "private.wallet").write_bytes(b"must never be packaged")
        (self.root / ".env").write_bytes(b"must never be packaged")

    def build(self, target_os="darwin", arch="arm64", output="output"):
        return package(self.root, self.binary, self.root / output, "v2.1.0-rc.1", target_os, arch, "a" * 40, "go1.27.2")

    def test_platform_contents_permissions_and_checksum(self):
        for target_os, arch in sorted(TARGETS):
            with self.subTest(target_os=target_os, arch=arch):
                archive = self.build(target_os, arch, f"{target_os}-{arch}")
                executable = "koinos-cli"
                expected = set(SUPPORT_FILES) | {executable, "BUILD-INFO.txt"}
                if archive.suffix == ".zip":
                    with zipfile.ZipFile(archive) as bundle:
                        self.assertEqual(set(bundle.namelist()), expected)
                        files = {name: bundle.read(name) for name in expected}
                        mode = bundle.getinfo(executable).external_attr >> 16
                        self.assertTrue(stat.S_ISREG(mode))
                        self.assertEqual(mode & 0o777, 0o755)
                else:
                    with tarfile.open(archive) as bundle:
                        self.assertEqual(set(bundle.getnames()), expected)
                        files = {name: bundle.extractfile(name).read() for name in expected}
                        self.assertEqual(bundle.getmember(executable).mode, 0o755)
                self.assertEqual(files[executable], b"binary fixture")
                for name in SUPPORT_FILES:
                    self.assertEqual(files[name], (self.root / name).read_bytes())
                self.assertIn(b"Commit: " + b"a" * 40, files["BUILD-INFO.txt"])
                expected_hash = hashlib.sha256(archive.read_bytes()).hexdigest()
                self.assertEqual(archive.with_name(archive.name + ".sha256").read_text(), f"{expected_hash}  {archive.name}\n")

    def test_rejects_invalid_or_unsafe_versions(self):
        for value in ("", "2.1.0", "v01.2.3", "../../v1.2.3", "v1.2.3\nunsafe", "v1.2.3;id", "v1.2.3/extra"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                validate_version(value)

    def test_accepts_release_and_prerelease_versions(self):
        for value in ("v2.1.0", "v3.0.0-rc.1", "v0.0.0-test"):
            validate_version(value)

    def test_refuses_to_replace_an_existing_archive(self):
        archive = self.build()
        before = archive.read_bytes()
        with self.assertRaises(FileExistsError):
            self.build()
        self.assertEqual(archive.read_bytes(), before)

    def test_missing_config_stops_before_creating_archive(self):
        (self.root / ".koinosrc").unlink()
        with self.assertRaises(ValueError):
            self.build()
        self.assertFalse((self.root / "output").exists())

    def test_refuses_unsupported_platform_or_missing_source_commit(self):
        with self.assertRaises(ValueError):
            self.build("windows", "amd64")
        with self.assertRaises(ValueError):
            package(self.root, self.binary, self.root / "output", "v2.1.0", "linux", "amd64", "unknown", "go1.27.2")


if __name__ == "__main__":
    unittest.main()

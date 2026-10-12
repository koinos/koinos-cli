"""Verify packages, reserve a new tag at the tested commit, and create a draft."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile
import zipfile

from package_release import TARGETS, archive_name, validate_version


def gh(arguments, payload=None):
    return subprocess.run(
        ["gh", *arguments], input=payload, text=True, capture_output=True,
        check=True, timeout=180,
    ).stdout


def verify_packages(folder, version, commit):
    archives = [folder / archive_name(version, *target) for target in sorted(TARGETS)]
    expected = {path.name for path in archives} | {path.name + ".sha256" for path in archives}
    if {path.name for path in folder.iterdir()} != expected:
        raise ValueError("Expected exactly one archive and checksum for each release target")
    checksums = []
    for archive, target in zip(archives, sorted(TARGETS)):
        sidecar = archive.with_name(archive.name + ".sha256")
        if any(path.is_symlink() or not path.is_file() for path in (archive, sidecar)):
            raise ValueError("Release artifacts must be regular files")
        checksum = f"{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n"
        if sidecar.read_text() != checksum:
            raise ValueError(f"Checksum mismatch: {archive.name}")
        if archive.suffix == ".zip":
            with zipfile.ZipFile(archive) as bundle:
                member = bundle.getinfo("BUILD-INFO.txt")
                if member.file_size > 4096:
                    raise ValueError("Oversized build information")
                info = bundle.read(member).decode()
        else:
            with tarfile.open(archive) as bundle:
                member = bundle.getmember("BUILD-INFO.txt")
                if not member.isfile() or member.size > 4096:
                    raise ValueError("Invalid build information")
                info = bundle.extractfile(member).read().decode()
        fields = dict(line.split(": ", 1) for line in info.splitlines())
        if (fields.get("Version"), fields.get("Commit"), fields.get("Target")) != (version, commit, "/".join(target)):
            raise ValueError(f"Package does not match the requested source and target: {archive.name}")
        checksums.append(checksum)
    return archives, "".join(checksums)


def prepare_draft(folder, version, commit, repository):
    validate_version(version)
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("Source commit must be a full Git commit ID")
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise ValueError("Expected an owner/repository name")
    archives, checksums = verify_packages(folder, version, commit)
    endpoint = f"repos/{repository}"
    # A draft may exist without a tag. Inspect every page, including drafts.
    pages = json.loads(gh(["api", "--paginate", "--slurp", f"{endpoint}/releases"]))
    if any(release["tag_name"] == version for page in pages for release in page):
        raise ValueError("This release version already exists; choose a new version")
    # Creating the ref atomically refuses an existing tag, including one created
    # after the release lookup. Never move an existing tag to a new source commit.
    ref = f"refs/tags/{version}"
    created = json.loads(gh(
        ["api", "--method", "POST", f"{endpoint}/git/refs", "--input", "-"],
        json.dumps({"ref": ref, "sha": commit}),
    ))
    if created.get("ref") != ref or created.get("object", {}).get("sha") != commit:
        raise ValueError("Could not confirm the tag points to the tested commit")
    sums = folder / "SHA256SUMS"
    with sums.open("x") as output:
        output.write(checksums)
    notes = f"""Packages built from commit {commit}.

Each archive includes the CLI, mainnet and testnet startup configurations, documentation, license, and build information. Verify downloads against SHA256SUMS.

New wallets use the updated encrypted format and cannot be opened by older CLI releases. Existing legacy wallets remain readable. Automatically discovered startup files require approval before execution.

Packages are not developer-signed or notarized. Review these notes and test the downloads before publishing this draft.
"""
    with tempfile.TemporaryDirectory(prefix="koinos-release-notes-") as temporary:
        notes_file = Path(temporary) / "notes.md"
        notes_file.write_text(notes)
        arguments = [
            "release", "create", version, *map(str, archives), str(sums),
            "--repo", repository, "--verify-tag", "--draft", "--title", version,
            "--generate-notes", "--notes-file", str(notes_file),
        ]
        if "-" in version:
            arguments.append("--prerelease")
        return gh(arguments)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--packages", type=Path, required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--repository", required=True)
    args = parser.parse_args()
    print(prepare_draft(args.packages, args.version, args.commit, args.repository))


if __name__ == "__main__":
    main()

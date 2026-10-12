"""Package the CLI and public support files, then optionally test the extracted binary."""

import argparse
import gzip
import hashlib
import io
from pathlib import Path
import re
import stat
import subprocess
import tarfile
import tempfile
import zipfile


SUPPORT_FILES = (".koinosrc", ".koinosrc.testnet", "README.md", "LICENSE.md")
TARGETS = {("linux", "amd64"), ("darwin", "amd64"), ("darwin", "arm64"), ("windows", "amd64")}


def validate_version(version):
    number = r"(?:0|[1-9][0-9]*)"
    if not re.fullmatch(rf"v{number}\.{number}\.{number}(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?", version):
        raise ValueError("Use a version such as v2.1.0 or v2.1.0-rc.1")


def archive_name(version, target_os, arch):
    platform = {"darwin": "macos", "linux": "linux", "windows": "windows"}[target_os]
    extension = "tar.gz" if target_os == "linux" else "zip"
    return f"koinos-cli.{version}.{platform}.{arch}.{extension}"


def package(root, binary, output, version, target_os, arch, commit, toolchain):
    validate_version(version)
    if (target_os, arch) not in TARGETS:
        raise ValueError("Unsupported release target")
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("Source commit must be a full Git commit ID")
    executable = "koinos-cli.exe" if target_os == "windows" else "koinos-cli"
    paths = [(binary, executable, 0o755)] + [(root / name, name, 0o644) for name in SUPPORT_FILES]
    entries = []
    for path, name, mode in paths:
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"Expected a regular release file: {path}")
        entries.append((name, path.read_bytes(), mode))
    info = f"Version: {version}\nCommit: {commit}\nTarget: {target_os}/{arch}\nToolchain: {toolchain}\n"
    entries.append(("BUILD-INFO.txt", info.encode(), 0o644))
    output.mkdir(parents=True, exist_ok=True)
    archive = output / archive_name(version, target_os, arch)
    with archive.open("xb") as destination:
        if archive.suffix == ".zip":
            with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
                for name, data, mode in entries:
                    member = zipfile.ZipInfo(name)
                    member.create_system = 3
                    member.external_attr = (stat.S_IFREG | mode) << 16
                    member.compress_type = zipfile.ZIP_DEFLATED
                    bundle.writestr(member, data)
        else:
            with gzip.GzipFile(filename="", fileobj=destination, mode="wb", mtime=0) as compressed:
                with tarfile.open(fileobj=compressed, mode="w") as bundle:
                    for name, data, mode in entries:
                        member = tarfile.TarInfo(name)
                        member.size, member.mode = len(data), mode
                        bundle.addfile(member, io.BytesIO(data))
    checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
    with archive.with_name(archive.name + ".sha256").open("x") as destination:
        destination.write(f"{checksum}  {archive.name}\n")
    return archive


def smoke_test(archive, version):
    """Run a package just created here, without startup files, wallets, or a network."""
    with tempfile.TemporaryDirectory(prefix="koinos-release-") as folder:
        root = Path(folder)
        if archive.suffix == ".zip":
            with zipfile.ZipFile(archive) as bundle:
                bundle.extractall(root)
                for member in bundle.infolist():
                    (root / member.filename).chmod((member.external_attr >> 16) & 0o777)
        else:
            with tarfile.open(archive) as bundle:
                bundle.extractall(root, filter="data")
        binary = root / "koinos-cli.exe" if (root / "koinos-cli.exe").exists() else root / "koinos-cli"
        result = subprocess.run([str(binary), "--version"], cwd=root, capture_output=True, text=True, check=True, timeout=30)
        if result.stdout.strip() != version:
            raise ValueError("Packaged executable has the wrong version")
        result = subprocess.run([str(binary), "--no-rc", "--execute", "status"], cwd=root, capture_output=True, text=True, check=True, timeout=30)
        if "Network: offline" not in result.stdout or "Wallet: locked" not in result.stdout:
            raise ValueError("Packaged executable failed the offline startup check")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--version", required=True)
    parser.add_argument("--os", required=True, choices=("linux", "darwin", "windows"))
    parser.add_argument("--arch", required=True, choices=("amd64", "arm64"))
    parser.add_argument("--commit", required=True)
    parser.add_argument("--smoke-test", action="store_true")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    toolchain = subprocess.check_output(["go", "version"], text=True).strip()
    archive = package(root, args.binary, args.output, args.version, args.os, args.arch, args.commit, toolchain)
    if args.smoke_test:
        smoke_test(archive, args.version)
    print(archive)


if __name__ == "__main__":
    main()

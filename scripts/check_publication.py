"""Validate the exact tracked publication set without printing secret values."""
import ipaddress
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path(__file__).resolve().parents[1]


def metadata():
    config = (ROOT / "fve_controller/config.yaml").read_text(encoding="utf-8")
    version = re.search(r'^version: "(\d+\.\d+\.\d+)"$', config, re.M).group(1)
    image = re.search(r'^image: (ghcr\.io/[a-z0-9_/-]+)$', config, re.M).group(1)
    return version, image


def check():
    version, image = metadata()
    package = json.loads((ROOT / "fve_controller/web/package.json").read_text(encoding="utf-8-sig"))
    assert package["version"] == version, "Frontend/config version mismatch"
    assert f"## {version}\n" in (ROOT / "fve_controller/CHANGELOG.md").read_text(encoding="utf-8")
    assert image == "ghcr.io/zabojnikm/ha-fve-controller"
    for name in ("README.md", "DOCS.md", "CHANGELOG.md", "config.yaml", "Dockerfile"):
        assert (ROOT / "fve_controller" / name).is_file(), name
    assert (ROOT / "repository.yaml").is_file()
    dockerfile = (ROOT / "fve_controller/Dockerfile").read_text(encoding="utf-8")
    for base in re.findall(r"^FROM (\S+)", dockerfile, re.M):
        assert re.search(r":\d[^@]*@sha256:[a-f0-9]{64}$", base), f"Unpinned base: {base}"
    paths = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT).decode().split("\0")
    errors = []
    forbidden = re.compile(r"(^|/)(reference-private|data|node_modules|dist|\.tools|\.pnpm-store|\.codex|\.agents)(/|$)|(^|/)\.env($|\.)|\.(db|sqlite|log|exe|zip)([-.~]|$)|(^|/)(secrets\.yaml|options\.json|AGENTS\.md|PROJECT\.md)$", re.I)
    secrets = re.compile(r"gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----|victron/N/(?!INSTALLATION_ID)[a-f0-9]{12}/")
    for name in filter(None, paths):
        if forbidden.search(name):
            errors.append(f"Forbidden tracked file: {name}")
            continue
        content = (ROOT / name).read_text(encoding="utf-8-sig")
        if secrets.search(content):
            errors.append(f"Potential secret/private identifier: {name}")
        for match in re.findall(r"(?<![\w.])(?:\d{1,3}\.){3}\d{1,3}(?![\w.])", content):
            try:
                address = ipaddress.ip_address(match)
            except ValueError:
                continue
            if address.is_private and match not in ("127.0.0.1", "0.0.0.0", "172.30.32.2"):
                errors.append(f"Private address requires review: {name}")
    assert not errors, "\n".join(sorted(set(errors)))
    print(f"Publication checks passed: {len(list(filter(None, paths)))} files, {image}:{version}")


if __name__ == "__main__":
    check()

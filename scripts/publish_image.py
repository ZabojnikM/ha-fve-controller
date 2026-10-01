"""Publish immutable version tags, allowing safe reruns for the same source SHA."""
import base64
import hashlib
import json
import os
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from check_publication import metadata
from verify_registry import ACCEPT

version, image = metadata()
name = image.removeprefix("ghcr.io/")
revision = os.environ["GITHUB_SHA"]
credentials = base64.b64encode((os.environ["GITHUB_ACTOR"] + ":" + os.environ["GH_TOKEN"]).encode()).decode()
query = urllib.parse.urlencode({"service": "ghcr.io", "scope": f"repository:{name}:pull,push"})
request = urllib.request.Request("https://ghcr.io/token?" + query, headers={"Authorization": "Basic " + credentials})
with urllib.request.urlopen(request) as response:
    token = json.load(response)["token"]


def fetch(path):
    request = urllib.request.Request(f"https://ghcr.io/v2/{name}/{path}", headers={"Authorization": "Bearer " + token, "Accept": ACCEPT})
    try:
        with urllib.request.urlopen(request) as response:
            return response.read()
    except urllib.error.HTTPError as error:
        if error.code == 404:
            return None
        raise


def run(*args):
    subprocess.run(args, check=True)


mode = sys.argv[1]
if mode in ("amd64", "aarch64"):
    tag = version + "-" + mode
    existing = fetch("manifests/" + tag)
    if existing:
        config = json.loads(fetch("blobs/" + json.loads(existing)["config"]["digest"]))
        assert config["config"]["Labels"]["org.opencontainers.image.revision"] == revision, "Version already belongs to another commit; choose a new version"
        print("Reusing immutable image " + tag)
    else:
        run("docker", "tag", "fve-test:local", image + ":" + tag)
        run("docker", "push", image + ":" + tag)
elif mode == "manifest":
    refs = []
    for arch in ("amd64", "aarch64"):
        raw = fetch("manifests/" + version + "-" + arch)
        assert raw, "Missing tested architecture"
        config = json.loads(fetch("blobs/" + json.loads(raw)["config"]["digest"]))
        assert config["config"]["Labels"]["org.opencontainers.image.revision"] == revision
        refs.append(image + "@sha256:" + hashlib.sha256(raw).hexdigest())
    existing = fetch("manifests/" + version)
    if existing:
        expected = {ref.split("@")[1] for ref in refs}
        assert {m["digest"] for m in json.loads(existing)["manifests"]} == expected, "Refusing to overwrite a published version"
    else:
        run("docker", "buildx", "imagetools", "create", "--tag", image + ":" + version, *refs)
else:
    raise ValueError("Unknown publishing mode")

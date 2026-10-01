"""Anonymous GHCR manifest AND layer access. Never uses credentials."""
import argparse
import hashlib
import json
import urllib.parse
import urllib.request

ACCEPT = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"


def verify(image, version):
    assert image.startswith("ghcr.io/")
    name = image.removeprefix("ghcr.io/")
    url = "https://ghcr.io/token?" + urllib.parse.urlencode({"service": "ghcr.io", "scope": f"repository:{name}:pull"})
    with urllib.request.urlopen(url) as response:
        token = json.load(response)["token"]
    def fetch(path):
        request = urllib.request.Request(f"https://ghcr.io/v2/{name}/{path}", headers={"Authorization": f"Bearer {token}", "Accept": ACCEPT})
        with urllib.request.urlopen(request) as response:
            return response.read()
    index = json.loads(fetch("manifests/" + version))
    platforms = {}
    for manifest in index["manifests"]:
        platform = manifest.get("platform", {})
        if platform.get("os") == "linux" and platform.get("architecture") in ("amd64", "arm64"):
            digest = manifest["digest"]
            data = fetch("manifests/" + digest)
            assert "sha256:" + hashlib.sha256(data).hexdigest() == digest
            body = json.loads(data)
            config = json.loads(fetch("blobs/" + body["config"]["digest"]))
            assert config["config"]["Labels"]["org.opencontainers.image.version"] == version
            # Download every layer anonymously; proves installation does not need a login.
            for layer in body["layers"]:
                data = fetch("blobs/" + layer["digest"])
                assert "sha256:" + hashlib.sha256(data).hexdigest() == layer["digest"]
            platforms[platform["architecture"]] = digest
    assert set(platforms) == {"amd64", "arm64"}, platforms
    print(json.dumps({"image": image, "version": version, "platforms": platforms}, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("image")
    parser.add_argument("version")
    args = parser.parse_args()
    verify(args.image, args.version)

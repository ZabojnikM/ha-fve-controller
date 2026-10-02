"""Run the actual native image; test API, relative assets, /data and ingress ACL."""
import json
import pathlib
import re
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


image = sys.argv[1]
with tempfile.TemporaryDirectory() as directory:
    last_balance = "2026-01-01T00:00:00Z"
    with sqlite3.connect(pathlib.Path(directory) / "simulation.db") as db:
        db.execute("CREATE TABLE settings(key TEXT PRIMARY KEY,value TEXT NOT NULL)")
        db.execute("INSERT INTO settings VALUES('simulation_last_balance',?)", (last_balance,))
    previous_at = None
    for ingress in (False, False, True):
        args = ["run", "-d", "-p", "127.0.0.1::8099", "-v", f"{directory}:/data"]
        if not ingress:
            args += ["-e", "INGRESS_ONLY=false"]
        container = docker(*args, image)
        try:
            binding = json.loads(docker("inspect", container))[0]["NetworkSettings"]["Ports"]["8099/tcp"][0]
            base = "http://127.0.0.1:" + binding["HostPort"]
            for attempt in range(30):
                try:
                    with urllib.request.urlopen(base + "/health", timeout=2) as response:
                        assert not ingress and response.read() == b"ok"
                    break
                except urllib.error.HTTPError as error:
                    if ingress and error.code == 403:
                        break
                    raise
                except (urllib.error.URLError, TimeoutError):
                    time.sleep(1)
            else:
                raise RuntimeError("Container not ready")
            if ingress:
                request = urllib.request.Request(base + "/api/state", headers={"X-Forwarded-For": "172.30.32.2"})
                try:
                    urllib.request.urlopen(request)
                    raise AssertionError("Ingress spoof accepted")
                except urllib.error.HTTPError as error:
                    assert error.code == 403
                continue
            def get(path):
                with urllib.request.urlopen(base + path) as response:
                    return json.load(response)
            state = get("/api/state")
            assert state["observe_only"] is True and state["control_enabled"] is False
            assert state["decision"]["last_balance"] == last_balance
            assert all(o["sent"] is None for o in state["outputs"].values())
            history = get("/api/history")
            if previous_at is not None:
                assert any(row["at"] == previous_at for row in history), "Lost /data across container replacement"
            previous_at = history[-1]["at"]
            with urllib.request.urlopen(base + "/") as response:
                html = response.read().decode()
            assets = re.findall(r'(?:src|href)="(\./assets/[^"]+)"', html)
            assert len(assets) >= 2, "Missing relative Ingress assets"
            for icon in ("favicon.png", "app-icon.png", "apple-touch-icon.png"):
                assert f'href="./{icon}"' in html, f"Missing relative icon: {icon}"
                with urllib.request.urlopen(base + "/" + icon) as response:
                    assert response.headers.get_content_type() == "image/png"
                    assert response.read(8) == b"\x89PNG\r\n\x1a\n"
            for asset in assets:
                with urllib.request.urlopen(base + "/" + asset) as response:
                    assert response.status == 200
        finally:
            docker("stop", "-t", "2", container)
            docker("rm", container)
print("Container API, persistent settings/history, assets and Ingress ACL passed")

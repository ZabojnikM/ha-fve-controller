"""Exercise real Git promotion in an isolated local bare repository."""
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import unittest

PROMOTE = pathlib.Path(__file__).with_name("promote.sh").resolve()


class ReleaseTest(unittest.TestCase):
    def test_first_update_and_safe_retry(self):
        bash = shutil.which("bash")
        if os.name == "nt":
            bash = "C:/Program Files/Git/bin/bash.exe"
        with tempfile.TemporaryDirectory() as temporary:
            root = pathlib.Path(temporary)
            remote = root / "remote.git"
            source = root / "source"
            def run(*args, cwd=source, env=None):
                try:
                    return subprocess.check_output(args, cwd=cwd, env=env, stderr=subprocess.STDOUT, text=True).strip()
                except subprocess.CalledProcessError as error:
                    print(error.output)
                    raise
            run("git", "init", "--bare", str(remote), cwd=root)
            run("git", "init", "-b", "main", str(source), cwd=root)
            run("git", "config", "user.name", "Release test")
            run("git", "config", "user.email", "test@example.invalid")
            run("git", "remote", "add", "origin", str(remote))
            (source / "fve_controller").mkdir()
            previous = None
            for version in ("0.1.0", "0.1.1"):
                (source / "fve_controller/config.yaml").write_text(f'version: "{version}"\n')
                (source / "fve_controller/CHANGELOG.md").write_text(f"## {version}\n")
                run("git", "add", ".")
                run("git", "commit", "-m", version)
                run("git", "push", "origin", "main")
                sha = run("git", "rev-parse", "HEAD")
                candidate = root / ("runner-" + version)
                run("git", "clone", "--branch", "main", str(remote), str(candidate), cwd=root)
                # Fake only the GitHub Release API; all Git operations are real.
                fakebin = candidate / "fake-bin"
                fakebin.mkdir()
                gh = fakebin / "gh"
                gh.write_text('#!/bin/sh\nif [ "$2" = "view" ]; then exit 1; fi\nexit 0\n', newline="\n")
                gh.chmod(0o755)
                env = dict(os.environ, VERSION=version, GITHUB_SHA=sha, PYTHON=pathlib.Path(sys.executable).as_posix())
                env["PATH"] = str(fakebin) + os.pathsep + env["PATH"]
                for _ in range(2):
                    run(bash, PROMOTE.as_posix(), cwd=candidate, env=env)
                stable = run("git", "rev-parse", "refs/heads/stable", cwd=remote)
                self.assertEqual(run("git", "rev-parse", "refs/tags/v" + version, cwd=remote), stable)
                self.assertEqual(run("git", "rev-parse", stable + "^{tree}", cwd=remote), run("git", "rev-parse", sha + "^{tree}", cwd=remote))
                if previous:
                    run("git", "merge-base", "--is-ancestor", previous, stable, cwd=remote)
                previous = stable
                # A stale workflow must not change a published ref.
                env["GITHUB_SHA"] = "0" * 40
                with self.assertRaises(subprocess.CalledProcessError):
                    run(bash, PROMOTE.as_posix(), cwd=candidate, env=env)
                self.assertEqual(run("git", "rev-parse", "refs/heads/stable", cwd=remote), stable)


if __name__ == "__main__":
    unittest.main()

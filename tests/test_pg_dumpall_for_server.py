"""Exercise the real selector with isolated PostgreSQL client stand-ins."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class PgDumpallForServerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        libexec = root / "libexec"
        source = Path(__file__).resolve().parents[1] / "scripts/pg_dumpall-for-server"
        # Relocate only the absolute client directory; execute the selector unchanged.
        self.script = root / "pg_dumpall-for-server"
        self.script.write_text(source.read_text().replace("/usr/libexec/", f"{libexec}/"))
        self.env = dict(os.environ, CALLS=str(root / "calls.json"))
        self.calls = root / "calls.json"
        for major in (16, 17, 18):
            directory = libexec / f"postgresql{major}"
            directory.mkdir(parents=True)
            for command in ("psql", "pg_dumpall"):
                client = directory / command
                client.write_text(
                    "#!/usr/bin/env python3\n"
                    "import json, os, sys\n"
                    "with open(os.environ['CALLS'], 'a') as calls:\n"
                    "    calls.write(json.dumps(sys.argv) + '\\n')\n"
                    "if sys.argv[0].endswith('/psql'):\n"
                    "    print(os.environ['SERVER_VERSION'])\n"
                    "    sys.exit(int(os.environ.get('PSQL_EXIT', '0')))\n"
                    "sys.exit(int(os.environ.get('DUMP_EXIT', '0')))\n"
                )
                client.chmod(0o755)

    def run_selector(self, version, *args, **env):
        return subprocess.run(
            ["sh", str(self.script), *args],
            env=dict(self.env, SERVER_VERSION=version, **env),
            capture_output=True, text=True,
        )

    def recorded_calls(self):
        return [json.loads(line) for line in self.calls.read_text().splitlines()]

    def test_postgresql_15_uses_16_and_preserves_dump_arguments(self):
        args = ["--no-password", "--clean", "--if-exists", "--host", "matrix-db",
                "--port=5433", "--username", "backup user"]
        result = self.run_selector("150004", *args)
        self.assertEqual(result.returncode, 0, result.stderr)
        probe, dump = self.recorded_calls()
        self.assertTrue(dump[0].endswith("/postgresql16/pg_dumpall"))
        self.assertEqual(dump[1:], args)
        self.assertEqual(probe[1:], ["--no-password", "--tuples-only", "--no-align",
                                   "--dbname=postgres", "--host=matrix-db", "--port=5433",
                                   "--username=backup user", "--command=SHOW server_version_num"])

    def test_existing_majors_keep_matching_clients(self):
        for major in (16, 17, 18):
            with self.subTest(major=major):
                result = self.run_selector(f"{major}0001", "--host=db", "--username=postgres")
                self.assertEqual(result.returncode, 0, result.stderr)
                dump = self.recorded_calls()[-1]
                self.assertTrue(dump[0].endswith(f"/postgresql{major}/pg_dumpall"))

    def test_unsupported_versions_do_not_start_a_dump(self):
        for version in ("140001", "190001", "invalid"):
            with self.subTest(version=version):
                result = self.run_selector(version)
                self.assertEqual(result.returncode, 1)
                self.assertIn(f"unsupported PostgreSQL server version: {version}", result.stderr)
        self.assertTrue(all(call[0].endswith("/psql") for call in self.recorded_calls()))

    def test_probe_failure_does_not_start_a_dump(self):
        result = self.run_selector("150004", PSQL_EXIT="2")
        self.assertEqual(result.returncode, 2)
        self.assertEqual(len(self.recorded_calls()), 1)

    def test_dump_failure_is_propagated(self):
        result = self.run_selector("150004", DUMP_EXIT="7")
        self.assertEqual(result.returncode, 7)


if __name__ == "__main__":
    unittest.main()

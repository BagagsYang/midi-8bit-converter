import hashlib
import json
from pathlib import Path
import unittest


REPO_ROOT = Path(__file__).resolve().parents[3]
FIXTURE_ROOT = REPO_ROOT / "backend" / "testdata" / "python-baseline"


class PythonParityFixtureTests(unittest.TestCase):
    def test_manifest_lists_existing_artifacts_with_current_hashes(self):
        manifest_path = FIXTURE_ROOT / "manifest.json"
        self.assertTrue(manifest_path.exists(), "Python baseline manifest is missing.")

        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        self.assertEqual("octabit.python_baseline_fixtures.v1", manifest["schema"])
        self.assertTrue(manifest["python_runtime_only"])

        artifact_paths = dict(manifest["artifacts"])
        artifact_paths.update({
            metadata["path"]: metadata["sha256"]
            for metadata in manifest["midi"].values()
        })
        for relative_path, expected_sha256 in artifact_paths.items():
            with self.subTest(relative_path=relative_path):
                path = FIXTURE_ROOT / relative_path
                self.assertTrue(path.exists(), f"{relative_path} is missing.")
                actual_sha256 = hashlib.sha256(path.read_bytes()).hexdigest()
                self.assertEqual(expected_sha256, actual_sha256)

    def test_api_fixtures_cover_stable_and_legacy_route_surfaces(self):
        implemented_routes = json.loads(
            (FIXTURE_ROOT / "api" / "implemented_routes.json").read_text(encoding="utf-8")
        )
        workspace_flow = json.loads(
            (FIXTURE_ROOT / "api" / "workspace_flow.json").read_text(encoding="utf-8")
        )
        error_responses = json.loads(
            (FIXTURE_ROOT / "api" / "error_responses.json").read_text(encoding="utf-8")
        )
        legacy_jobs = json.loads(
            (FIXTURE_ROOT / "api" / "legacy_jobs.json").read_text(encoding="utf-8")
        )

        self.assertIn("api_health", {
            record["label"] for record in implemented_routes["records"]
        })
        self.assertIn("preview_pulse_50", {
            record["label"] for record in implemented_routes["records"]
        })
        self.assertIn("create_synthesis_job", {
            record["label"] for record in workspace_flow["records"]
        })
        self.assertIn("render_queue_full", {
            record["label"] for record in error_responses["records"]
        })
        self.assertIn("download_legacy_job", {
            record["label"] for record in legacy_jobs["records"]
        })

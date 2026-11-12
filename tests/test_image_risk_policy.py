import json
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path


class ImageRiskPolicyTests(unittest.TestCase):
    def check(self, *, review_offset=1, architecture="arm64", installed="1.0"):
        image = "example.invalid/example:1"
        today = datetime.now(timezone.utc).date()
        finding = {
            "VulnerabilityID": "CVE-TEST",
            "PkgName": "example",
            "InstalledVersion": installed,
            "Severity": "HIGH",
        }
        risks = {
            "reviewDate": str(today + timedelta(days=review_offset)),
            "reason": "No fix in the official chart reviewed.",
            "images": {image: {"findings": [{
                "advisory": "CVE-TEST",
                "package": "example",
                "installedVersion": "1.0",
                "severity": "HIGH",
                "architectures": ["arm64"],
            }]}},
        }
        with tempfile.TemporaryDirectory() as directory:
            report = Path(directory) / "report.json"
            report.write_text(json.dumps({"Results": [{"Vulnerabilities": [finding]}]}))
            policy = Path(directory) / "risks.json"
            policy.write_text(json.dumps(risks))
            reporter = Path(__file__).with_name("report-image-findings.py")
            return subprocess.run(
                [sys.executable, str(reporter), str(report), str(policy), image, architecture],
                capture_output=True, text=True, check=False,
            )

    def test_current_exact_match_is_reported(self):
        result = self.check()
        self.assertEqual(result.returncode, 0)
        self.assertIn("accepted through", result.stdout)

    def test_expired_entry_blocks(self):
        result = self.check(review_offset=-1)
        self.assertEqual(result.returncode, 1)
        self.assertIn("review expired", result.stdout)

    def test_unlisted_architecture_or_installed_version_blocks(self):
        for result in (self.check(architecture="amd64"), self.check(installed="2.0")):
            self.assertEqual(result.returncode, 1)
            self.assertIn("unaccepted", result.stdout)


if __name__ == "__main__":
    unittest.main()

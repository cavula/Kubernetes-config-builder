"""Report chart-image findings and fail on unaccepted or expired entries."""

import json
import sys
from datetime import datetime, timezone
from pathlib import Path


def report_findings(report_path, risks_path, image, architecture):
    report = json.loads(report_path.read_text())
    risks = json.loads(risks_path.read_text())
    review_date = datetime.fromisoformat(risks["reviewDate"]).date()
    review_current = datetime.now(timezone.utc).date() <= review_date
    accepted = {
        (risk["advisory"], risk["package"], risk["installedVersion"], risk["severity"])
        for risk in risks["images"].get(image, {}).get("findings", [])
        if architecture in risk["architectures"]
    }
    blocked = False
    reported = set()
    for target in report.get("Results") or []:
        for issue in target.get("Vulnerabilities") or []:
            key = tuple(issue[field] for field in ("VulnerabilityID", "PkgName", "InstalledVersion", "Severity"))
            if key in reported:
                continue
            reported.add(key)
            if key in accepted and review_current:
                disposition = f"accepted through {review_date}: {risks['reason']}"
            else:
                disposition = "review expired" if key in accepted else "unaccepted"
                blocked = True
            fixed = issue.get("FixedVersion") or "none listed"
            print(f"{key[3]} {key[0]} {key[1]} {key[2]} (upstream fix: {fixed}) — {disposition}")
    return int(blocked)


if __name__ == "__main__":
    report, risks, image, architecture = sys.argv[1:]
    sys.exit(report_findings(Path(report), Path(risks), image, architecture))

# Prebuilt Docker releases

Publish a public GHCR image for each stable release on Linux AMD64 and ARM64. Build the release source with the existing verified Dockerfile, including patched promtool and chart and schema checksum checks. Verify both platform builds and publish a release-tagged manifest.

Testers use the release Compose file and `docker compose up -d` with a fresh image cache and no registry credentials. Measure elapsed time from invoking Compose to a successful dashboard HTTP response. Developers can still build locally with an explicit Compose override.

Backfill the already reviewed v0.3.0 release from its existing tag and attach the pull-based Compose file to the release.

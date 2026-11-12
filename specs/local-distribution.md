# Local Docker distribution

## Inputs

The v0.1.0 source release and a running Docker installation with Compose.

## Outputs

A local dashboard with the built frontend, Go API, validation binaries, pinned charts, and pinned Kubernetes schemas in one image.

## Acceptance criteria

- [x] A user can start the dashboard from the extracted release with one Docker Compose command.
- [x] The dashboard listens on localhost port 8080 and needs no Kubernetes credentials or developer tools.
- [x] The image build verifies the chart archives and bundled schemas against committed SHA-256 checksums.
- [x] The packaged dashboard serves the UI, renders both charts, and validates both tools across the three supported Kubernetes versions.

FROM node:26.13.1-bookworm-slim@sha256:5df79989ed685e09f0a98019db4cebfb22fb9df1a53c0161664a011f7ff6b0f1 AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json frontend/.npmrc ./
RUN npm ci
COPY frontend/ ./
COPY templates/ /src/templates/
COPY security/accepted-risks.json /src/security/
COPY backend/prometheus-arm64-values.json /src/backend/
RUN npm run build

FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS backend
WORKDIR /src/backend
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -trimpath -o /out/kcb . \
    && GOBIN=/out go install github.com/yannh/kubeconform/cmd/kubeconform@v0.7.0

FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS assets
ARG TARGETARCH
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*
RUN set -eu; \
    archive="helm-v4.3.1-linux-${TARGETARCH}.tar.gz"; \
    curl -fsSLO "https://get.helm.sh/$archive"; \
    curl -fsSLO "https://get.helm.sh/$archive.sha256sum"; \
    sha256sum --check "$archive.sha256sum"; \
    tar -xzf "$archive" "linux-${TARGETARCH}/helm"; \
    install "linux-${TARGETARCH}/helm" /usr/local/bin/helm
COPY backend/download-charts.sh backend/chart-checksums.txt backend/download-schemas.py backend/schema-checksums.txt backend/
RUN ./backend/download-charts.sh && python3 backend/download-schemas.py \
    && cd .runtime/schemas && sha256sum --check --strict /src/backend/schema-checksums.txt

FROM scratch
COPY --from=backend /out/ /usr/local/bin/
COPY --from=assets /usr/local/bin/helm /usr/local/bin/helm
COPY --from=assets /src/.runtime/ /app/assets/
COPY --from=frontend /src/frontend/dist/ /app/web/
USER 65532:65532
ENV HOME=/tmp PATH=/usr/local/bin
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/kcb", "-listen", "0.0.0.0:8080", "-web", "/app/web", "-charts", "/app/assets/charts", "-schemas", "/app/assets/schemas"]

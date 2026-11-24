# Builds the two container images. Run from the repository root:
#
#   docker build --target web    -t <image> .   # the site: API + React app
#   docker build --target ingest -t <image> .   # the private ingest service
#
# Multi-stage build: the early stages hold the full Node and Go toolchains and
# do the compiling; the final images copy in only the finished files. They
# have no shell, package manager, or compiler, which makes them small and
# leaves little for an attacker to use.

# ---- Stage 1: build the React app into static files ----
FROM node:24-slim AS frontend
WORKDIR /src
# Install dependencies in their own layer, so they are cached and only
# reinstalled when package.json or package-lock.json change.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: compile both Go programs ----
FROM golang:1.27 AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# CGO_ENABLED=0 builds fully static binaries that need no C libraries.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ingest ./cmd/ingest

# ---- Final image: the site ----
# "distroless" images hold only what a static program needs to run. The
# nonroot variant runs the program as an unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot AS web
COPY --from=backend /out/api /server
COPY --from=frontend /src/dist /static
ENV STATIC_DIR=/static
ENTRYPOINT ["/server"]

# ---- Final image: the ingest service ----
FROM gcr.io/distroless/static-debian12:nonroot AS ingest
COPY --from=backend /out/ingest /server
ENTRYPOINT ["/server"]

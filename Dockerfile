# Build the manager binary
FROM docker.io/library/golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=0.0.0

WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends jq && rm -rf /var/lib/apt/lists/*
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# Disable CGO to avoid building C dependencies (NVIDIA nvml, etc.) that aren't needed
RUN CGO_ENABLED=0 go mod download

# Install swag CLI for generating Swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY cmd/component.json cmd/component.json
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/
COPY proto/ proto/

# Stamp the exact image tag into the JSON consumed by go:embed.
RUN jq --arg version "${VERSION}" '.version = $version' cmd/component.json > cmd/component.stamped.json && \
    mv cmd/component.stamped.json cmd/component.json

# Generate Swagger documentation (BEFORE building binary)
# This ensures docs are always up-to-date in every built image
RUN swag init -g internal/api/server.go --parseDependency --parseInternal -o internal/api/docs

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/base:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY config/crd/bases/ /crds/
USER 65532:65532

ENTRYPOINT ["/manager"]

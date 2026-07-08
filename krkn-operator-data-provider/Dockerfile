# oc build
FROM golang:1.26.2 AS oc-build
RUN apt-get update && apt-get install -y --no-install-recommends libkrb5-dev
WORKDIR /tmp
# oc build
RUN git clone --branch release-4.18 https://github.com/openshift/oc.git
WORKDIR /tmp/oc
RUN go mod edit -go 1.26.2 &&\
    go mod edit -require github.com/moby/buildkit@v0.12.5 &&\
    go mod edit -require github.com/containerd/containerd@v1.7.29&&\
    go mod edit -require github.com/docker/docker@v27.5.1+incompatible&&\
    go mod edit -require github.com/opencontainers/runc@v1.2.8&&\
    go mod edit -require github.com/go-git/go-git/v5@v5.13.0&&\
    go mod edit -require github.com/opencontainers/selinux@v1.13.0&&\
    go mod edit -require github.com/ulikunitz/xz@v0.5.15&&\
    go mod edit -require golang.org/x/net@v0.38.0&&\
    go mod edit -require github.com/containerd/containerd@v1.7.27&&\
    go mod edit -require golang.org/x/oauth2@v0.27.0&&\
    go mod edit -require golang.org/x/crypto@v0.35.0&&\
    go mod edit -replace github.com/containerd/containerd@v1.7.27=github.com/containerd/containerd@v1.7.29&&\
    go mod tidy && go mod vendor

RUN make GO_REQUIRED_MIN_VERSION:= oc

# Use Python 3.11 slim image
FROM docker.io/library/python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    git \
    kubectl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install dependencies globally
COPY --from=oc-build /tmp/oc/oc /usr/bin/oc
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY server.py .
COPY generated/ ./generated/

# Create non-root user for security
RUN useradd -r -u 1001 -m -s /sbin/nologin nonroot && \
    chown -R nonroot:nonroot /app

# Set environment variables
ENV PYTHONUNBUFFERED=1

# Switch to non-root user
USER 1001

# Expose gRPC port
EXPOSE 50051

# Run the server
CMD ["python", "server.py"]

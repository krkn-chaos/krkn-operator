# NVIDIA NVML Stub

This stub implementation is **copied from krknctl** to enable building without CGO or NVIDIA libraries.

## Source

- **Upstream**: https://github.com/krkn-chaos/krknctl/tree/main/hack/stub-nvml
- **Documentation**: https://github.com/krkn-chaos/krknctl/blob/main/hack/README.md

## Why This Exists

The krkn-operator imports `krknctl` which has a transitive dependency on `github.com/NVIDIA/go-nvml`.
While krknctl uses NVML for GPU detection to select optimized container images, the operator itself
doesn't need GPU detection functionality.

This stub allows the operator to build with `CGO_ENABLED=0` in Docker without NVIDIA libraries.

## Maintenance

**Do not modify this stub directly.** It should be kept in sync with krknctl's version.

To update:
1. Check krknctl releases: https://github.com/krkn-chaos/krknctl/releases
2. Copy the latest `hack/stub-nvml/` from krknctl
3. Verify the operator builds: `go build ./...`

## Alternative

Instead of maintaining a local copy, you could use a `replace` directive pointing to krknctl's stub
directly, but that requires krknctl to be checked out locally during builds.

The local copy approach is preferred for:
- ✅ Reproducible builds (pinned to specific stub version)
- ✅ No dependency on external repository structure
- ✅ Works in air-gapped environments
# Release Context

This file owns context for installer, packaging, release artifacts, and
distribution.

## Current Status

Release installer work has not started.

## Not Implemented Yet

- `install.sh`
- exact version installation
- OS/architecture detection
- checksum verification
- GitHub Release artifact layout
- optional signature verification
- self-update design

## Roadmap

Start release installer work after the binary has meaningful node behavior to
ship and the secure connection foundation is reliable enough for early users.

Expected installer goals:

- Install a specific Tailedbox version.
- Detect OS and architecture.
- Download release artifacts from GitHub Releases.
- Verify checksums before installation.
- Install the `tailedbox` binary in a predictable path.
- Provide clear dry-run or preview behavior where possible.

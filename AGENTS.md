# Relayward Xray Plugin AGENTS.md

## Project Role

This repository owns the official Xray runtime plugin for Relayward. It adapts the Relayward plugin contracts to an independently managed Xray process on a node.

The Relayward control plane, Agent runtime, shared SDK contracts, risk analysis, and protocol-specific product configuration do not belong in this repository.

## Architecture Boundaries

- Implement both the center and node plugin entry points required by the Relayward SDK.
- Keep Xray-specific configuration and lifecycle behavior inside this repository.
- Use official `XTLS/Xray-core` release artifacts by default. Do not build or depend on a maintained Xray fork.
- Treat the plugin configuration as opaque to Relayward. Do not introduce protocol, certificate, inbound, or outbound product models until their requirements are explicitly approved.
- Do not add sing-box support here. A different runtime must use a separate plugin repository.

## Security

- Treat configuration, manifests, release metadata, archives, and RPC requests as untrusted input.
- Never log raw Xray configuration, credentials, private keys, subscription data, or access logs.
- Verify the size and SHA-256 digest of downloaded artifacts before installation.
- Reject unsafe archive paths, links, unexpected executable locations, unknown configuration fields, and unsupported platforms.
- Keep installed versions immutable and runtime configuration private to the plugin data directory.
- Start Xray in its own process group and terminate the group when replacing Xray or shutting down the plugin.

## Engineering Conventions

- Target Linux AMD64 only unless the supported platform policy is changed explicitly.
- Use the versioned contracts and helpers from `relayward-sdk`; do not duplicate shared protocol definitions.
- Keep release resolution, installation, configuration validation, and process supervision independently testable.
- Surface failures explicitly. Do not report a healthy generation until the matching Xray process is running.
- Preserve the last healthy process and configuration when candidate validation or startup fails.

## Git And Releases

- Keep commits, tags, releases, and release notes within this repository.
- Do not push, tag, or publish a release without explicit confirmation.
- Release artifacts must include the plugin manifest and Linux AMD64 center and node binaries.
- Do not commit downloaded Xray archives, Xray binaries, generated release bundles, local state, or credentials.

## Validation

Run the checks relevant to each change:

- `go test ./...`
- `go vet ./...`
- `go build ./...`
- release script and manifest validation when packaging changes

Prefer deterministic fake-Xray lifecycle tests during development. Use the real official artifact only for focused integration validation.

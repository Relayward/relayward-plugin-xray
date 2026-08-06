# Relayward Xray Plugin

`relayward-plugin-xray` is the official Xray runtime plugin for Relayward. The center artifact participates in Relayward plugin lifecycle management, while the node artifact installs and supervises an official Xray release on each node.

## Current Scope

- Linux AMD64 center and node artifacts
- responsive Simplified Chinese and English administration page
- structured VLESS + REALITY + TCP Vision configuration
- official stable `XTLS/Xray-core` release resolution
- bounded download with exact size and SHA-256 verification
- private, immutable Xray version installations
- Xray-native configuration checks with `xray run -test`
- process replacement with restoration of the previous healthy configuration when candidate startup fails
- dynamic authorization enforcement through Xray's local Handler API
- cumulative per-authorization upload and download counters through Xray's local Stats API
- VLESS URI, Mihomo, and sing-box subscription contributions
- Relayward generation, digest, and health reporting

The current runtime supports one service per node: `vless-reality`. Routing, DNS, additional protocols and transports, certificates, access-event collection, and dynamic source-IP blocking are not implemented.

## Configuration

Relayward treats runtime-plugin configuration as opaque JSON. The Xray plugin owns the following structure:

```json
{
  "xray_version": "26.3.27",
  "api_port": 10085,
  "credential_seed": "base64url-encoded-32-byte-secret",
  "vless_reality": {
    "enabled": true,
    "service_id": "vless-reality",
    "display_name": "VLESS Reality",
    "listen": "0.0.0.0",
    "port": 443,
    "public_port": 443,
    "target": "www.microsoft.com:443",
    "server_names": ["www.microsoft.com"],
    "private_key": "base64url-encoded-X25519-private-key",
    "short_ids": ["0123456789abcdef"],
    "flow": "xtls-rprx-vision",
    "fingerprint": "chrome"
  }
}
```

The administration page generates secrets for a node's first configuration and preserves them during normal edits. Unknown fields, prerelease Xray versions, invalid ports, non-domain REALITY targets, malformed keys, and trailing JSON are rejected. Relayward stores the opaque configuration through its encrypted plugin-configuration path.

Each authorization receives a deterministic UUID derived as `HMAC-SHA256(credential_seed, authorization_id + NUL + service_id)`. The UUID is stable for one node configuration, differs between authorizations, and cannot be derived from public Relayward identifiers without the node secret. Subscription rendering repeats the derivation without creating or mutating state.

## Release Trust

The node plugin queries the fixed official `XTLS/Xray-core` GitHub Release endpoint and selects only `Xray-linux-64.zip`. It requires the release to be published and stable, verifies the asset URL, bounds its size, and checks the SHA-256 digest supplied by GitHub before extraction. Xray is downloaded directly by the node plugin and is not bundled in Relayward plugin releases. Runtime control uses a persistent loopback gRPC connection and does not import or build Xray-core as a Go dependency.

## Development

```sh
go test ./...
go vet ./...
go build ./...
./scripts/build-release.sh 0.0.0-dev /tmp/relayward-plugin-xray-release
```

Release builds contain `relayward-plugin.json`, separate center and node Linux AMD64 artifacts, the sandboxed UI archive, and `SHA256SUMS`.

## License

Relayward Xray Plugin is licensed under GPL-3.0. Xray-core is an independent project distributed under MPL-2.0 and is downloaded from its official releases.

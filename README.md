# Relayward Xray Plugin

`relayward-plugin-xray` is the official Xray runtime plugin for Relayward. The center artifact participates in Relayward plugin lifecycle management, while the node artifact installs and supervises an official Xray release on each node.

## Current Scope

- Linux AMD64 center and node artifacts
- responsive Simplified Chinese and English administration page
- multiple independent VLESS + REALITY + TCP Vision services per node
- official stable `XTLS/Xray-core` release resolution
- bounded download with exact size and SHA-256 verification
- private, immutable Xray version installations
- Xray-native configuration checks with `xray run -test`
- process replacement with restoration of the previous healthy configuration when candidate startup fails
- dynamic authorization enforcement through Xray's local Handler API
- cumulative per-authorization upload and download counters through Xray's local Stats API
- recent accepted activity from Xray's online-user Stats API with a persistent telemetry cursor
- per-authorization dynamic source-IP blocking through Xray's local Routing API
- ordered static domain-suffix, destination-CIDR, and sniffed-protocol routing to direct or blocked outbounds
- VLESS URI, Mihomo, and sing-box subscription contributions
- Relayward generation, digest, and health reporting

The current runtime supports up to 64 independently identified VLESS + REALITY + TCP Vision services and 128 static routing rules per node. DNS configuration, additional outbound types, additional protocols and transports, certificates, and full access-log collection are not implemented.

Recent activity is an online-presence signal rather than a full request log. While an authorization remains online, the plugin emits at most one accepted activity event per authorization, service, and source IP every 30 seconds. The stream ID, sequence cursor, unacknowledged events, and refresh index are stored atomically in a private state file so Agent retries and plugin restarts do not create sequence gaps. Dynamic blocks match authorization email, inbound service, and one source IP together, avoiding collateral blocking of another authorization behind the same NAT. Runtime routing replacement always rebuilds the complete managed rule set in API, dynamic-block, then static-rule order, so a static direct rule cannot bypass a Relayward soft IP block.

## Configuration

Relayward treats runtime-plugin configuration as opaque JSON. The Xray plugin owns the following structure:

```json
{
  "xray_version": "26.3.27",
  "api_port": 10085,
  "credential_seed": "base64url-encoded-32-byte-secret",
  "services": [
    {
      "type": "vless-reality",
      "enabled": true,
      "service_id": "reality-main",
      "display_name": "Reality Main",
      "listen": "0.0.0.0",
      "port": 443,
      "public_port": 443,
      "vless_reality": {
        "target": "addons.mozilla.org:443",
        "server_names": ["addons.mozilla.org"],
        "private_key": "base64url-encoded-X25519-private-key",
        "short_ids": ["0123456789abcdef"],
        "flow": "xtls-rprx-vision",
        "fingerprint": "chrome"
      }
    }
  ],
  "routing": {
    "rules": [
      {
        "rule_id": "block-private",
        "display_name": "Block private destinations",
        "enabled": true,
        "domains": [],
        "ip_cidrs": ["192.0.2.0/24"],
        "protocols": [],
        "action": "blocked"
      }
    ]
  }
}
```

Each service keeps listener identity and public endpoint fields at the common service level. Protocol-specific fields are stored in the matching typed configuration object, currently `vless_reality`. The service-type catalog declares runtime and subscription capabilities, and conformance tests require every registered type to implement each declared layer before it can be added.

Each service ID is unique within its node configuration and becomes the Xray inbound tag used by authorization control, telemetry, dynamic blocking, and subscription rendering. Services are stored in service-ID order. The administration page generates a node credential seed and independent REALITY secrets for each new service. Editing a service preserves its secrets by service ID; deleting a service removes them. The retired single-service shape and the former flat multi-service REALITY fields are intentionally unsupported.

Static routing rules retain their configured order and have stable rule IDs. Values within one match category are alternatives, while every populated category on a rule must match. A domain value matches that domain and its subdomains; raw Xray expressions and regular expressions are not accepted. IP matches must use canonical IPv4 or IPv6 CIDR notation. Protocol matches are limited to `http`, `tls`, `quic`, and `bittorrent`; Xray reports HTTP/1 traffic as `http1`, which is covered by its `http` protocol-prefix matcher. Rules may send matching traffic only to the built-in `direct` or `blocked` outbound. Domain or protocol rules enable route-only HTTP, TLS, and QUIC sniffing on enabled service inbounds, preserving the original connection target while making the sniffed destination available to routing.

Unknown fields, prerelease Xray versions, duplicate service or rule IDs, conflicting listeners, non-domain REALITY targets, noncanonical CIDRs, unsupported routing expressions, malformed keys, and trailing JSON are rejected. Relayward stores the opaque configuration through its encrypted plugin-configuration path.

The target is a starting value, not a universal deployment choice. It must be reachable from the node, support TLS 1.3, and complete a real REALITY handshake with the selected Xray release; a successful TCP or ordinary TLS probe alone is insufficient.

Each authorization receives a deterministic UUID derived as `HMAC-SHA256(credential_seed, authorization_id + NUL + service_id)`. The UUID is stable for one node configuration, differs between authorizations, and cannot be derived from public Relayward identifiers without the node secret. Subscription rendering repeats the derivation without creating or mutating state.

## Release Trust

The node plugin queries the fixed official `XTLS/Xray-core` GitHub Release endpoint and selects only `Xray-linux-64.zip`. It requires the release to be published and stable, verifies the asset URL, bounds its size, and checks the SHA-256 digest supplied by GitHub before extraction. Xray is downloaded directly by the node plugin and is not bundled in Relayward plugin releases. Runtime control uses a persistent loopback gRPC connection and does not import or build Xray-core as a Go dependency. Routing API encoding supports both the legacy schema used through Xray `v26.7.10` and the geodata rule schema introduced in `v26.7.11`.

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

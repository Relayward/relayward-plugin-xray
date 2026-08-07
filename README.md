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
- ordered system, UDP, TCP, and DNS-over-HTTPS resolvers with global IPv4/IPv6 query strategy and per-server domain selection
- VLESS URI, Mihomo, and sing-box subscription contributions
- Relayward generation, digest, and health reporting

The current runtime supports up to 64 independently identified VLESS + REALITY + TCP Vision services, 128 static routing rules, and 16 ordered DNS servers per node. Additional outbound types, additional protocols and transports, certificates, and full access-log collection are not implemented.

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
  },
  "dns": {
    "enabled": true,
    "query_strategy": "use-ipv4",
    "servers": [
      {
        "server_id": "regional",
        "display_name": "Regional DNS",
        "enabled": true,
        "transport": "doh",
        "address": "https://dns.example.com/dns-query",
        "port": 0,
        "domains": ["example.com"]
      },
      {
        "server_id": "system",
        "display_name": "System DNS",
        "enabled": true,
        "transport": "system",
        "address": "",
        "port": 0,
        "domains": []
      }
    ]
  }
}
```

Each service keeps listener identity and public endpoint fields at the common service level. Protocol-specific fields are stored in the matching typed configuration object, currently `vless_reality`. The service-type catalog declares runtime and subscription capabilities, and conformance tests require every registered type to implement each declared layer before it can be added.

Each service ID is unique within its node configuration and becomes the Xray inbound tag used by authorization control, telemetry, dynamic blocking, and subscription rendering. Services are stored in service-ID order. The administration page generates a node credential seed and independent REALITY secrets for each new service. Editing a service preserves its secrets by service ID; deleting a service removes them. The retired single-service shape and the former flat multi-service REALITY fields are intentionally unsupported.

Static routing rules retain their configured order and have stable rule IDs. Values within one match category are alternatives, while every populated category on a rule must match. A domain value matches that domain and its subdomains; raw Xray expressions and regular expressions are not accepted. IP matches must use canonical IPv4 or IPv6 CIDR notation. Protocol matches are limited to `http`, `tls`, `quic`, and `bittorrent`; Xray reports HTTP/1 traffic as `http1`, which is covered by its `http` protocol-prefix matcher. Rules may send matching traffic only to the built-in `direct` or `blocked` outbound. Domain or protocol rules enable route-only HTTP, TLS, and QUIC sniffing on enabled service inbounds, preserving the original connection target while making the sniffed destination available to routing.

DNS is disabled unless explicitly enabled. Enabling it makes Xray use the configured resolver list for routing fallback and direct outbound domain resolution; disabling it preserves the previous `AsIs` direct-outbound behavior. The global query strategy is `use-ip`, `use-ipv4`, or `use-ipv6`. Servers retain their configured order and may use the system resolver, classic UDP, local TCP, or local DNS-over-HTTPS. UDP and TCP endpoints require a canonical IP address and explicit port. DNS-over-HTTPS endpoints require a credential-free HTTPS URL and are rendered in Xray local mode to avoid recursive bootstrap through the configured resolver chain.

An empty server domain list makes that server a general fallback resolver. A populated list contains lowercase domain suffixes and restricts the server to those domains and their subdomains. When at least one domain-specific server matches, general fallback servers are not queried for that lookup. Disabled servers remain editable in Relayward but are omitted from the generated Xray configuration.

Unknown fields, prerelease Xray versions, duplicate service, rule, or DNS server IDs, conflicting listeners, non-domain REALITY targets, noncanonical addresses or CIDRs, insecure DNS-over-HTTPS URLs, unsupported routing expressions, malformed keys, and trailing JSON are rejected. Relayward stores the opaque configuration through its encrypted plugin-configuration path.

The target is a starting value, not a universal deployment choice. It must be reachable from the node, support TLS 1.3, and complete a real REALITY handshake with the selected Xray release; a successful TCP or ordinary TLS probe alone is insufficient.

Each authorization receives a deterministic UUID derived as `HMAC-SHA256(credential_seed, authorization_id + NUL + service_id)`. The UUID is stable for one node configuration, differs between authorizations, and cannot be derived from public Relayward identifiers without the node secret. Subscription rendering repeats the derivation without creating or mutating state.

## Release Trust

The node plugin queries the fixed official `XTLS/Xray-core` GitHub Release endpoint and selects only `Xray-linux-64.zip`. It requires the release to be published and stable, verifies the asset URL, bounds its size, and checks the SHA-256 digest supplied by GitHub before extraction. Xray is downloaded directly by the node plugin and is not bundled in Relayward plugin releases. Runtime control uses a persistent loopback gRPC connection and does not import or build Xray-core as a Go dependency. Routing API encoding supports both the legacy schema used through Xray `v26.7.10` and the geodata rule schema introduced in `v26.7.11`.

## Development

```sh
go test ./...
go vet ./...
go build ./...

cd ui
npm ci
npm run typecheck
npm run lint
npm test
npm run build
cd ..

./scripts/build-release.sh 0.0.0-dev /tmp/relayward-plugin-xray-release
```

Run `npm run dev` from `ui/` for local UI development. The page is a sandboxed iframe application and communicates with Relayward through the vendored UI SDK; a host simulator is required for standalone browser interaction.

Release builds contain `relayward-plugin.json`, separate center and node Linux AMD64 artifacts, the sandboxed UI archive, and `SHA256SUMS`.

## License

Relayward Xray Plugin is licensed under GPL-3.0. Xray-core is an independent project distributed under MPL-2.0 and is downloaded from its official releases.

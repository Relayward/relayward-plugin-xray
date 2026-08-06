# Relayward Xray Plugin

`relayward-plugin-xray` is the official Xray runtime plugin for Relayward. The center artifact participates in Relayward plugin lifecycle management, while the node artifact installs and supervises an official Xray release on each node.

## First Release Scope

- Linux AMD64 center and node artifacts
- strict plugin configuration validation
- official stable `XTLS/Xray-core` release resolution
- bounded download with exact size and SHA-256 verification
- private, immutable Xray version installations
- Xray-native configuration checks with `xray run -test`
- process replacement with restoration of the previous healthy configuration when candidate startup fails
- Relayward generation, digest, and health reporting

The first release does not define protocol, inbound, outbound, certificate, subscription, telemetry, quota, or dynamic-blocking models. Those capabilities are not advertised to Relayward until they are implemented.

## Configuration

Relayward treats runtime-plugin configuration as opaque JSON. This plugin currently accepts exactly two top-level fields:

```json
{
  "xray_version": "26.3.27",
  "xray_config": {}
}
```

`xray_version` is an exact stable semantic version without a leading `v`. `xray_config` is passed to Xray as its complete configuration object. Unknown plugin-level fields, prerelease versions, trailing JSON, and non-object Xray configurations are rejected.

## Release Trust

The node plugin queries the fixed official `XTLS/Xray-core` GitHub Release endpoint and selects only `Xray-linux-64.zip`. It requires the release to be published and stable, verifies the asset URL, bounds its size, and checks the SHA-256 digest supplied by GitHub before extraction. Xray is downloaded directly by the node plugin and is not bundled in Relayward plugin releases.

## Development

```sh
go test ./...
go vet ./...
go build ./...
./scripts/build-release.sh 0.0.0-dev /tmp/relayward-plugin-xray-release
```

Release builds contain `relayward-plugin.json`, separate center and node Linux AMD64 artifacts, and `SHA256SUMS`.

## License

Relayward Xray Plugin is licensed under GPL-3.0. Xray-core is an independent project distributed under MPL-2.0 and is downloaded from its official releases.

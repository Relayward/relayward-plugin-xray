export type Locale = "zh-CN" | "en"
export type Theme = "light" | "dark"

export interface NodeSummary {
  id: string
  name: string
  connected: boolean
}

export interface ServiceType {
  id: string
  display_name: string
}

export interface VLESSReality {
  target: string
  server_name: string
  fingerprint: string
}

export interface ProxyService {
  type: string
  enabled: boolean
  service_id: string
  display_name: string
  listen: string
  port: number
  public_host: string
  public_port: number
  vless_reality: VLESSReality
}

export type RoutingAction = "blocked" | "direct"
export type RoutingProtocol = "http" | "tls" | "quic" | "bittorrent"

export interface RoutingRule {
  rule_id: string
  display_name: string
  enabled: boolean
  domains: string[]
  ip_cidrs: string[]
  protocols: RoutingProtocol[]
  action: RoutingAction
}

export type DNSTransport = "system" | "udp" | "tcp" | "doh"
export type DNSQueryStrategy = "use-ip" | "use-ipv4" | "use-ipv6"

export interface DNSServer {
  server_id: string
  display_name: string
  enabled: boolean
  transport: DNSTransport
  address: string
  port: number
  domains: string[]
}

export interface DNSConfiguration {
  enabled: boolean
  query_strategy: DNSQueryStrategy
  servers: DNSServer[]
}

export interface EditableConfiguration {
  xray_version: string
  api_port: number
  services: ProxyService[]
  routing: { rules: RoutingRule[] }
  dns: DNSConfiguration
}

export interface StoredConfiguration {
  exists: boolean
  node_id: string
  generation?: number
  version?: string
  sha256?: string
  configuration?: EditableConfiguration
}

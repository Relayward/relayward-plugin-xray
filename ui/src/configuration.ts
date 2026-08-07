import type {
  DNSConfiguration,
  DNSServer,
  EditableConfiguration,
  Locale,
  ProxyService,
  RoutingRule,
  ServiceType,
  StoredConfiguration,
} from "@/types"

export function cloneServices(values: ProxyService[]): ProxyService[] {
  return values.map((service) => ({
    ...service,
    vless_reality: { ...service.vless_reality },
  }))
}

export function cloneRoutingRules(values: RoutingRule[]): RoutingRule[] {
  return values.map((rule) => ({
    ...rule,
    domains: [...rule.domains],
    ip_cidrs: [...rule.ip_cidrs],
    protocols: [...rule.protocols],
  }))
}

export function cloneDNSConfiguration(value: DNSConfiguration): DNSConfiguration {
  return {
    enabled: value.enabled,
    query_strategy: value.query_strategy,
    servers: value.servers.map((server) => ({ ...server, domains: [...server.domains] })),
  }
}

export function defaultDNSConfiguration(locale: Locale): DNSConfiguration {
  return {
    enabled: false,
    query_strategy: "use-ip",
    servers: [{
      server_id: "system",
      display_name: locale === "zh-CN" ? "系统 DNS" : "System DNS",
      enabled: true,
      transport: "system",
      address: "",
      port: 0,
      domains: [],
    }],
  }
}

export function nextServiceDefaults(services: ProxyService[], serviceTypes: ServiceType[]): ProxyService {
  const serviceType = serviceTypes[0]?.id ?? "vless-reality"
  let suffix = services.length === 0 ? 1 : 2
  let serviceID = services.length === 0 ? "vless-reality" : `vless-reality-${suffix}`
  while (services.some((service) => service.service_id === serviceID)) {
    suffix += 1
    serviceID = `vless-reality-${suffix}`
  }
  let port = services.length === 0 ? 443 : 8443
  while (services.some((service) => service.port === port) && port < 65535) port += 1
  return {
    type: serviceType,
    enabled: true,
    service_id: serviceID,
    display_name: services.length === 0 ? "VLESS Reality" : `VLESS Reality ${services.length + 1}`,
    listen: "0.0.0.0",
    port,
    public_host: "edge.example.com",
    public_port: port,
    vless_reality: {
      target: "www.cloudflare.com:443",
      server_name: "www.cloudflare.com",
      fingerprint: "chrome",
    },
  }
}

export function nextRoutingRuleDefaults(rules: RoutingRule[], locale: Locale): RoutingRule {
  let suffix = rules.length + 1
  let ruleID = `routing-rule-${suffix}`
  while (rules.some((rule) => rule.rule_id === ruleID)) {
    suffix += 1
    ruleID = `routing-rule-${suffix}`
  }
  return {
    rule_id: ruleID,
    display_name: locale === "zh-CN" ? `路由规则 ${suffix}` : `Routing rule ${suffix}`,
    enabled: true,
    domains: [],
    ip_cidrs: [],
    protocols: [],
    action: "blocked",
  }
}

export function nextDNSServerDefaults(servers: DNSServer[], locale: Locale): DNSServer {
  let suffix = servers.length + 1
  let serverID = `dns-server-${suffix}`
  while (servers.some((server) => server.server_id === serverID)) {
    suffix += 1
    serverID = `dns-server-${suffix}`
  }
  return {
    server_id: serverID,
    display_name: locale === "zh-CN" ? `DNS 服务器 ${suffix}` : `DNS server ${suffix}`,
    enabled: true,
    transport: "udp",
    address: "1.1.1.1",
    port: 53,
    domains: [],
  }
}

export function configurationFromStored(
  stored: StoredConfiguration,
  serviceTypes: ServiceType[],
  locale: Locale,
): EditableConfiguration {
  if (!stored.exists || stored.configuration == null) {
    return {
      xray_version: "26.3.27",
      api_port: 10085,
      services: [nextServiceDefaults([], serviceTypes)],
      routing: { rules: [] },
      dns: defaultDNSConfiguration(locale),
    }
  }
  const value = stored.configuration
  const dns = value.dns?.query_strategy || value.dns?.servers?.length > 0
    ? cloneDNSConfiguration(value.dns)
    : defaultDNSConfiguration(locale)
  return {
    xray_version: value.xray_version,
    api_port: value.api_port,
    services: cloneServices(Array.isArray(value.services) ? value.services : []),
    routing: { rules: cloneRoutingRules(Array.isArray(value.routing?.rules) ? value.routing.rules : []) },
    dns,
  }
}

export function configurationForSave(value: EditableConfiguration): EditableConfiguration {
  return {
    xray_version: value.xray_version.trim(),
    api_port: value.api_port,
    services: cloneServices(value.services).sort((first, second) => first.service_id.localeCompare(second.service_id)),
    routing: { rules: cloneRoutingRules(value.routing.rules) },
    dns: cloneDNSConfiguration(value.dns),
  }
}

export function lines(value: string): string[] {
  return value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean)
}

export function moveItem<T>(values: T[], index: number, offset: number): T[] {
  const target = index + offset
  if (target < 0 || target >= values.length) return values
  const result = [...values]
  ;[result[index], result[target]] = [result[target], result[index]]
  return result
}

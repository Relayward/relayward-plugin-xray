export interface RelaywardUIContext {
  plugin_id: string
  theme: "light" | "dark"
  locale: "zh-CN" | "en"
}

export interface RelaywardUIClient {
  context(): Promise<RelaywardUIContext>
  rpc(method: string, parameters: Record<string, unknown>): Promise<unknown>
  navigate(target: "plugins" | "nodes" | "users" | "authorizations" | "audit"): Promise<void>
  confirm(options: {
    title: string
    message: string
    confirm_label: string
    destructive: boolean
  }): Promise<boolean>
  dispose(): void
}

export function browserUITransport(): unknown
export function createRelaywardUIClient(transport: unknown, timeoutMilliseconds?: number): RelaywardUIClient

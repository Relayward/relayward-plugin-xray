import {
  browserUITransport,
  createRelaywardUIClient,
  type RelaywardUIClient,
} from "../vendor/relayward-ui-sdk.js"

export function createClient(): RelaywardUIClient {
  return createRelaywardUIClient(browserUITransport())
}

export type { RelaywardUIClient }

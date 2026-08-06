import { browserUITransport, createRelaywardUIClient } from "./vendor/relayward-ui-sdk.js";

const client = createRelaywardUIClient(browserUITransport());
const elements = Object.fromEntries(Array.from(document.querySelectorAll("[id]")).flatMap((element) => {
  const camel = element.id.replace(/-([a-z])/g, (_, character) => character.toUpperCase());
  return [[element.id, element], [camel, element]];
}));

const messages = {
  "Node configuration": "节点配置",
  "Not selected": "未选择",
  "Node": "节点",
  "Refresh": "刷新",
  "Service": "服务",
  "Network": "网络",
  "Runtime": "运行时",
  "VLESS REALITY": "VLESS REALITY",
  "Service identity and client defaults": "服务标识与客户端默认值",
  "Enabled": "已启用",
  "Display name": "显示名称",
  "Xray version": "Xray 版本",
  "Client fingerprint": "客户端指纹",
  "Listener and REALITY": "监听与 REALITY",
  "Inbound ports and handshake target": "入站端口与握手目标",
  "Listen address": "监听地址",
  "Listen port": "监听端口",
  "Public port": "公网端口",
  "REALITY target": "REALITY 目标",
  "Server name": "服务器名称",
  "Local runtime": "本地运行时",
  "Private control endpoint": "私有控制端点",
  "Local API port": "本地 API 端口",
  "Transport": "传输组合",
  "Save configuration": "保存配置",
  "No nodes available": "暂无可用节点",
  "Online": "在线",
  "Offline": "离线",
  "Generation {generation}": "第 {generation} 代配置",
  "Not configured": "尚未配置",
  "Configuration saved.": "配置已保存。",
  "Loading...": "正在加载...",
  "Saving...": "正在保存...",
  "The request could not be completed.": "请求未能完成。",
};

let locale = "en";
let nodes = [];
let selectedNode;
let stored;
let busy = false;

function text(message, values = {}) {
  let result = locale === "zh-CN" ? (messages[message] ?? message) : message;
  for (const [key, value] of Object.entries(values)) result = result.replace(`{${key}}`, String(value));
  return result;
}

function translatePage() {
  document.documentElement.lang = locale;
  for (const element of document.querySelectorAll("[data-i18n]")) {
    element.textContent = text(element.dataset.i18n);
  }
  updateNodeStatus();
  updateGeneration();
}

function showError(cause) {
  elements.notice.hidden = true;
  elements.error.textContent = cause instanceof Error && cause.message ? cause.message : text("The request could not be completed.");
  elements.error.hidden = false;
}

function clearMessages() {
  elements.notice.hidden = true;
  elements.error.hidden = true;
}

function setBusy(value, label) {
  busy = value;
  elements.refreshButton.disabled = value;
  elements.nodeSelect.disabled = value || nodes.length === 0;
  elements.saveButton.disabled = value;
  elements.saveButton.textContent = value && label === "save" ? text("Saving...") : text("Save configuration");
}

function updateNodeStatus() {
  elements.nodeStatus.classList.toggle("connected", selectedNode?.connected === true);
  elements.nodeStatus.textContent = selectedNode == null ? text("Not selected") : text(selectedNode.connected ? "Online" : "Offline");
}

function updateGeneration() {
  elements.generation.textContent = stored?.exists ? text("Generation {generation}", { generation: stored.generation }) : text("Not configured");
}

function renderNodes() {
  const previous = elements.nodeSelect.value;
  elements.nodeSelect.replaceChildren();
  for (const node of nodes) {
    const option = document.createElement("option");
    option.value = node.id;
    option.textContent = node.name;
    elements.nodeSelect.append(option);
  }
  const next = nodes.some((node) => node.id === previous) ? previous : (nodes[0]?.id ?? "");
  elements.nodeSelect.value = next;
  selectedNode = nodes.find((node) => node.id === next);
  elements.emptyState.hidden = nodes.length !== 0;
  elements.configurationCard.hidden = nodes.length === 0;
  elements.nodeSelect.disabled = nodes.length === 0 || busy;
  updateNodeStatus();
}

function numberValue(id) {
  return Number(elements[id].value);
}

function defaults() {
  return {
    enabled: true,
    displayName: "VLESS Reality",
    xrayVersion: "26.3.27",
    fingerprint: "chrome",
    listen: "0.0.0.0",
    port: 443,
    publicPort: 443,
    target: "www.microsoft.com:443",
    serverName: "www.microsoft.com",
    apiPort: 10085,
  };
}

function populateForm() {
  const configuration = stored?.exists ? stored.configuration : undefined;
  const service = configuration?.vless_reality;
  const value = configuration == null ? defaults() : {
    enabled: service.enabled,
    displayName: service.display_name,
    xrayVersion: configuration.xray_version,
    fingerprint: service.fingerprint,
    listen: service.listen,
    port: service.port,
    publicPort: service.public_port,
    target: service.target,
    serverName: service.server_name,
    apiPort: configuration.api_port,
  };
  elements.enabled.checked = value.enabled;
  elements.displayName.value = value.displayName;
  elements.xrayVersion.value = value.xrayVersion;
  elements.fingerprint.value = value.fingerprint;
  elements.listenAddress.value = value.listen;
  elements.listenPort.value = String(value.port);
  elements.publicPort.value = String(value.publicPort);
  elements.realityTarget.value = value.target;
  elements.serverName.value = value.serverName;
  elements.apiPort.value = String(value.apiPort);
  updateGeneration();
}

async function loadConfiguration() {
  if (selectedNode == null) return;
  clearMessages();
  setBusy(true, "load");
  try {
    stored = await client.rpc("configuration.get", { node_id: selectedNode.id });
    populateForm();
  } catch (cause) {
    showError(cause);
  } finally {
    setBusy(false);
  }
}

async function refreshNodes() {
  clearMessages();
  setBusy(true, "load");
  try {
    const response = await client.rpc("nodes.list", {});
    nodes = Array.isArray(response.nodes) ? response.nodes : [];
    renderNodes();
    await loadConfiguration();
  } catch (cause) {
    showError(cause);
  } finally {
    setBusy(false);
  }
}

function configurationForSave() {
  return {
    xray_version: elements.xrayVersion.value.trim(),
    api_port: numberValue("api-port"),
    vless_reality: {
      enabled: elements.enabled.checked,
      display_name: elements.displayName.value.trim(),
      listen: elements.listenAddress.value.trim(),
      port: numberValue("listen-port"),
      public_port: numberValue("public-port"),
      target: elements.realityTarget.value.trim(),
      server_name: elements.serverName.value.trim(),
      fingerprint: elements.fingerprint.value,
    },
  };
}

async function saveConfiguration(event) {
  event.preventDefault();
  if (busy || selectedNode == null || !elements.configurationForm.reportValidity()) return;
  clearMessages();
  setBusy(true, "save");
  try {
    const configuration = configurationForSave();
    await client.rpc("configuration.save", {
      node_id: selectedNode.id,
      expected_generation: stored?.exists ? stored.generation : 0,
      configuration,
    });
    stored = await client.rpc("configuration.get", { node_id: selectedNode.id });
    populateForm();
    elements.notice.textContent = text("Configuration saved.");
    elements.notice.hidden = false;
  } catch (cause) {
    showError(cause);
  } finally {
    setBusy(false);
  }
}

function selectTab(button) {
  for (const tab of document.querySelectorAll("[role=tab]")) {
    const active = tab === button;
    tab.classList.toggle("active", active);
    tab.setAttribute("aria-selected", String(active));
    elements[tab.dataset.panel].hidden = !active;
  }
}

for (const tab of document.querySelectorAll("[role=tab]")) tab.addEventListener("click", () => selectTab(tab));
elements.nodeSelect.addEventListener("change", () => {
  selectedNode = nodes.find((node) => node.id === elements.nodeSelect.value);
  updateNodeStatus();
  void loadConfiguration();
});
elements.refreshButton.addEventListener("click", () => void refreshNodes());
elements.configurationForm.addEventListener("submit", (event) => void saveConfiguration(event));
window.addEventListener("pagehide", () => client.dispose(), { once: true });

try {
  const context = await client.context();
  locale = context.locale;
  document.documentElement.dataset.theme = context.theme;
  translatePage();
  await refreshNodes();
} catch (cause) {
  showError(cause);
}

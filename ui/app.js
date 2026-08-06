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
  "Services": "服务",
  "Runtime": "运行时",
  "Proxy services": "代理服务",
  "Independent listeners published by this node": "此节点发布的独立监听服务",
  "Add service": "新增服务",
  "No services configured": "尚未配置服务",
  "Add a service to publish an Xray inbound": "新增服务后即可发布 Xray 入站",
  "Local runtime": "本地运行时",
  "Xray release and private control endpoint": "Xray 版本与私有控制端点",
  "Xray version": "Xray 版本",
  "Local API port": "本地 API 端口",
  "Supported transport": "支持的传输组合",
  "Save configuration": "保存配置",
  "No nodes available": "暂无可用节点",
  "Configure one independent VLESS REALITY listener": "配置一个独立的 VLESS REALITY 监听服务",
  "Identity": "标识",
  "Enabled": "已启用",
  "Service ID": "服务 ID",
  "Service ID already exists.": "服务 ID 已存在。",
  "A node can contain at most 64 services.": "一个节点最多可配置 64 个服务。",
  "Display name": "显示名称",
  "Service type": "服务类型",
  "Client fingerprint": "客户端指纹",
  "Listener and REALITY": "监听与 REALITY",
  "Listen address": "监听地址",
  "Listen port": "监听端口",
  "Public port": "公网端口",
  "REALITY target": "REALITY 目标",
  "Server name": "服务器名称",
  "Cancel": "取消",
  "Apply service": "应用服务",
  "Add proxy service": "新增代理服务",
  "Edit proxy service": "编辑代理服务",
  "Edit": "编辑",
  "Delete": "删除",
  "Enabled status": "已启用",
  "Disabled status": "已停用",
  "Listen {address}:{port}": "监听 {address}:{port}",
  "Public port {port}": "公网端口 {port}",
  "Delete service": "删除服务",
  "Delete {name}? The change is published only after you save the configuration.": "删除 {name}？只有保存配置后，此更改才会发布。",
  "Service changes are ready. Save the configuration to publish them.": "服务更改已就绪，保存配置后发布。",
  "Online": "在线",
  "Offline": "离线",
  "Generation {generation}": "第 {generation} 代配置",
  "Not configured": "尚未配置",
  "Configuration saved.": "配置已保存。",
  "Loading...": "正在加载...",
  "Saving...": "正在保存...",
  "The request could not be completed.": "请求未能完成。",
  "Close": "关闭",
};

let locale = "en";
let nodes = [];
let selectedNode;
let stored;
let services = [];
let editingServiceID = null;
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
  elements.closeServiceDialog.title = text("Close");
  elements.closeServiceDialog.setAttribute("aria-label", text("Close"));
  updateNodeStatus();
  updateGeneration();
  renderServices();
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

function showPendingNotice() {
  elements.notice.textContent = text("Service changes are ready. Save the configuration to publish them.");
  elements.notice.hidden = false;
  elements.error.hidden = true;
}

function setBusy(value, label) {
  busy = value;
  elements.refreshButton.disabled = value;
  elements.nodeSelect.disabled = value || nodes.length === 0;
  elements.saveButton.disabled = value;
  elements.addServiceButton.disabled = value;
  elements.saveButton.textContent = value && label === "save" ? text("Saving...") : text("Save configuration");
  renderServices();
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

function nextServiceDefaults() {
  let suffix = services.length === 0 ? 1 : 2;
  let serviceID = services.length === 0 ? "vless-reality" : `vless-reality-${suffix}`;
  while (services.some((service) => service.service_id === serviceID)) {
    suffix += 1;
    serviceID = `vless-reality-${suffix}`;
  }
  let port = services.length === 0 ? 443 : 8443;
  while (services.some((service) => service.port === port) && port < 65535) port += 1;
  return {
    type: "vless-reality",
    enabled: true,
    service_id: serviceID,
    display_name: services.length === 0 ? "VLESS Reality" : `VLESS Reality ${services.length + 1}`,
    listen: "0.0.0.0",
    port,
    public_port: port,
    target: "www.cloudflare.com:443",
    server_name: "www.cloudflare.com",
    fingerprint: "chrome",
  };
}

function cloneServices(values) {
  return values.map((service) => ({ ...service }));
}

function populateConfiguration() {
  const configuration = stored?.exists ? stored.configuration : undefined;
  services = [];
  if (configuration == null) {
    elements.xrayVersion.value = "26.3.27";
    elements.apiPort.value = "10085";
    services = [nextServiceDefaults()];
  } else {
    elements.xrayVersion.value = configuration.xray_version;
    elements.apiPort.value = String(configuration.api_port);
    services = cloneServices(Array.isArray(configuration.services) ? configuration.services : []);
  }
  renderServices();
  updateGeneration();
}

function serviceStatus(service) {
  return service.enabled ? text("Enabled status") : text("Disabled status");
}

function serviceRow(service) {
  const row = document.createElement("article");
  row.className = "service-row";

  const identity = document.createElement("div");
  identity.className = "service-identity";
  const heading = document.createElement("div");
  heading.className = "service-name-line";
  const name = document.createElement("strong");
  name.textContent = service.display_name;
  const status = document.createElement("span");
  status.className = `badge service-status${service.enabled ? " connected" : ""}`;
  status.textContent = serviceStatus(service);
  heading.append(name, status);
  const identifier = document.createElement("span");
  identifier.className = "service-id";
  identifier.textContent = service.service_id;
  identity.append(heading, identifier);

  const network = document.createElement("div");
  network.className = "service-network";
  const listener = document.createElement("span");
  listener.textContent = text("Listen {address}:{port}", { address: service.listen, port: service.port });
  const publicPort = document.createElement("span");
  publicPort.textContent = text("Public port {port}", { port: service.public_port });
  network.append(listener, publicPort);

  const actions = document.createElement("div");
  actions.className = "service-actions";
  const edit = document.createElement("button");
  edit.className = "button button-outline button-compact";
  edit.type = "button";
  edit.disabled = busy;
  edit.textContent = text("Edit");
  edit.addEventListener("click", () => openServiceDialog(service.service_id));
  const remove = document.createElement("button");
  remove.className = "button button-ghost-destructive button-compact";
  remove.type = "button";
  remove.disabled = busy;
  remove.textContent = text("Delete");
  remove.addEventListener("click", () => void removeService(service.service_id));
  actions.append(edit, remove);

  row.append(identity, network, actions);
  return row;
}

function renderServices() {
  elements.serviceList.replaceChildren(...services.map(serviceRow));
  elements.serviceListEmpty.hidden = services.length !== 0;
}

async function loadConfiguration() {
  if (selectedNode == null) return;
  clearMessages();
  setBusy(true, "load");
  try {
    stored = await client.rpc("configuration.get", { node_id: selectedNode.id });
    populateConfiguration();
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
    services: cloneServices(services).sort((first, second) => first.service_id.localeCompare(second.service_id)),
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
    populateConfiguration();
    elements.notice.textContent = text("Configuration saved.");
    elements.notice.hidden = false;
  } catch (cause) {
    showError(cause);
  } finally {
    setBusy(false);
  }
}

function populateServiceDialog(service) {
  elements.serviceEnabled.checked = service.enabled;
  elements.serviceId.value = service.service_id;
  elements.serviceId.disabled = editingServiceID != null;
  elements.displayName.value = service.display_name;
  elements.fingerprint.value = service.fingerprint;
  elements.listenAddress.value = service.listen;
  elements.listenPort.value = String(service.port);
  elements.publicPort.value = String(service.public_port);
  elements.realityTarget.value = service.target;
  elements.serverName.value = service.server_name;
  elements.serviceId.setCustomValidity("");
}

function openServiceDialog(serviceID = null) {
  if (busy) return;
  if (serviceID == null && services.length >= 64) {
    showError(new Error(text("A node can contain at most 64 services.")));
    return;
  }
  editingServiceID = serviceID;
  const service = serviceID == null ? nextServiceDefaults() : services.find((candidate) => candidate.service_id === serviceID);
  if (service == null) return;
  elements.serviceDialogTitle.textContent = text(serviceID == null ? "Add proxy service" : "Edit proxy service");
  populateServiceDialog(service);
  elements.serviceDialog.showModal();
}

function closeServiceDialog() {
  editingServiceID = null;
  elements.serviceDialog.close();
}

function serviceFromDialog() {
  return {
    type: "vless-reality",
    enabled: elements.serviceEnabled.checked,
    service_id: elements.serviceId.value.trim(),
    display_name: elements.displayName.value.trim(),
    listen: elements.listenAddress.value.trim(),
    port: numberValue("listen-port"),
    public_port: numberValue("public-port"),
    target: elements.realityTarget.value.trim(),
    server_name: elements.serverName.value.trim(),
    fingerprint: elements.fingerprint.value,
  };
}

function applyService(event) {
  event.preventDefault();
  const candidate = serviceFromDialog();
  const duplicate = services.some((service) => service.service_id === candidate.service_id && service.service_id !== editingServiceID);
  elements.serviceId.setCustomValidity(duplicate ? text("Service ID already exists.") : "");
  if (!elements.serviceForm.reportValidity()) return;
  if (editingServiceID == null) {
    services.push(candidate);
  } else {
    const index = services.findIndex((service) => service.service_id === editingServiceID);
    if (index < 0) return;
    services[index] = candidate;
  }
  services.sort((first, second) => first.service_id.localeCompare(second.service_id));
  closeServiceDialog();
  renderServices();
  showPendingNotice();
}

async function removeService(serviceID) {
  if (busy) return;
  const service = services.find((candidate) => candidate.service_id === serviceID);
  if (service == null) return;
  let confirmed;
  try {
    confirmed = await client.confirm({
      title: text("Delete service"),
      message: text("Delete {name}? The change is published only after you save the configuration.", { name: service.display_name }),
      confirm_label: text("Delete"),
      destructive: true,
    });
  } catch (cause) {
    showError(cause);
    return;
  }
  if (!confirmed) return;
  services = services.filter((candidate) => candidate.service_id !== serviceID);
  renderServices();
  showPendingNotice();
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
elements.addServiceButton.addEventListener("click", () => openServiceDialog());
elements.closeServiceDialog.addEventListener("click", closeServiceDialog);
elements.cancelServiceButton.addEventListener("click", closeServiceDialog);
elements.serviceId.addEventListener("input", () => elements.serviceId.setCustomValidity(""));
elements.serviceForm.addEventListener("submit", applyService);
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

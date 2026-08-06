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
  "Routing": "路由",
  "Runtime": "运行时",
  "Proxy services": "代理服务",
  "Independent listeners published by this node": "此节点发布的独立监听服务",
  "Add service": "新增服务",
  "No services configured": "尚未配置服务",
  "Add a service to publish an Xray inbound": "新增服务后即可发布 Xray 入站",
  "Static routing rules": "静态路由规则",
  "Rules are evaluated from top to bottom after Relayward dynamic blocks": "Relayward 动态阻断之后，静态规则按从上到下的顺序匹配",
  "Add rule": "新增规则",
  "No static routing rules": "尚未配置静态路由规则",
  "Unmatched traffic uses the direct outbound": "未匹配的流量使用直连出站",
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
  "Configuration changes are ready. Save to publish them.": "配置更改已就绪，保存后发布。",
  "All populated match categories must match": "已填写的匹配分类必须同时满足",
  "Identity and action": "标识与动作",
  "Rule ID": "规则 ID",
  "Rule ID already exists.": "规则 ID 已存在。",
  "A node can contain at most 128 routing rules.": "一个节点最多可配置 128 条路由规则。",
  "Action": "动作",
  "Block": "阻断",
  "Direct": "直连",
  "Destination matches": "目标匹配",
  "Domain suffixes": "域名后缀",
  "One lowercase domain per line; subdomains are included": "每行一个小写域名，同时匹配其子域名",
  "IP CIDRs": "IP CIDR",
  "One canonical IPv4 or IPv6 CIDR per line": "每行一个规范的 IPv4 或 IPv6 CIDR",
  "Sniffed protocols": "嗅探协议",
  "Apply rule": "应用规则",
  "Add routing rule": "新增路由规则",
  "Edit routing rule": "编辑路由规则",
  "Delete routing rule": "删除路由规则",
  "Delete {name}? The rule remains active until you save the configuration.": "删除 {name}？保存配置前，该规则仍保持生效。",
  "Move up": "上移",
  "Move down": "下移",
  "No match values configured.": "至少需要配置一个匹配项。",
  "Domains {domains} · CIDRs {cidrs} · Protocols {protocols}": "域名 {domains} · CIDR {cidrs} · 协议 {protocols}",
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
let serviceTypes = [];
let selectedNode;
let stored;
let services = [];
let editingServiceID = null;
let routingRules = [];
let editingRoutingRuleID = null;
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
  elements.closeRoutingRuleDialog.title = text("Close");
  elements.closeRoutingRuleDialog.setAttribute("aria-label", text("Close"));
  updateNodeStatus();
  updateGeneration();
  renderServices();
  renderRoutingRules();
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
  elements.notice.textContent = text("Configuration changes are ready. Save to publish them.");
  elements.notice.hidden = false;
  elements.error.hidden = true;
}

function setBusy(value, label) {
  busy = value;
  elements.refreshButton.disabled = value;
  elements.nodeSelect.disabled = value || nodes.length === 0;
  elements.saveButton.disabled = value;
  elements.addServiceButton.disabled = value;
  elements.addRoutingRuleButton.disabled = value;
  elements.saveButton.textContent = value && label === "save" ? text("Saving...") : text("Save configuration");
  renderServices();
  renderRoutingRules();
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
  const serviceType = serviceTypes[0]?.id ?? "vless-reality";
  let suffix = services.length === 0 ? 1 : 2;
  let serviceID = services.length === 0 ? "vless-reality" : `vless-reality-${suffix}`;
  while (services.some((service) => service.service_id === serviceID)) {
    suffix += 1;
    serviceID = `vless-reality-${suffix}`;
  }
  let port = services.length === 0 ? 443 : 8443;
  while (services.some((service) => service.port === port) && port < 65535) port += 1;
  return {
    type: serviceType,
    enabled: true,
    service_id: serviceID,
    display_name: services.length === 0 ? "VLESS Reality" : `VLESS Reality ${services.length + 1}`,
    listen: "0.0.0.0",
    port,
    public_port: port,
    vless_reality: {
      target: "www.cloudflare.com:443",
      server_name: "www.cloudflare.com",
      fingerprint: "chrome",
    },
  };
}

function cloneServices(values) {
  return values.map((service) => ({
    ...service,
    vless_reality: service.vless_reality == null ? undefined : { ...service.vless_reality },
  }));
}

function cloneRoutingRules(values) {
  return values.map((rule) => ({
    ...rule,
    domains: [...(rule.domains ?? [])],
    ip_cidrs: [...(rule.ip_cidrs ?? [])],
    protocols: [...(rule.protocols ?? [])],
  }));
}

function populateConfiguration() {
  const configuration = stored?.exists ? stored.configuration : undefined;
  services = [];
  routingRules = [];
  if (configuration == null) {
    elements.xrayVersion.value = "26.3.27";
    elements.apiPort.value = "10085";
    services = [nextServiceDefaults()];
  } else {
    elements.xrayVersion.value = configuration.xray_version;
    elements.apiPort.value = String(configuration.api_port);
    services = cloneServices(Array.isArray(configuration.services) ? configuration.services : []);
    routingRules = cloneRoutingRules(Array.isArray(configuration.routing?.rules) ? configuration.routing.rules : []);
  }
  renderServices();
  renderRoutingRules();
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

function routingRuleStatus(rule) {
  return rule.enabled ? text("Enabled status") : text("Disabled status");
}

function routingRuleRow(rule, index) {
  const row = document.createElement("article");
  row.className = "routing-rule-row";

  const identity = document.createElement("div");
  identity.className = "service-identity";
  const heading = document.createElement("div");
  heading.className = "service-name-line";
  const name = document.createElement("strong");
  name.textContent = rule.display_name;
  const status = document.createElement("span");
  status.className = `badge service-status${rule.enabled ? " connected" : ""}`;
  status.textContent = routingRuleStatus(rule);
  const action = document.createElement("span");
  action.className = `badge${rule.action === "blocked" ? " destructive" : ""}`;
  action.textContent = text(rule.action === "blocked" ? "Block" : "Direct");
  heading.append(name, status, action);
  const identifier = document.createElement("span");
  identifier.className = "service-id";
  identifier.textContent = rule.rule_id;
  identity.append(heading, identifier);

  const matches = document.createElement("div");
  matches.className = "routing-rule-matches";
  matches.textContent = text("Domains {domains} · CIDRs {cidrs} · Protocols {protocols}", {
    domains: rule.domains?.length ?? 0,
    cidrs: rule.ip_cidrs?.length ?? 0,
    protocols: rule.protocols?.length ?? 0,
  });

  const actions = document.createElement("div");
  actions.className = "routing-rule-actions";
  const moveUp = document.createElement("button");
  moveUp.className = "icon-button";
  moveUp.type = "button";
  moveUp.disabled = busy || index === 0;
  moveUp.title = text("Move up");
  moveUp.setAttribute("aria-label", text("Move up"));
  moveUp.textContent = "↑";
  moveUp.addEventListener("click", () => moveRoutingRule(index, -1));
  const moveDown = document.createElement("button");
  moveDown.className = "icon-button";
  moveDown.type = "button";
  moveDown.disabled = busy || index === routingRules.length - 1;
  moveDown.title = text("Move down");
  moveDown.setAttribute("aria-label", text("Move down"));
  moveDown.textContent = "↓";
  moveDown.addEventListener("click", () => moveRoutingRule(index, 1));
  const edit = document.createElement("button");
  edit.className = "button button-outline button-compact";
  edit.type = "button";
  edit.disabled = busy;
  edit.textContent = text("Edit");
  edit.addEventListener("click", () => openRoutingRuleDialog(rule.rule_id));
  const remove = document.createElement("button");
  remove.className = "button button-ghost-destructive button-compact";
  remove.type = "button";
  remove.disabled = busy;
  remove.textContent = text("Delete");
  remove.addEventListener("click", () => void removeRoutingRule(rule.rule_id));
  actions.append(moveUp, moveDown, edit, remove);

  row.append(identity, matches, actions);
  return row;
}

function renderRoutingRules() {
  elements.routingRuleList.replaceChildren(...routingRules.map(routingRuleRow));
  elements.routingRuleListEmpty.hidden = routingRules.length !== 0;
}

function moveRoutingRule(index, offset) {
  const target = index + offset;
  if (busy || target < 0 || target >= routingRules.length) return;
  [routingRules[index], routingRules[target]] = [routingRules[target], routingRules[index]];
  renderRoutingRules();
  showPendingNotice();
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

async function loadServiceTypes() {
  const response = await client.rpc("service-types.list", {});
  serviceTypes = Array.isArray(response.service_types) ? response.service_types : [];
  if (serviceTypes.length === 0) throw new Error(text("The request could not be completed."));
}

function configurationForSave() {
  return {
    xray_version: elements.xrayVersion.value.trim(),
    api_port: numberValue("api-port"),
    services: cloneServices(services).sort((first, second) => first.service_id.localeCompare(second.service_id)),
    routing: { rules: cloneRoutingRules(routingRules) },
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
  const reality = service.vless_reality ?? {};
  elements.serviceEnabled.checked = service.enabled;
  elements.serviceId.value = service.service_id;
  elements.serviceId.disabled = editingServiceID != null;
  elements.displayName.value = service.display_name;
  elements.serviceTypeOutput.textContent = serviceTypes.find((candidate) => candidate.id === service.type)?.display_name ?? service.type;
  elements.serviceTypeOutput.dataset.type = service.type;
  elements.fingerprint.value = reality.fingerprint ?? "chrome";
  elements.listenAddress.value = service.listen;
  elements.listenPort.value = String(service.port);
  elements.publicPort.value = String(service.public_port);
	elements.realityTarget.value = reality.target ?? "";
	elements.serverName.value = reality.server_name ?? "";
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
  const type = elements.serviceTypeOutput.dataset.type || serviceTypes[0]?.id || "vless-reality";
  return {
    type,
    enabled: elements.serviceEnabled.checked,
    service_id: elements.serviceId.value.trim(),
    display_name: elements.displayName.value.trim(),
    listen: elements.listenAddress.value.trim(),
    port: numberValue("listen-port"),
    public_port: numberValue("public-port"),
    vless_reality: {
      target: elements.realityTarget.value.trim(),
      server_name: elements.serverName.value.trim(),
      fingerprint: elements.fingerprint.value,
    },
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

function nextRoutingRuleDefaults() {
  let suffix = routingRules.length + 1;
  let ruleID = `routing-rule-${suffix}`;
  while (routingRules.some((rule) => rule.rule_id === ruleID)) {
    suffix += 1;
    ruleID = `routing-rule-${suffix}`;
  }
  return {
    rule_id: ruleID,
    display_name: locale === "zh-CN" ? `路由规则 ${suffix}` : `Routing rule ${suffix}`,
    enabled: true,
    domains: [],
    ip_cidrs: [],
    protocols: [],
    action: "blocked",
  };
}

function lines(value) {
  return value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function populateRoutingRuleDialog(rule) {
  elements.routingRuleEnabled.checked = rule.enabled;
  elements.routingRuleId.value = rule.rule_id;
  elements.routingRuleId.disabled = editingRoutingRuleID != null;
  elements.routingRuleDisplayName.value = rule.display_name;
  elements.routingRuleAction.value = rule.action;
  elements.routingDomains.value = (rule.domains ?? []).join("\n");
  elements.routingIpCidrs.value = (rule.ip_cidrs ?? []).join("\n");
  const protocols = new Set(rule.protocols ?? []);
  for (const protocol of ["http", "tls", "quic", "bittorrent"]) {
    elements[`routingProtocol${protocol[0].toUpperCase()}${protocol.slice(1)}`].checked = protocols.has(protocol);
  }
  elements.routingRuleId.setCustomValidity("");
  elements.routingDomains.setCustomValidity("");
}

function openRoutingRuleDialog(ruleID = null) {
  if (busy) return;
  if (ruleID == null && routingRules.length >= 128) {
    showError(new Error(text("A node can contain at most 128 routing rules.")));
    return;
  }
  editingRoutingRuleID = ruleID;
  const rule = ruleID == null ? nextRoutingRuleDefaults() : routingRules.find((candidate) => candidate.rule_id === ruleID);
  if (rule == null) return;
  elements.routingRuleDialogTitle.textContent = text(ruleID == null ? "Add routing rule" : "Edit routing rule");
  populateRoutingRuleDialog(rule);
  elements.routingRuleDialog.showModal();
}

function closeRoutingRuleDialog() {
  editingRoutingRuleID = null;
  elements.routingRuleDialog.close();
}

function routingRuleFromDialog() {
  const protocols = ["http", "tls", "quic", "bittorrent"].filter((protocol) => {
    return elements[`routingProtocol${protocol[0].toUpperCase()}${protocol.slice(1)}`].checked;
  });
  return {
    rule_id: elements.routingRuleId.value.trim(),
    display_name: elements.routingRuleDisplayName.value.trim(),
    enabled: elements.routingRuleEnabled.checked,
    domains: lines(elements.routingDomains.value),
    ip_cidrs: lines(elements.routingIpCidrs.value),
    protocols,
    action: elements.routingRuleAction.value,
  };
}

function applyRoutingRule(event) {
  event.preventDefault();
  const candidate = routingRuleFromDialog();
  const duplicate = routingRules.some((rule) => rule.rule_id === candidate.rule_id && rule.rule_id !== editingRoutingRuleID);
  elements.routingRuleId.setCustomValidity(duplicate ? text("Rule ID already exists.") : "");
  const hasMatch = candidate.domains.length + candidate.ip_cidrs.length + candidate.protocols.length > 0;
  elements.routingDomains.setCustomValidity(hasMatch ? "" : text("No match values configured."));
  if (!elements.routingRuleForm.reportValidity()) return;
  if (editingRoutingRuleID == null) {
    routingRules.push(candidate);
  } else {
    const index = routingRules.findIndex((rule) => rule.rule_id === editingRoutingRuleID);
    if (index < 0) return;
    routingRules[index] = candidate;
  }
  closeRoutingRuleDialog();
  renderRoutingRules();
  showPendingNotice();
}

async function removeRoutingRule(ruleID) {
  if (busy) return;
  const rule = routingRules.find((candidate) => candidate.rule_id === ruleID);
  if (rule == null) return;
  let confirmed;
  try {
    confirmed = await client.confirm({
      title: text("Delete routing rule"),
      message: text("Delete {name}? The rule remains active until you save the configuration.", { name: rule.display_name }),
      confirm_label: text("Delete"),
      destructive: true,
    });
  } catch (cause) {
    showError(cause);
    return;
  }
  if (!confirmed) return;
  routingRules = routingRules.filter((candidate) => candidate.rule_id !== ruleID);
  renderRoutingRules();
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
elements.addRoutingRuleButton.addEventListener("click", () => openRoutingRuleDialog());
elements.closeServiceDialog.addEventListener("click", closeServiceDialog);
elements.cancelServiceButton.addEventListener("click", closeServiceDialog);
elements.serviceId.addEventListener("input", () => elements.serviceId.setCustomValidity(""));
elements.serviceForm.addEventListener("submit", applyService);
elements.closeRoutingRuleDialog.addEventListener("click", closeRoutingRuleDialog);
elements.cancelRoutingRuleButton.addEventListener("click", closeRoutingRuleDialog);
elements.routingRuleId.addEventListener("input", () => elements.routingRuleId.setCustomValidity(""));
elements.routingDomains.addEventListener("input", () => elements.routingDomains.setCustomValidity(""));
elements.routingIpCidrs.addEventListener("input", () => elements.routingDomains.setCustomValidity(""));
elements.routingRuleForm.addEventListener("submit", applyRoutingRule);
elements.configurationForm.addEventListener("submit", (event) => void saveConfiguration(event));
window.addEventListener("pagehide", () => client.dispose(), { once: true });

try {
  const context = await client.context();
  locale = context.locale;
  document.documentElement.dataset.theme = context.theme;
  translatePage();
  await loadServiceTypes();
  await refreshNodes();
} catch (cause) {
  showError(cause);
}

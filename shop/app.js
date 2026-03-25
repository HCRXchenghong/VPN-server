const state = {
  apiBase: "http://localhost:8080",
  accessToken: "",
  refreshToken: "",
  latestTopupOrder: null,
  plans: [],
  entitlements: [],
  devices: [],
  nodes: [],
};

const el = {
  apiBase: document.querySelector("#apiBase"),
  statusLine: document.querySelector("#statusLine"),
  walletBalance: document.querySelector("#walletBalance"),
  walletLedger: document.querySelector("#walletLedger"),
  plans: document.querySelector("#plans"),
  entitlements: document.querySelector("#entitlements"),
  devices: document.querySelector("#devices"),
  notices: document.querySelector("#notices"),
  connectDevice: document.querySelector("#connectDevice"),
  connectEntitlement: document.querySelector("#connectEntitlement"),
  connectNode: document.querySelector("#connectNode"),
  connectProtocol: document.querySelector("#connectProtocol"),
  profileOutput: document.querySelector("#profileOutput"),
};

document.querySelector("#loadPublic").addEventListener("click", loadPublicData);
document.querySelector("#registerForm").addEventListener("submit", onRegister);
document.querySelector("#loginForm").addEventListener("submit", onLogin);
document.querySelector("#verifyEmail").addEventListener("click", verifyEmail);
document.querySelector("#logoutBtn").addEventListener("click", logout);
document.querySelector("#topupForm").addEventListener("submit", onCreateTopup);
document.querySelector("#simulateCallback").addEventListener("click", simulateCallback);
document.querySelector("#reloadPlans").addEventListener("click", loadPlans);
document.querySelector("#reloadEntitlements").addEventListener("click", loadEntitlements);
document.querySelector("#reloadDevices").addEventListener("click", loadDevices);
document.querySelector("#deviceForm").addEventListener("submit", onBindDevice);
document.querySelector("#connectForm").addEventListener("submit", onConnect);
document.querySelector("#reloadNotices").addEventListener("click", loadNotices);

loadPublicData();

async function api(path, options = {}) {
  state.apiBase = el.apiBase.value.trim() || state.apiBase;
  const headers = {
    "Content-Type": "application/json",
    ...(options.headers || {}),
  };
  if (state.accessToken) {
    headers.Authorization = `Bearer ${state.accessToken}`;
  }
  const response = await fetch(`${state.apiBase}${path}`, { ...options, headers });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || `Request failed: ${response.status}`);
  }
  return payload;
}

async function loadPublicData() {
  await Promise.all([loadPlans(), loadNotices()]);
}

async function onRegister(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const payload = await api("/auth/register", {
    method: "POST",
    body: JSON.stringify({
      email: form.get("email"),
      password: form.get("password"),
    }),
  });
  setStatus(`已注册: ${payload.email}`);
}

async function onLogin(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const payload = await api("/auth/login", {
    method: "POST",
    body: JSON.stringify({
      email: form.get("email"),
      password: form.get("password"),
    }),
  });
  state.accessToken = payload.access_token;
  state.refreshToken = payload.refresh_token;
  setStatus(`已登录: ${payload.user.email}`);
  await loadPrivateData();
}

async function verifyEmail() {
  const user = await api("/auth/verify-email", { method: "POST" });
  setStatus(`邮箱已验证: ${user.email}`);
  await loadPrivateData();
}

function logout() {
  state.accessToken = "";
  state.refreshToken = "";
  state.latestTopupOrder = null;
  setStatus("未登录");
  renderList(el.walletLedger, []);
  renderList(el.entitlements, []);
  renderList(el.devices, []);
  renderProfile("");
  syncConnectOptions();
}

async function onCreateTopup(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  const payload = await api("/wallet/topups/alipay", {
    method: "POST",
    body: JSON.stringify({ points: Number(form.get("points")) }),
  });
  state.latestTopupOrder = payload.order;
  if (payload.payment?.order_string) {
    setStatus(`已创建充值单: ${payload.order.id}，后端已生成支付宝订单串`);
    renderProfile(payload.payment.order_string);
    return;
  }
  setStatus(`已创建充值单: ${payload.order.id}`);
}

async function simulateCallback() {
  if (!state.latestTopupOrder) {
    setStatus("没有待支付订单");
    return;
  }
  await api("/wallet/topups/alipay/callback", {
    method: "POST",
    body: JSON.stringify({
      order_id: state.latestTopupOrder.id,
      trade_no: `ALI_SIM_${Date.now()}`,
      status: "paid",
    }),
  });
  setStatus(`充值到账: ${state.latestTopupOrder.points} 积分`);
  await loadPrivateData();
}

async function loadPrivateData() {
  await Promise.all([loadWallet(), loadEntitlements(), loadDevices(), loadNodes()]);
}

async function loadWallet() {
  const payload = await api("/wallet/ledger");
  el.walletBalance.textContent = payload.balance;
  renderList(
    el.walletLedger,
    payload.items.map((item) => `
      <div class="list-item">
        <strong>${item.type}</strong>
        <div class="meta"><span>${item.points_delta > 0 ? "+" : ""}${item.points_delta} points</span><span>balance ${item.balance}</span></div>
        <div>${item.note}</div>
      </div>
    `),
  );
}

async function loadPlans() {
  state.plans = await api("/plans");
  el.plans.innerHTML = state.plans.map((plan) => `
    <div class="card">
      <h3>${plan.name}</h3>
      <div>${plan.description}</div>
      <div class="meta">
        <span>${plan.price_points} 积分</span>
        <span>${plan.duration_days} 天</span>
        <span>${plan.max_bound_devices} 台设备</span>
        <span>${plan.max_concurrent_sessions} 路并发</span>
      </div>
      <button data-plan-id="${plan.id}">兑换</button>
    </div>
  `).join("");
  el.plans.querySelectorAll("button[data-plan-id]").forEach((button) => {
    button.addEventListener("click", () => redeemPlan(button.dataset.planId));
  });
}

async function redeemPlan(planId) {
  const payload = await api("/redeems", {
    method: "POST",
    body: JSON.stringify({ plan_id: planId }),
  });
  setStatus(`兑换成功: ${payload.order.plan_name}`);
  await loadPrivateData();
}

async function loadEntitlements() {
  if (!state.accessToken) {
    state.entitlements = [];
    renderList(el.entitlements, []);
    syncConnectOptions();
    return;
  }
  state.entitlements = await api("/entitlements");
  renderList(
    el.entitlements,
    state.entitlements.map((item) => `
      <div class="list-item">
        <strong>${item.plan_id}</strong>
        <div class="meta">
          <span>${item.status}</span>
          <span>${item.max_bound_devices} 台设备</span>
          <span>${item.max_concurrent_sessions} 路并发</span>
        </div>
        <div>到期: ${new Date(item.ends_at).toLocaleString()}</div>
      </div>
    `),
  );
  syncConnectOptions();
}

async function onBindDevice(event) {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  await api("/devices/bind", {
    method: "POST",
    body: JSON.stringify({
      name: form.get("name"),
      platform: form.get("platform"),
    }),
  });
  setStatus("设备已绑定");
  event.currentTarget.reset();
  await loadDevices();
}

async function loadDevices() {
  if (!state.accessToken) {
    state.devices = [];
    renderList(el.devices, []);
    syncConnectOptions();
    return;
  }
  state.devices = await api("/devices");
  renderList(
    el.devices,
    state.devices.map((item) => `
      <div class="list-item">
        <strong>${item.name}</strong>
        <div class="meta"><span>${item.platform}</span><span>${item.status}</span></div>
      </div>
    `),
  );
  syncConnectOptions();
}

async function loadNodes() {
  if (!state.accessToken) {
    state.nodes = [];
    syncConnectOptions();
    return;
  }
  state.nodes = await api("/nodes");
  syncConnectOptions();
}

async function onConnect(event) {
  event.preventDefault();
  const payload = await api("/sessions/connect", {
    method: "POST",
    body: JSON.stringify({
      device_id: el.connectDevice.value,
      entitlement_id: el.connectEntitlement.value,
      node_id: el.connectNode.value,
      protocol: el.connectProtocol.value,
    }),
  });
  const profile = await api(
    `/profiles/${el.connectProtocol.value}?device_id=${encodeURIComponent(el.connectDevice.value)}&entitlement_id=${encodeURIComponent(el.connectEntitlement.value)}&node_id=${encodeURIComponent(el.connectNode.value)}`,
  );
  renderProfile(`# Session ${payload.id}\n${profile.config}`);
  setStatus(`连接已建立: ${payload.protocol} -> ${payload.node_id}`);
}

async function loadNotices() {
  const notices = await api("/notices");
  renderList(
    el.notices,
    notices.map((notice) => `
      <div class="list-item">
        <strong>${notice.title}</strong>
        <div class="meta"><span>${notice.severity}</span><span>${new Date(notice.created_at).toLocaleString()}</span></div>
        <div>${notice.body}</div>
      </div>
    `),
  );
}

function renderList(target, items) {
  target.innerHTML = items.length ? items.join("") : `<div class="list-item muted">暂无数据</div>`;
}

function renderProfile(text) {
  el.profileOutput.textContent = text || "连接配置会显示在这里。";
}

function syncConnectOptions() {
  setOptions(el.connectDevice, state.devices, "name");
  setOptions(el.connectEntitlement, state.entitlements, "plan_id");
  setOptions(el.connectNode, state.nodes, "name");
}

function setOptions(select, items, labelKey) {
  select.innerHTML = items.length
    ? items.map((item) => `<option value="${item.id}">${item[labelKey]}</option>`).join("")
    : `<option value="">暂无可选项</option>`;
}

function setStatus(message) {
  el.statusLine.textContent = message;
}

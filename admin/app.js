const el = {
  apiBase: document.querySelector("#apiBase"),
  adminKey: document.querySelector("#adminKey"),
  stats: document.querySelector("#stats"),
  users: document.querySelector("#users"),
  points: document.querySelector("#points"),
  orders: document.querySelector("#orders"),
  nodes: document.querySelector("#nodes"),
  audit: document.querySelector("#audit"),
};

document.querySelector("#refreshAll").addEventListener("click", refreshAll);
document.querySelectorAll("[data-refresh]").forEach((button) => {
  button.addEventListener("click", () => refreshSection(button.dataset.refresh));
});

refreshAll();

async function api(path, options = {}) {
  const response = await fetch(`${el.apiBase.value.trim()}${path}`, {
    method: options.method || "GET",
    headers: {
      "X-Admin-Key": el.adminKey.value.trim(),
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload.error || `Request failed: ${response.status}`);
  }
  return payload;
}

async function refreshAll() {
  await Promise.all([
    refreshSection("users"),
    refreshSection("points"),
    refreshSection("orders"),
    refreshSection("nodes"),
    refreshSection("audit"),
  ]);
}

async function refreshSection(section) {
  try {
    switch (section) {
      case "users":
        renderUsers(await api("/admin/users"));
        break;
      case "points":
        renderPoints(await api("/admin/points"));
        break;
      case "orders":
        renderOrders(await api("/admin/orders"));
        break;
      case "nodes":
        renderNodes(await api("/admin/nodes"));
        break;
      case "audit":
        renderAudit(await api("/admin/audit-logs"));
        break;
      default:
        break;
    }
  } catch (error) {
    renderError(section, error.message);
  }
}

function renderUsers(users) {
  el.users.innerHTML = users.length
    ? users
        .map(
          (user) => `
    <div class="item">
      <strong>${user.email}</strong>
      <div class="meta">
        <span>${user.status}</span>
        <span>${user.email_verified ? "verified" : "unverified"}</span>
        <span>${new Date(user.created_at).toLocaleString()}</span>
      </div>
    </div>
  `,
        )
        .join("")
    : empty();

  el.stats.innerHTML = `
    <div class="stat"><span>Users</span><strong>${users.length}</strong></div>
    <div class="stat"><span>Verified</span><strong>${users.filter((user) => user.email_verified).length}</strong></div>
    <div class="stat"><span>Active</span><strong>${users.filter((user) => user.status === "active").length}</strong></div>
    <div class="stat"><span>Device cap</span><strong>${users[0]?.bound_device_cap ?? 3}</strong></div>
  `;
}

function renderPoints(points) {
  el.points.innerHTML = points.length
    ? points
        .map(
          (item) => `
    <div class="item">
      <strong>${item.email}</strong>
      <div class="meta">
        <span>Balance ${item.balance}</span>
        <span>${item.verified ? "verified" : "unverified"}</span>
      </div>
    </div>
  `,
        )
        .join("")
    : empty();
}

function renderOrders(orders) {
  const topups = orders.topups || [];
  const redeems = orders.redeems || [];
  el.orders.innerHTML =
    [
      ...topups.map(
        (order) => `
    <div class="item">
      <strong>Topup ${order.id}</strong>
      <div class="meta">
        <span>${order.status}</span>
        <span>${order.points} points</span>
        <span>${order.payment_channel}</span>
        <span>${order.trade_no || "no-trade-no"}</span>
      </div>
      ${
        order.status !== "paid"
          ? `<button class="ghost js-query-topup" data-order-id="${order.id}">Query Status</button>`
          : ""
      }
      ${
        order.status !== "paid"
          ? `<button class="ghost js-confirm-topup" data-order-id="${order.id}">Manual Confirm</button>`
          : ""
      }
    </div>
  `,
      ),
      ...redeems.map(
        (order) => `
    <div class="item">
      <strong>Redeem ${order.plan_name}</strong>
      <div class="meta">
        <span>${order.status}</span>
        <span>${order.points_spent} points</span>
        <span>${order.entitlement_id}</span>
      </div>
    </div>
  `,
      ),
    ].join("") || empty();

  el.orders.querySelectorAll(".js-query-topup").forEach((button) => {
    button.addEventListener("click", () => queryTopup(button.dataset.orderId));
  });
  el.orders.querySelectorAll(".js-confirm-topup").forEach((button) => {
    button.addEventListener("click", () => manualConfirmTopup(button.dataset.orderId));
  });
}

async function queryTopup(orderId) {
  try {
    await api("/admin/orders/topups/query", {
      method: "POST",
      body: { order_id: orderId },
    });
    await refreshSection("orders");
    await refreshSection("points");
  } catch (error) {
    renderError("orders", error.message);
  }
}

async function manualConfirmTopup(orderId) {
  const tradeNo = window.prompt("Trade no (optional)", "");
  try {
    await api("/admin/orders/topups/confirm", {
      method: "POST",
      body: { order_id: orderId, trade_no: tradeNo || "" },
    });
    await refreshSection("orders");
    await refreshSection("points");
  } catch (error) {
    renderError("orders", error.message);
  }
}

function renderNodes(nodes) {
  el.nodes.innerHTML = nodes.length
    ? nodes
        .map(
          (node) => `
    <div class="item">
      <strong>${node.name}</strong>
      <div class="meta">
        <span>${node.status}</span>
        <span>${node.region}</span>
        <span>${node.group_id}</span>
      </div>
      <div class="meta">
        <span>WG ${node.wireguard_endpoint}</span>
        <span>IKEv2 ${node.ikev2_endpoint}</span>
      </div>
    </div>
  `,
        )
        .join("")
    : empty();
}

function renderAudit(audit) {
  el.audit.innerHTML = audit.length
    ? audit
        .slice(0, 30)
        .map(
          (item) => `
    <div class="item">
      <strong>${item.action}</strong>
      <div class="meta">
        <span>${item.actor_type}:${item.actor_id}</span>
        <span>${item.resource}:${item.resource_id}</span>
        <span>${new Date(item.created_at).toLocaleString()}</span>
      </div>
      <div>${item.description}</div>
    </div>
  `,
        )
        .join("")
    : empty();
}

function renderError(section, message) {
  const target = el[section];
  if (target) {
    target.innerHTML = `<div class="item">${message}</div>`;
  }
}

function empty() {
  return `<div class="item">No data</div>`;
}

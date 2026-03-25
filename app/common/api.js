import config from './config'

const state = {
  accessToken: '',
  refreshToken: '',
  user: null
}

function request(path, options = {}) {
  const header = {
    'Content-Type': 'application/json',
    ...(options.header || {})
  }
  if (state.accessToken) {
    header.Authorization = `Bearer ${state.accessToken}`
  }

  return new Promise((resolve, reject) => {
    uni.request({
      url: `${config.baseURL}${path}`,
      method: options.method || 'GET',
      data: options.data,
      header,
      success: (res) => {
        const payload = res.data || {}
        if (res.statusCode >= 400) {
          reject(new Error(payload.error || `request failed: ${res.statusCode}`))
          return
        }
        resolve(payload)
      },
      fail: reject
    })
  })
}

export function sessionState() {
  return state
}

export async function register(email, password) {
  return request('/auth/register', {
    method: 'POST',
    data: { email, password }
  })
}

export async function login(email, password) {
  const payload = await request('/auth/login', {
    method: 'POST',
    data: { email, password }
  })
  state.accessToken = payload.access_token
  state.refreshToken = payload.refresh_token
  state.user = payload.user
  return payload
}

export async function verifyEmail() {
  const user = await request('/auth/verify-email', { method: 'POST' })
  state.user = user
  return user
}

export async function walletSnapshot() {
  return request('/wallet/ledger')
}

export async function plans() {
  return request('/plans')
}

export async function topup(points) {
  return request('/wallet/topups/alipay', {
    method: 'POST',
    data: { points, scene: 'app' }
  })
}

export async function topupOrders() {
  return request('/wallet/topups')
}

export async function topupOrder(orderId) {
  return request(`/wallet/topups/${encodeURIComponent(orderId)}`)
}

export async function queryTopupOrder(orderId) {
  return request('/wallet/topups/query', {
    method: 'POST',
    data: { order_id: orderId }
  })
}

export async function entitlements() {
  return request('/entitlements')
}

export async function devices() {
  return request('/devices')
}

export async function bindDevice(name, platform) {
  return request('/devices/bind', {
    method: 'POST',
    data: { name, platform }
  })
}

export async function nodes() {
  return request('/nodes')
}

export async function connectSession(deviceId, entitlementId, nodeId, protocol) {
  return request('/sessions/connect', {
    method: 'POST',
    data: {
      device_id: deviceId,
      entitlement_id: entitlementId,
      node_id: nodeId,
      protocol
    }
  })
}

export async function profile(protocol, deviceId, entitlementId, nodeId) {
  return request(
    `/profiles/${protocol}?device_id=${encodeURIComponent(deviceId)}&entitlement_id=${encodeURIComponent(entitlementId)}&node_id=${encodeURIComponent(nodeId)}`
  )
}

export async function redeem(planId) {
  return request('/redeems', {
    method: 'POST',
    data: { plan_id: planId }
  })
}

export async function notices() {
  return request('/notices')
}

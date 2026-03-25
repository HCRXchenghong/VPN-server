<template>
  <view class="page">
    <view class="hero">
      <text class="eyebrow">HBuilderX / uni-app</text>
      <text class="title">Commercial VPN App</text>
      <text class="subtitle">{{ status }}</text>
    </view>

    <view class="card">
      <text class="section-title">Account</text>
      <input v-model.trim="form.email" class="input" placeholder="Email" />
      <input v-model="form.password" class="input" placeholder="Password" password />
      <view class="row">
        <button class="primary" @click="handleRegister">Register</button>
        <button class="primary ghost" @click="handleLogin">Login</button>
        <button class="primary ghost" @click="handleVerify">Verify Email</button>
      </view>
    </view>

    <view class="card">
      <view class="section-head">
        <text class="section-title">Wallet</text>
        <text class="metric">{{ wallet.balance }} points</text>
      </view>
      <view class="row">
        <input v-model.number="topupPoints" class="input" type="number" placeholder="Topup points" />
        <button class="primary" @click="handleTopup">Alipay Topup</button>
      </view>
      <view class="log" v-for="item in topupList" :key="item.id">
        <text>{{ item.id }}</text>
        <text class="muted">{{ item.status }} | {{ item.points }} points</text>
        <button v-if="item.status !== 'paid'" class="primary ghost mini" @click="handleQueryOrder(item.id)">Query</button>
      </view>
      <view class="log" v-for="item in wallet.items" :key="item.id || item.created_at">
        <text>{{ item.type }} {{ item.points_delta > 0 ? '+' : '' }}{{ item.points_delta }}</text>
        <text class="muted">{{ item.note }}</text>
      </view>
    </view>

    <view class="card">
      <view class="section-head">
        <text class="section-title">Plans</text>
        <button class="primary ghost" @click="loadPlans">Reload</button>
      </view>
      <view class="list-item" v-for="plan in planList" :key="plan.id">
        <view>
          <text class="item-title">{{ plan.name }}</text>
          <text class="muted">{{ plan.duration_days }} days / {{ plan.price_points }} points</text>
        </view>
        <button class="primary ghost" @click="handleRedeem(plan.id)">Redeem</button>
      </view>
    </view>

    <view class="card">
      <view class="section-head">
        <text class="section-title">Devices</text>
        <button class="primary ghost" @click="loadDevices">Reload</button>
      </view>
      <view class="row">
        <input v-model.trim="device.name" class="input" placeholder="Device name" />
        <picker :range="platformOptions" @change="onPlatformChange">
          <view class="picker">{{ device.platform }}</view>
        </picker>
        <button class="primary ghost" @click="handleBindDevice">Bind</button>
      </view>
      <view class="log" v-for="item in deviceList" :key="item.id">
        <text>{{ item.name }}</text>
        <text class="muted">{{ item.platform }} | {{ item.status }}</text>
      </view>
    </view>

    <view class="card">
      <view class="section-head">
        <text class="section-title">Connection</text>
        <button class="primary ghost" @click="loadConnectData">Reload</button>
      </view>
      <picker :range="deviceOptions" range-key="name" @change="onSelectDevice">
        <view class="picker">{{ selectedDeviceLabel }}</view>
      </picker>
      <picker :range="entitlementOptions" range-key="plan_id" @change="onSelectEntitlement">
        <view class="picker">{{ selectedEntitlementLabel }}</view>
      </picker>
      <picker :range="nodeOptions" range-key="name" @change="onSelectNode">
        <view class="picker">{{ selectedNodeLabel }}</view>
      </picker>
      <picker :range="protocolOptions" @change="onProtocolChange">
        <view class="picker">{{ protocol }}</view>
      </picker>
      <button class="primary" @click="handleConnect">Connect</button>
      <text class="code">{{ profileText }}</text>
    </view>

    <view class="card">
      <text class="section-title">Notices</text>
      <view class="log" v-for="item in noticeList" :key="item.id">
        <text>{{ item.title }}</text>
        <text class="muted">{{ item.body }}</text>
      </view>
    </view>
  </view>
</template>

<script>
import config from '../../common/config'
import {
  bindDevice,
  connectSession,
  devices,
  entitlements,
  login,
  nodes,
  notices,
  plans,
  profile,
  queryTopupOrder,
  redeem,
  register,
  topup,
  topupOrders,
  verifyEmail,
  walletSnapshot
} from '../../common/api'

export default {
  data() {
    return {
      status: 'Not logged in',
      form: {
        email: '',
        password: ''
      },
      wallet: {
        balance: 0,
        items: []
      },
      topupList: [],
      topupPoints: 100,
      planList: [],
      noticeList: [],
      device: {
        name: '',
        platform: 'ios'
      },
      platformOptions: ['ios', 'android'],
      deviceList: [],
      entitlementOptions: [],
      nodeOptions: [],
      protocolOptions: ['wireguard', 'ikev2'],
      protocol: 'wireguard',
      selectedDeviceIndex: 0,
      selectedEntitlementIndex: 0,
      selectedNodeIndex: 0,
      profileText: 'Connection profile will appear here.'
    }
  },
  computed: {
    deviceOptions() {
      return this.deviceList
    },
    selectedDeviceLabel() {
      return this.deviceOptions[this.selectedDeviceIndex]?.name || 'Select device'
    },
    selectedEntitlementLabel() {
      return this.entitlementOptions[this.selectedEntitlementIndex]?.plan_id || 'Select entitlement'
    },
    selectedNodeLabel() {
      return this.nodeOptions[this.selectedNodeIndex]?.name || 'Select node'
    }
  },
  onLoad() {
    this.bootstrap()
  },
  methods: {
    async bootstrap() {
      try {
        await Promise.all([this.loadPlans(), this.loadNotices()])
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleRegister() {
      try {
        const user = await register(this.form.email, this.form.password)
        this.status = `Registered: ${user.email}`
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleLogin() {
      try {
        const payload = await login(this.form.email, this.form.password)
        this.status = `Logged in: ${payload.user.email}`
        await this.refreshPrivateData()
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleVerify() {
      try {
        const user = await verifyEmail()
        this.status = `Email verified: ${user.email}`
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleTopup() {
      try {
        const payload = await topup(this.topupPoints)
        await this.loadTopupOrders()
        if (!payload.payment?.order_string) {
          this.status = `Created order ${payload.order.id}, but real Alipay config is not enabled on the backend`
          return
        }

        // #ifdef APP-PLUS
        await new Promise((resolve, reject) => {
          uni.requestPayment({
            provider: 'alipay',
            orderInfo: payload.payment.order_string,
            success: resolve,
            fail: reject
          })
        })
        this.status = `Alipay opened, polling order ${payload.order.id}`
        await this.pollTopupOrderStatus(payload.order.id)
        // #endif

        // #ifndef APP-PLUS
        this.status = `Created order ${payload.order.id}. In a native package, uni.requestPayment will launch Alipay directly.`
        this.profileText = payload.payment.order_string
        // #endif
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleQueryOrder(orderId) {
      try {
        const order = await queryTopupOrder(orderId)
        await this.loadTopupOrders()
        await this.loadWallet()
        this.status = `Order ${order.id} status: ${order.status}`
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleRedeem(planId) {
      try {
        await redeem(planId)
        this.status = 'Entitlement redeemed'
        await this.refreshPrivateData()
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleBindDevice() {
      try {
        await bindDevice(this.device.name, this.device.platform)
        this.device.name = ''
        this.status = 'Device bound'
        await this.loadDevices()
      } catch (error) {
        this.notifyError(error)
      }
    },
    async handleConnect() {
      try {
        const device = this.deviceOptions[this.selectedDeviceIndex]
        const entitlement = this.entitlementOptions[this.selectedEntitlementIndex]
        const node = this.nodeOptions[this.selectedNodeIndex]
        if (!device || !entitlement || !node) {
          throw new Error('Select device, entitlement, and node first')
        }
        await connectSession(device.id, entitlement.id, node.id, this.protocol)
        const payload = await profile(this.protocol, device.id, entitlement.id, node.id)
        this.profileText = payload.config
        this.status = `Connected with ${this.protocol}`
      } catch (error) {
        this.notifyError(error)
      }
    },
    async refreshPrivateData() {
      await Promise.all([
        this.loadWallet(),
        this.loadTopupOrders(),
        this.loadDevices(),
        this.loadEntitlements(),
        this.loadNodes()
      ])
    },
    async pollTopupOrderStatus(orderId) {
      for (let i = 0; i < config.walletPollRetries; i += 1) {
        await this.sleep(config.walletPollIntervalMs)
        const order = await queryTopupOrder(orderId)
        await this.loadTopupOrders()
        if (order.status === 'paid') {
          await this.loadWallet()
          this.status = `Order ${order.id} paid and credited`
          return
        }
      }
      this.status = `Order ${orderId} is still pending callback`
    },
    async loadWallet() {
      this.wallet = await walletSnapshot()
    },
    async loadTopupOrders() {
      this.topupList = await topupOrders()
    },
    async loadPlans() {
      this.planList = await plans()
    },
    async loadDevices() {
      this.deviceList = await devices()
    },
    async loadEntitlements() {
      this.entitlementOptions = await entitlements()
    },
    async loadNodes() {
      this.nodeOptions = await nodes()
    },
    async loadConnectData() {
      try {
        await Promise.all([this.loadDevices(), this.loadEntitlements(), this.loadNodes()])
      } catch (error) {
        this.notifyError(error)
      }
    },
    async loadNotices() {
      this.noticeList = await notices()
    },
    onPlatformChange(event) {
      this.device.platform = this.platformOptions[event.detail.value]
    },
    onSelectDevice(event) {
      this.selectedDeviceIndex = Number(event.detail.value)
    },
    onSelectEntitlement(event) {
      this.selectedEntitlementIndex = Number(event.detail.value)
    },
    onSelectNode(event) {
      this.selectedNodeIndex = Number(event.detail.value)
    },
    onProtocolChange(event) {
      this.protocol = this.protocolOptions[event.detail.value]
    },
    notifyError(error) {
      const message = error?.message || String(error)
      this.status = message
      uni.showToast({
        title: message,
        icon: 'none',
        duration: 2500
      })
    },
    sleep(ms) {
      return new Promise((resolve) => setTimeout(resolve, ms))
    }
  }
}
</script>

<style lang="scss">
.page {
  padding: 24rpx;
  display: grid;
  gap: 20rpx;
}

.hero,
.card {
  background: rgba(255, 255, 255, 0.88);
  border: 1rpx solid rgba(31, 68, 62, 0.08);
  border-radius: 28rpx;
  padding: 24rpx;
  box-shadow: 0 20rpx 60rpx rgba(18, 53, 49, 0.08);
}

.eyebrow {
  font-size: 22rpx;
  color: #0e7a6d;
  letter-spacing: 4rpx;
  text-transform: uppercase;
}

.title {
  display: block;
  margin-top: 12rpx;
  font-size: 52rpx;
  font-weight: 700;
}

.subtitle,
.muted {
  display: block;
  margin-top: 10rpx;
  color: #617775;
}

.section-title,
.item-title {
  font-size: 30rpx;
  font-weight: 600;
}

.section-head,
.row {
  display: flex;
  gap: 16rpx;
  align-items: center;
  justify-content: space-between;
}

.row {
  margin-top: 18rpx;
  flex-wrap: wrap;
}

.input,
.picker,
.code {
  width: 100%;
  margin-top: 16rpx;
  border-radius: 20rpx;
  background: #f4f7f6;
  padding: 22rpx;
}

.picker {
  color: #243534;
}

.primary {
  margin: 0;
  background: #0e7a6d;
  color: #fff;
  border-radius: 999rpx;
  padding: 0 28rpx;
}

.ghost {
  background: rgba(14, 122, 109, 0.12);
  color: #0e7a6d;
}

.mini {
  display: inline-flex;
  width: auto;
  margin-top: 12rpx;
}

.metric {
  color: #0e7a6d;
  font-weight: 700;
}

.list-item,
.log {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16rpx;
  padding: 18rpx 0;
  border-bottom: 1rpx solid rgba(31, 68, 62, 0.08);
}

.log {
  flex-wrap: wrap;
}

.code {
  white-space: pre-wrap;
  font-family: Consolas, monospace;
  color: #173330;
}
</style>

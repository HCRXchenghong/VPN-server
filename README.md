# VPN Platform Skeleton

This repository contains a runnable V1 skeleton for a commercial VPN platform with a Go control plane, HBuilderX mobile app shell, Windows client shell, admin console, storefront, and node agent.

## Layout

- `server/`: Go control plane with auth, wallet, plan redemption, device binding, session management, admin APIs, and agent endpoints.
- `agent/`: Go node agent skeleton that bootstraps and sends heartbeat events.
- `shop/`: Static storefront UI for registration, login, email verification, point top-ups, plan redemption, device binding, and connection testing.
- `admin/`: Static admin console for users, points, orders, nodes, and audit logs.
- `client_sdk/`: Shared Dart SDK for Flutter clients.
- `app/`: HBuilderX / uni-app mobile shell with Alipay App payment flow.
- `win/`: Flutter Windows shell wired to the shared client SDK.

## Current Progress

The following parts are already implemented in this repository:

- `server/`
  - Email and password registration, login, refresh token, logout, and email verification APIs.
  - Wallet ledger, Alipay top-up order creation, Alipay callback verification, admin manual confirmation, and proactive Alipay order query.
  - Plan listing, point redemption, entitlement issuance, device binding, concurrent session checks, node listing, protocol profile delivery, admin APIs, and audit logs.
  - Configurable storage backends: in-memory mode and PostgreSQL JSONB snapshot persistence.
- `agent/`
  - Node bootstrap and heartbeat flow for manual node enrollment.
- `app/`
  - HBuilderX / uni-app mobile shell with register, login, wallet, plans, top-up, proactive payment status query, notices, device binding, and connection test flow.
- `admin/`
  - Admin dashboard for users, points, orders, nodes, and audit logs.
  - Pending recharge orders can be actively queried against Alipay or manually confirmed.
- `shop/`
  - Web storefront shell for account actions, wallet actions, plan redemption, and basic device/session testing.
- `win/`
  - Windows Flutter shell scaffold wired to the shared Dart SDK.

## Current Boundary

The following parts are not finished yet:

- VPN tunnel implementations are still scaffold-level integrations. WireGuard and IKEv2 control-plane data paths exist, but production-grade tunnel lifecycle integration for Android, iOS, and Windows is not complete.
- PostgreSQL is currently used as a JSONB snapshot store, not yet split into normalized business tables such as `users`, `topup_orders`, and `wallet_ledger_entries`.
- Alipay is wired for real order creation, callback verification, and trade query, but this repository has not been verified against a live production Alipay account in this environment.
- iOS client packaging, App Store/TestFlight delivery workflow, real node-side WireGuard/strongSwan orchestration, and automated deployment are not implemented yet.
- Risk control, refund flow, customer support workflow, and commercial compliance materials are outside the current code scope.

## Repository Status

- Intended remote repository: `https://github.com/HCRXchenghong/VPN-server`
- This README reflects the implementation status as of `2026-03-25`.

## Run

### Start the Go API

```powershell
cd server
go run ./cmd/server
```

The API listens on `http://localhost:8080` by default.

Set `VPN_STORAGE_BACKEND=postgres` and `VPN_DATABASE_URL=...` if you want persistent PostgreSQL-backed state. The bootstrap schema is in `server/db/init/001_app_state_snapshots.sql`.

### Start the node agent

```powershell
cd agent
go run ./cmd/agent
```

### Open the storefront and admin shells

Open these files directly in a browser:

- `shop/index.html`
- `admin/index.html`

The admin shell uses `dev-admin-key` by default.

## HBuilder app

- Open `app/` with HBuilderX.
- In `manifest.json`, keep the `Payment` module enabled and ensure the Alipay SDK configuration is included before cloud packaging.
- Adjust `app/common/config.js` to point to your deployed API host.

## Real Alipay configuration

Copy `server/config.example.env` to your own environment file or system environment variables, then fill:

- `VPN_ALIPAY_APP_ID`
- `VPN_ALIPAY_NOTIFY_URL`
- `VPN_ALIPAY_PRIVATE_KEY` or `VPN_ALIPAY_PRIVATE_KEY_FILE`
- `VPN_ALIPAY_PUBLIC_KEY` or `VPN_ALIPAY_PUBLIC_KEY_FILE`

When these are set and `VPN_ALIPAY_ENABLED=true`, the backend will return a real `order_string` for `uni.requestPayment({ provider: 'alipay' })`.

## Notes

- Storage can now run in memory or PostgreSQL snapshot mode. PostgreSQL persists users, tokens, top-up orders, wallet ledger, entitlements, devices, sessions, nodes, notices, and audit logs in one JSONB-backed state record.
- The storefront still keeps a JSON callback simulator for local testing, but the backend now also supports real Alipay form callbacks with RSA2 verification.
- Flutter is not installed in the current environment, so the `win/` code was scaffolded but not executed here.

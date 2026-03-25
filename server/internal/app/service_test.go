package app

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"vpn/server/internal/payment"
)

func TestFullCommerceFlow(t *testing.T) {
	svc, err := NewService(Config{AdminAPIKey: "dev-admin-key"})
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	user, err := svc.Register(RegisterInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	user, err = svc.VerifyEmail(user.ID)
	if err != nil {
		t.Fatalf("verify email failed: %v", err)
	}
	if !user.EmailVerified {
		t.Fatalf("expected email verified")
	}

	login, err := svc.Login(LoginInput{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if login.AccessToken == "" || login.RefreshToken == "" {
		t.Fatalf("expected auth tokens")
	}

	order, err := svc.CreateTopup(user.ID, TopupInput{Points: 300})
	if err != nil {
		t.Fatalf("create topup failed: %v", err)
	}
	if _, err := svc.ConfirmTopup(ConfirmTopupInput{OrderID: order.ID, TradeNo: "ALI202603250001", Status: "paid"}); err != nil {
		t.Fatalf("confirm topup failed: %v", err)
	}
	if balance := svc.WalletBalance(user.ID); balance != 300 {
		t.Fatalf("unexpected balance: %d", balance)
	}

	redeemOrder, entitlement, err := svc.Redeem(user.ID, RedeemInput{PlanID: "plan_month"})
	if err != nil {
		t.Fatalf("redeem failed: %v", err)
	}
	if redeemOrder.Status != "paid" {
		t.Fatalf("expected paid redeem order")
	}

	device, err := svc.BindDevice(user.ID, BindDeviceInput{Name: "iPhone 16", Platform: "ios"})
	if err != nil {
		t.Fatalf("bind device failed: %v", err)
	}

	session, err := svc.Connect(user.ID, ConnectInput{
		DeviceID:      device.ID,
		EntitlementID: entitlement.ID,
		NodeID:        "node_tokyo_1",
		Protocol:      "wireguard",
	})
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	profile, err := svc.GetProfile(user.ID, "wireguard", device.ID, entitlement.ID, "node_tokyo_1")
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if profile.Protocol != "wireguard" {
		t.Fatalf("unexpected protocol: %s", profile.Protocol)
	}

	if err := svc.Disconnect(user.ID, session.ID); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
}

func TestConcurrentSessionLimit(t *testing.T) {
	svc, err := NewService(Config{AdminAPIKey: "dev-admin-key"})
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}
	user, err := svc.Register(RegisterInput{Email: "limit@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.VerifyEmail(user.ID); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	order, _ := svc.CreateTopup(user.ID, TopupInput{Points: 1000})
	if _, err := svc.ConfirmTopup(ConfirmTopupInput{OrderID: order.ID, TradeNo: "ALI202603250002", Status: "paid"}); err != nil {
		t.Fatalf("confirm topup failed: %v", err)
	}
	_, entitlement, err := svc.Redeem(user.ID, RedeemInput{PlanID: "plan_month"})
	if err != nil {
		t.Fatalf("redeem failed: %v", err)
	}

	first, _ := svc.BindDevice(user.ID, BindDeviceInput{Name: "iPhone", Platform: "ios"})
	second, _ := svc.BindDevice(user.ID, BindDeviceInput{Name: "Android", Platform: "android"})
	third, _ := svc.BindDevice(user.ID, BindDeviceInput{Name: "Windows", Platform: "windows"})

	if _, err := svc.Connect(user.ID, ConnectInput{DeviceID: first.ID, EntitlementID: entitlement.ID, NodeID: "node_tokyo_1", Protocol: "wireguard"}); err != nil {
		t.Fatalf("first connect failed: %v", err)
	}
	if _, err := svc.Connect(user.ID, ConnectInput{DeviceID: second.ID, EntitlementID: entitlement.ID, NodeID: "node_la_1", Protocol: "ikev2"}); err != nil {
		t.Fatalf("second connect failed: %v", err)
	}
	if _, err := svc.Connect(user.ID, ConnectInput{DeviceID: third.ID, EntitlementID: entitlement.ID, NodeID: "node_tokyo_1", Protocol: "wireguard"}); err == nil {
		t.Fatalf("expected concurrent limit error")
	}
}

func TestTopupOrderQueryAndManualConfirm(t *testing.T) {
	svc, err := NewService(Config{AdminAPIKey: "dev-admin-key"})
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	user, err := svc.Register(RegisterInput{Email: "orders@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.VerifyEmail(user.ID); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	checkout, err := svc.CreateTopupCheckout(user.ID, TopupInput{Points: 120})
	if err != nil {
		t.Fatalf("create checkout failed: %v", err)
	}

	orders := svc.ListUserTopupOrders(user.ID)
	if len(orders) != 1 || orders[0].ID != checkout.Order.ID {
		t.Fatalf("unexpected orders: %+v", orders)
	}

	order, err := svc.GetTopupOrder(user.ID, checkout.Order.ID)
	if err != nil {
		t.Fatalf("get topup order failed: %v", err)
	}
	if order.Status != "pending" {
		t.Fatalf("expected pending order")
	}

	order, err = svc.AdminConfirmTopup(AdminTopupConfirmInput{OrderID: checkout.Order.ID})
	if err != nil {
		t.Fatalf("manual confirm failed: %v", err)
	}
	if order.Status != "paid" {
		t.Fatalf("expected paid order, got %s", order.Status)
	}
	if balance := svc.WalletBalance(user.ID); balance != 120 {
		t.Fatalf("unexpected balance after manual confirm: %d", balance)
	}
}

func TestQueryUserTopupOrderSyncsPaidStatus(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		var bizContent map[string]string
		if err := json.Unmarshal([]byte(r.PostForm.Get("biz_content")), &bizContent); err != nil {
			t.Fatalf("unmarshal biz content: %v", err)
		}
		responsePayload := map[string]string{
			"code":         "10000",
			"msg":          "Success",
			"out_trade_no": bizContent["out_trade_no"],
			"trade_no":     "202603252200149999",
			"trade_status": "TRADE_SUCCESS",
			"total_amount": "12.00",
		}
		responseJSON, err := json.Marshal(responsePayload)
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashString(string(responseJSON)))
		if err != nil {
			t.Fatalf("sign response: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"alipay_trade_query_response": json.RawMessage(responseJSON),
			"sign":                        base64.StdEncoding.EncodeToString(signature),
		})
	}))
	defer gateway.Close()

	svc, err := NewService(Config{
		AdminAPIKey: "dev-admin-key",
		Alipay: payment.Config{
			Enabled:       true,
			AppID:         "2026032500000001",
			NotifyURL:     "https://example.com/wallet/topups/alipay/callback",
			GatewayURL:    gateway.URL,
			PrivateKeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})),
			PublicKeyPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})),
			HTTPClient:    gateway.Client(),
		},
	})
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	user, err := svc.Register(RegisterInput{Email: "query@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.VerifyEmail(user.ID); err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	checkout, err := svc.CreateTopupCheckout(user.ID, TopupInput{Points: 120})
	if err != nil {
		t.Fatalf("create checkout failed: %v", err)
	}

	order, err := svc.QueryUserTopupOrder(user.ID, checkout.Order.ID)
	if err != nil {
		t.Fatalf("query user topup order failed: %v", err)
	}
	if order.Status != "paid" {
		t.Fatalf("expected paid order after query, got %s", order.Status)
	}
	if balance := svc.WalletBalance(user.ID); balance != 120 {
		t.Fatalf("unexpected balance after query sync: %d", balance)
	}
}

func hashString(payload string) []byte {
	sum := sha256.Sum256([]byte(payload))
	return sum[:]
}

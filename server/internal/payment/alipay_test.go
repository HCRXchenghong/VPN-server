package payment

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
	"net/url"
	"strings"
	"testing"

	"vpn/server/internal/domain"
)

func TestCreateAndVerifyAppCheckout(t *testing.T) {
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

	ali, err := NewAlipay(Config{
		Enabled:       true,
		Sandbox:       true,
		AppID:         "2026032500000001",
		NotifyURL:     "https://example.com/wallet/topups/alipay/callback",
		PrivateKeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})),
		PublicKeyPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})),
	})
	if err != nil {
		t.Fatalf("new alipay: %v", err)
	}

	checkout, err := ali.CreateAppCheckout(domain.TopupOrder{
		ID:        "topup_123",
		UserID:    "user_123",
		Points:    100,
		AmountCNY: 10,
	})
	if err != nil {
		t.Fatalf("create checkout: %v", err)
	}
	if checkout == nil || !strings.Contains(checkout.OrderString, "alipay.trade.app.pay") {
		t.Fatalf("unexpected checkout payload: %+v", checkout)
	}

	values := url.Values{
		"app_id":       []string{"2026032500000001"},
		"out_trade_no": []string{"topup_123"},
		"trade_no":     []string{"202603252200149999"},
		"trade_status": []string{"TRADE_SUCCESS"},
		"total_amount": []string{"10.00"},
		"charset":      []string{"utf-8"},
		"seller_id":    []string{"208810217"},
		"sign_type":    []string{"RSA2"},
	}
	payload := canonicalValues(values)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, cryptoHash(), hashBytes(payload))
	if err != nil {
		t.Fatalf("sign notify payload: %v", err)
	}
	values.Set("sign", base64.StdEncoding.EncodeToString(signature))

	notify, err := ali.VerifyNotification(values)
	if err != nil {
		t.Fatalf("verify notification: %v", err)
	}
	if notify.OrderID != "topup_123" || notify.TradeStatus != "TRADE_SUCCESS" {
		t.Fatalf("unexpected notify: %+v", notify)
	}
}

func TestQueryTrade(t *testing.T) {
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostForm.Get("method"); got != "alipay.trade.query" {
			t.Fatalf("unexpected method: %s", got)
		}

		responsePayload := map[string]string{
			"code":         "10000",
			"msg":          "Success",
			"out_trade_no": "topup_123",
			"trade_no":     "202603252200149999",
			"trade_status": "TRADE_SUCCESS",
			"total_amount": "10.00",
		}
		responseJSON, err := json.Marshal(responsePayload)
		if err != nil {
			t.Fatalf("marshal query response: %v", err)
		}
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, cryptoHash(), hashBytes(string(responseJSON)))
		if err != nil {
			t.Fatalf("sign query response: %v", err)
		}
		envelope := map[string]any{
			"alipay_trade_query_response": json.RawMessage(responseJSON),
			"sign":                        base64.StdEncoding.EncodeToString(signature),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope)
	}))
	defer server.Close()

	ali, err := NewAlipay(Config{
		Enabled:       true,
		AppID:         "2026032500000001",
		NotifyURL:     "https://example.com/wallet/topups/alipay/callback",
		GatewayURL:    server.URL,
		PrivateKeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})),
		PublicKeyPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})),
		HTTPClient:    server.Client(),
	})
	if err != nil {
		t.Fatalf("new alipay: %v", err)
	}

	result, err := ali.QueryTrade(domain.TopupOrder{
		ID:        "topup_123",
		UserID:    "user_123",
		Points:    100,
		AmountCNY: 10,
	})
	if err != nil {
		t.Fatalf("query trade: %v", err)
	}
	if result.TradeStatus != "TRADE_SUCCESS" || result.OutTradeNo != "topup_123" {
		t.Fatalf("unexpected query result: %+v", result)
	}
}

func cryptoHash() crypto.Hash {
	return crypto.SHA256
}

func hashBytes(payload string) []byte {
	sum := sha256.Sum256([]byte(payload))
	return sum[:]
}

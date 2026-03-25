package payment

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"vpn/server/internal/domain"
)

type Config struct {
	Enabled        bool
	Sandbox        bool
	AppID          string
	GatewayURL     string
	NotifyURL      string
	SubjectPrefix  string
	PrivateKeyPEM  string
	PrivateKeyFile string
	PublicKeyPEM   string
	PublicKeyFile  string
	HTTPClient     *http.Client
}

type AppCheckout struct {
	Provider    string `json:"provider"`
	Scene       string `json:"scene"`
	OrderString string `json:"order_string"`
	NotifyURL   string `json:"notify_url"`
	GatewayURL  string `json:"gateway_url"`
	Sandbox     bool   `json:"sandbox"`
}

type Notification struct {
	OrderID     string
	TradeNo     string
	TradeStatus string
	TotalAmount string
	AppID       string
}

type QueryResult struct {
	Code        string `json:"code"`
	Msg         string `json:"msg"`
	SubCode     string `json:"sub_code"`
	SubMsg      string `json:"sub_msg"`
	TradeNo     string `json:"trade_no"`
	OutTradeNo  string `json:"out_trade_no"`
	TradeStatus string `json:"trade_status"`
	TotalAmount string `json:"total_amount"`
}

type Alipay struct {
	config     Config
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	client     *http.Client
}

func NewAlipay(config Config) (*Alipay, error) {
	privateKeyPEM, err := resolvePEM(config.PrivateKeyPEM, config.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	publicKeyPEM, err := resolvePEM(config.PublicKeyPEM, config.PublicKeyFile)
	if err != nil {
		return nil, err
	}

	config.PrivateKeyPEM = privateKeyPEM
	config.PublicKeyPEM = publicKeyPEM
	if config.SubjectPrefix == "" {
		config.SubjectPrefix = "VPN Points Recharge"
	}
	if config.GatewayURL == "" {
		if config.Sandbox {
			config.GatewayURL = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
		} else {
			config.GatewayURL = "https://openapi.alipay.com/gateway.do"
		}
	}

	ali := &Alipay{
		config: config,
		client: config.HTTPClient,
	}
	if ali.client == nil {
		ali.client = &http.Client{Timeout: 10 * time.Second}
	}
	if !ali.Enabled() {
		return ali, nil
	}

	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse alipay private key: %w", err)
	}
	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse alipay public key: %w", err)
	}
	ali.privateKey = privateKey
	ali.publicKey = publicKey
	return ali, nil
}

func (a *Alipay) Enabled() bool {
	return a != nil &&
		a.config.Enabled &&
		a.config.AppID != "" &&
		a.config.NotifyURL != "" &&
		(a.config.PrivateKeyPEM != "" || a.config.PrivateKeyFile != "") &&
		(a.config.PublicKeyPEM != "" || a.config.PublicKeyFile != "")
}

func (a *Alipay) CreateAppCheckout(order domain.TopupOrder) (*AppCheckout, error) {
	if !a.Enabled() {
		return nil, nil
	}

	bizContent, err := json.Marshal(map[string]string{
		"body":            fmt.Sprintf("%d points", order.Points),
		"subject":         fmt.Sprintf("%s %d points", a.config.SubjectPrefix, order.Points),
		"out_trade_no":    order.ID,
		"timeout_express": "30m",
		"total_amount":    formatAmount(order.AmountCNY),
		"product_code":    "QUICK_MSECURITY_PAY",
	})
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"app_id":      a.config.AppID,
		"biz_content": string(bizContent),
		"charset":     "utf-8",
		"format":      "json",
		"method":      "alipay.trade.app.pay",
		"notify_url":  a.config.NotifyURL,
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
	}
	if err != nil {
		return nil, err
	}
	params, err = a.signParams(params)
	if err != nil {
		return nil, err
	}

	return &AppCheckout{
		Provider:    "alipay",
		Scene:       "app",
		OrderString: encodedString(params),
		NotifyURL:   a.config.NotifyURL,
		GatewayURL:  a.config.GatewayURL,
		Sandbox:     a.config.Sandbox,
	}, nil
}

func (a *Alipay) QueryTrade(order domain.TopupOrder) (QueryResult, error) {
	if !a.Enabled() {
		return QueryResult{}, errors.New("alipay payment is not configured")
	}

	bizContent, err := json.Marshal(map[string]string{
		"out_trade_no": order.ID,
	})
	if err != nil {
		return QueryResult{}, err
	}

	params, err := a.signParams(map[string]string{
		"app_id":      a.config.AppID,
		"biz_content": string(bizContent),
		"charset":     "utf-8",
		"format":      "json",
		"method":      "alipay.trade.query",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
	})
	if err != nil {
		return QueryResult{}, err
	}

	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}

	request, err := http.NewRequest(http.MethodPost, a.config.GatewayURL, strings.NewReader(form.Encode()))
	if err != nil {
		return QueryResult{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := a.client.Do(request)
	if err != nil {
		return QueryResult{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return QueryResult{}, err
	}
	if response.StatusCode >= 300 {
		return QueryResult{}, fmt.Errorf("alipay query http status: %s", response.Status)
	}

	return a.decodeQueryResponse(body)
}

func (a *Alipay) VerifyNotification(values url.Values) (Notification, error) {
	if !a.Enabled() {
		return Notification{}, errors.New("alipay payment is not configured")
	}
	signature := values.Get("sign")
	if signature == "" {
		return Notification{}, errors.New("missing sign")
	}
	if signType := values.Get("sign_type"); signType != "" && !strings.EqualFold(signType, "RSA2") {
		return Notification{}, errors.New("unsupported sign_type")
	}

	if err := a.verify(canonicalValues(values), signature); err != nil {
		return Notification{}, err
	}

	return Notification{
		OrderID:     values.Get("out_trade_no"),
		TradeNo:     values.Get("trade_no"),
		TradeStatus: values.Get("trade_status"),
		TotalAmount: values.Get("total_amount"),
		AppID:       values.Get("app_id"),
	}, nil
}

func (a *Alipay) signParams(params map[string]string) (map[string]string, error) {
	unsigned := canonicalString(params)
	signature, err := a.sign(unsigned)
	if err != nil {
		return nil, err
	}
	signed := make(map[string]string, len(params)+1)
	for key, value := range params {
		signed[key] = value
	}
	signed["sign"] = signature
	return signed, nil
}

func (a *Alipay) decodeQueryResponse(body []byte) (QueryResult, error) {
	var envelope struct {
		Response json.RawMessage `json:"alipay_trade_query_response"`
		Sign     string          `json:"sign"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return QueryResult{}, err
	}
	if len(envelope.Response) == 0 {
		return QueryResult{}, errors.New("missing alipay_trade_query_response")
	}
	if envelope.Sign == "" {
		return QueryResult{}, errors.New("missing sign in gateway response")
	}
	if err := a.verify(bytes.TrimSpace(envelope.Response), envelope.Sign); err != nil {
		return QueryResult{}, fmt.Errorf("verify alipay query response: %w", err)
	}

	var result QueryResult
	if err := json.Unmarshal(envelope.Response, &result); err != nil {
		return QueryResult{}, err
	}
	if result.Code != "10000" && result.Code != "" {
		if result.SubCode != "" {
			return result, fmt.Errorf("alipay query failed: %s %s", result.SubCode, result.SubMsg)
		}
		return result, fmt.Errorf("alipay query failed: %s", result.Msg)
	}
	return result, nil
}

func canonicalString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func canonicalValues(values url.Values) string {
	params := make(map[string]string, len(values))
	for key, item := range values {
		if key == "sign" || key == "sign_type" || len(item) == 0 || item[0] == "" {
			continue
		}
		params[key] = item[0]
	}
	return canonicalString(params)
}

func encodedString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "" || params[key] == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+url.QueryEscape(params[key]))
	}
	return strings.Join(parts, "&")
}

func (a *Alipay) sign(payload string) (string, error) {
	digest := sha256.Sum256([]byte(payload))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (a *Alipay) verify(payload, signature string) error {
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(payload))
	return rsa.VerifyPKCS1v15(a.publicKey, crypto.SHA256, digest[:], decoded)
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("private key pem not found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parsePublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("public key pem not found")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return rsaKey, nil
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate public key is not RSA")
		}
		return rsaKey, nil
	}
	return nil, errors.New("unsupported public key format")
}

func resolvePEM(raw, file string) (string, error) {
	if raw != "" {
		return normalizePEM(raw), nil
	}
	if file == "" {
		return "", nil
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return normalizePEM(string(content)), nil
}

func normalizePEM(raw string) string {
	return strings.TrimSpace(strings.ReplaceAll(raw, "\\n", "\n"))
}

func formatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

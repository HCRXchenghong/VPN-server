package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"vpn/server/internal/app"
	"vpn/server/internal/httpapi"
	"vpn/server/internal/payment"
	"vpn/server/internal/state"
)

func main() {
	addr := os.Getenv("VPN_SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	store, err := buildStateStore()
	if err != nil {
		log.Fatal(err)
	}

	service, err := app.NewService(app.Config{
		AdminAPIKey: envOrDefault("VPN_ADMIN_API_KEY", "dev-admin-key"),
		Alipay: payment.Config{
			Enabled:        envBool("VPN_ALIPAY_ENABLED"),
			Sandbox:        envBool("VPN_ALIPAY_SANDBOX"),
			AppID:          os.Getenv("VPN_ALIPAY_APP_ID"),
			GatewayURL:     os.Getenv("VPN_ALIPAY_GATEWAY_URL"),
			NotifyURL:      os.Getenv("VPN_ALIPAY_NOTIFY_URL"),
			SubjectPrefix:  envOrDefault("VPN_ALIPAY_SUBJECT_PREFIX", "VPN Points Recharge"),
			PrivateKeyPEM:  os.Getenv("VPN_ALIPAY_PRIVATE_KEY"),
			PrivateKeyFile: os.Getenv("VPN_ALIPAY_PRIVATE_KEY_FILE"),
			PublicKeyPEM:   os.Getenv("VPN_ALIPAY_PUBLIC_KEY"),
			PublicKeyFile:  os.Getenv("VPN_ALIPAY_PUBLIC_KEY_FILE"),
		},
		StateStore: store,
	})
	if err != nil {
		log.Fatal(err)
	}

	api := httpapi.New(service)

	log.Printf("vpn control plane listening on %s using %s storage", addr, service.StorageBackend())
	if err := http.ListenAndServe(addr, api.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func buildStateStore() (state.Store, error) {
	backend := strings.ToLower(envOrDefault("VPN_STORAGE_BACKEND", "memory"))
	switch backend {
	case "memory":
		return state.NewMemoryStore(), nil
	case "postgres":
		databaseURL := os.Getenv("VPN_DATABASE_URL")
		if databaseURL == "" {
			return nil, fmt.Errorf("VPN_DATABASE_URL is required when VPN_STORAGE_BACKEND=postgres")
		}
		return state.NewPostgresStore(context.Background(), databaseURL)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", backend)
	}
}

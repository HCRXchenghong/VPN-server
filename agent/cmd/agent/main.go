package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type bootstrapRequest struct {
	NodeID string `json:"node_id"`
	Token  string `json:"token"`
}

type heartbeatRequest struct {
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

func main() {
	baseURL := envOrDefault("VPN_CONTROL_PLANE", "http://localhost:8080")
	nodeID := envOrDefault("VPN_NODE_ID", "node_tokyo_1")
	token := envOrDefault("VPN_NODE_TOKEN", "bootstrap_tokyo_01")

	if err := postJSON(baseURL+"/agent/bootstrap", bootstrapRequest{NodeID: nodeID, Token: token}); err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	log.Printf("node %s bootstrapped against %s", nodeID, baseURL)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := postJSON(baseURL+"/agent/heartbeat", heartbeatRequest{NodeID: nodeID, Status: "online"}); err != nil {
			log.Printf("heartbeat failed: %v", err)
			continue
		}
		log.Printf("heartbeat sent for %s", nodeID)
	}
}

func postJSON(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return &httpError{StatusCode: resp.StatusCode}
	}
	return nil
}

type httpError struct {
	StatusCode int
}

func (e *httpError) Error() string {
	return http.StatusText(e.StatusCode)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

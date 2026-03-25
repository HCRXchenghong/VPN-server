package domain

import "time"

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	EmailVerified  bool      `json:"email_verified"`
	Status         string    `json:"status"`
	DefaultRole    string    `json:"default_role"`
	CreatedAt      time.Time `json:"created_at"`
	LastLoginAt    time.Time `json:"last_login_at,omitempty"`
	BoundDeviceCap int       `json:"bound_device_cap"`
	ConcurrentCap  int       `json:"concurrent_cap"`
}

type WalletLedgerEntry struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	OrderID     string    `json:"order_id,omitempty"`
	Type        string    `json:"type"`
	PointsDelta int       `json:"points_delta"`
	Balance     int       `json:"balance"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type TopupOrder struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Points         int       `json:"points"`
	AmountCNY      float64   `json:"amount_cny"`
	Status         string    `json:"status"`
	PaymentChannel string    `json:"payment_channel"`
	TradeNo        string    `json:"trade_no,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	PaidAt         time.Time `json:"paid_at,omitempty"`
}

type RedeemOrder struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	PlanID        string    `json:"plan_id"`
	PlanName      string    `json:"plan_name"`
	PointsSpent   int       `json:"points_spent"`
	DurationDays  int       `json:"duration_days"`
	Status        string    `json:"status"`
	EntitlementID string    `json:"entitlement_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type Plan struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	PricePoints           int       `json:"price_points"`
	DurationDays          int       `json:"duration_days"`
	MaxBoundDevices       int       `json:"max_bound_devices"`
	MaxConcurrentSessions int       `json:"max_concurrent_sessions"`
	SupportedProtocols    []string  `json:"supported_protocols"`
	NodeGroups            []string  `json:"node_groups"`
	CreatedAt             time.Time `json:"created_at"`
}

type Entitlement struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	PlanID                string    `json:"plan_id"`
	Status                string    `json:"status"`
	SupportedProtocols    []string  `json:"supported_protocols"`
	NodeGroups            []string  `json:"node_groups"`
	MaxBoundDevices       int       `json:"max_bound_devices"`
	MaxConcurrentSessions int       `json:"max_concurrent_sessions"`
	StartsAt              time.Time `json:"starts_at"`
	EndsAt                time.Time `json:"ends_at"`
	CreatedAt             time.Time `json:"created_at"`
}

type Device struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Platform   string    `json:"platform"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ConnectionSession struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	DeviceID      string    `json:"device_id"`
	EntitlementID string    `json:"entitlement_id"`
	NodeID        string    `json:"node_id"`
	Protocol      string    `json:"protocol"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at,omitempty"`
}

type Node struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	GroupID           string    `json:"group_id"`
	Region            string    `json:"region"`
	WireGuardEndpoint string    `json:"wireguard_endpoint"`
	IKEv2Endpoint     string    `json:"ikev2_endpoint"`
	BootstrapToken    string    `json:"bootstrap_token,omitempty"`
	Status            string    `json:"status"`
	LastHeartbeatAt   time.Time `json:"last_heartbeat_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type NodeGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProtocolProfile struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	DeviceID      string            `json:"device_id"`
	EntitlementID string            `json:"entitlement_id"`
	NodeID        string            `json:"node_id"`
	Protocol      string            `json:"protocol"`
	Config        string            `json:"config"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
}

type Notice struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Severity  string    `json:"severity"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminRole struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditLog struct {
	ID          string    `json:"id"`
	ActorType   string    `json:"actor_type"`
	ActorID     string    `json:"actor_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	ResourceID  string    `json:"resource_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

package app

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"vpn/server/internal/domain"
	"vpn/server/internal/payment"
	"vpn/server/internal/platform"
	"vpn/server/internal/state"
)

var (
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidInput      = errors.New("invalid input")
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInsufficientPoint = errors.New("insufficient points")
)

type Config struct {
	AdminAPIKey string
	Alipay      payment.Config
	StateStore  state.Store
}

type Service struct {
	mu sync.RWMutex

	config Config
	alipay *payment.Alipay
	store  state.Store

	users         map[string]*domain.User
	usersByEmail  map[string]string
	tokens        map[string]string
	plans         map[string]*domain.Plan
	topupOrders   map[string]*domain.TopupOrder
	redeemOrders  map[string]*domain.RedeemOrder
	ledgerEntries map[string][]*domain.WalletLedgerEntry
	entitlements  map[string]*domain.Entitlement
	devices       map[string]*domain.Device
	sessions      map[string]*domain.ConnectionSession
	nodes         map[string]*domain.Node
	nodeGroups    map[string]*domain.NodeGroup
	notices       map[string]*domain.Notice
	roles         map[string]*domain.AdminRole
	auditLogs     []*domain.AuditLog
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	User         domain.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
}

func NewService(config Config) (*Service, error) {
	alipay, err := payment.NewAlipay(config.Alipay)
	if err != nil {
		return nil, err
	}
	if config.StateStore == nil {
		config.StateStore = state.NewMemoryStore()
	}

	svc := &Service{
		config:        config,
		alipay:        alipay,
		store:         config.StateStore,
		users:         map[string]*domain.User{},
		usersByEmail:  map[string]string{},
		tokens:        map[string]string{},
		plans:         map[string]*domain.Plan{},
		topupOrders:   map[string]*domain.TopupOrder{},
		redeemOrders:  map[string]*domain.RedeemOrder{},
		ledgerEntries: map[string][]*domain.WalletLedgerEntry{},
		entitlements:  map[string]*domain.Entitlement{},
		devices:       map[string]*domain.Device{},
		sessions:      map[string]*domain.ConnectionSession{},
		nodes:         map[string]*domain.Node{},
		nodeGroups:    map[string]*domain.NodeGroup{},
		notices:       map[string]*domain.Notice{},
		roles:         map[string]*domain.AdminRole{},
		auditLogs:     []*domain.AuditLog{},
	}

	snapshot, err := svc.store.Load(context.Background())
	switch {
	case errors.Is(err, state.ErrEmptySnapshot):
		svc.seed()
		if err := svc.saveLocked(); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		svc.applySnapshot(snapshot)
	}

	return svc, nil
}

func (s *Service) seed() {
	now := time.Now().UTC()

	for _, group := range []*domain.NodeGroup{
		{ID: "group_asia", Name: "Asia Premium", Description: "Low-latency Asia cluster", CreatedAt: now},
		{ID: "group_global", Name: "Global Access", Description: "Global fallback nodes", CreatedAt: now},
	} {
		s.nodeGroups[group.ID] = group
	}

	for _, node := range []*domain.Node{
		{
			ID:                "node_tokyo_1",
			Name:              "Tokyo-01",
			GroupID:           "group_asia",
			Region:            "ap-northeast-1",
			WireGuardEndpoint: "tokyo-01.example.com:51820",
			IKEv2Endpoint:     "tokyo-01.example.com",
			BootstrapToken:    "bootstrap_tokyo_01",
			Status:            "online",
			CreatedAt:         now,
		},
		{
			ID:                "node_la_1",
			Name:              "LosAngeles-01",
			GroupID:           "group_global",
			Region:            "us-west-1",
			WireGuardEndpoint: "la-01.example.com:51820",
			IKEv2Endpoint:     "la-01.example.com",
			BootstrapToken:    "bootstrap_la_01",
			Status:            "online",
			CreatedAt:         now,
		},
	} {
		s.nodes[node.ID] = node
	}

	for _, plan := range []*domain.Plan{
		{
			ID:                    "plan_month",
			Name:                  "Monthly Pass",
			Description:           "30 days, 3 devices, 2 concurrent sessions",
			PricePoints:           100,
			DurationDays:          30,
			MaxBoundDevices:       3,
			MaxConcurrentSessions: 2,
			SupportedProtocols:    []string{"wireguard", "ikev2"},
			NodeGroups:            []string{"group_asia", "group_global"},
			CreatedAt:             now,
		},
		{
			ID:                    "plan_quarter",
			Name:                  "Quarterly Pass",
			Description:           "90 days, 4 devices, 2 concurrent sessions",
			PricePoints:           270,
			DurationDays:          90,
			MaxBoundDevices:       4,
			MaxConcurrentSessions: 2,
			SupportedProtocols:    []string{"wireguard", "ikev2"},
			NodeGroups:            []string{"group_asia", "group_global"},
			CreatedAt:             now,
		},
		{
			ID:                    "plan_year",
			Name:                  "Annual Pass",
			Description:           "365 days, 6 devices, 3 concurrent sessions",
			PricePoints:           960,
			DurationDays:          365,
			MaxBoundDevices:       6,
			MaxConcurrentSessions: 3,
			SupportedProtocols:    []string{"wireguard", "ikev2"},
			NodeGroups:            []string{"group_asia", "group_global"},
			CreatedAt:             now,
		},
	} {
		s.plans[plan.ID] = plan
	}

	role := &domain.AdminRole{
		ID:          "role_super_admin",
		Name:        "super_admin",
		Permissions: []string{"users:*", "wallet:*", "plans:*", "nodes:*", "audit:*", "notices:*"},
		CreatedAt:   now,
	}
	s.roles[role.ID] = role

	notice := &domain.Notice{
		ID:        "notice_launch",
		Title:     "Service bootstrap",
		Body:      "First skeleton environment is ready for testing.",
		Severity:  "info",
		CreatedAt: now,
	}
	s.notices[notice.ID] = notice
	s.recordAudit("system", "bootstrap", "seed", "platform", "seed", "seeded default plans, nodes, and notices")
}

func (s *Service) Register(input RegisterInput) (domain.User, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if !strings.Contains(email, "@") || len(input.Password) < 8 {
		return domain.User{}, ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.usersByEmail[email]; exists {
		return domain.User{}, ErrConflict
	}

	user := &domain.User{
		ID:             platform.NewID("user"),
		Email:          email,
		PasswordHash:   platform.HashPassword(input.Password),
		EmailVerified:  false,
		Status:         "active",
		DefaultRole:    "user",
		CreatedAt:      time.Now().UTC(),
		BoundDeviceCap: 3,
		ConcurrentCap:  2,
	}
	s.users[user.ID] = user
	s.usersByEmail[email] = user.ID
	s.recordAudit("user", user.ID, "register", "user", user.ID, "user registered")
	if err := s.saveLocked(); err != nil {
		return domain.User{}, err
	}
	return *user, nil
}

func (s *Service) VerifyEmail(userID string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	user.EmailVerified = true
	s.recordAudit("user", user.ID, "verify_email", "user", user.ID, "email verified")
	if err := s.saveLocked(); err != nil {
		return domain.User{}, err
	}
	return *user, nil
}

func (s *Service) Login(input LoginInput) (LoginResult, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))

	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.usersByEmail[email]
	if !ok {
		return LoginResult{}, ErrUnauthorized
	}
	user := s.users[userID]
	if user.Status != "active" || !platform.PasswordMatches(input.Password, user.PasswordHash) {
		return LoginResult{}, ErrUnauthorized
	}

	user.LastLoginAt = time.Now().UTC()
	accessToken := platform.NewToken("access")
	refreshToken := platform.NewToken("refresh")
	s.tokens[accessToken] = user.ID
	s.tokens[refreshToken] = user.ID
	s.recordAudit("user", user.ID, "login", "session", accessToken, "user login")
	if err := s.saveLocked(); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: *user, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) Refresh(token string) (LoginResult, error) {
	user, err := s.Authenticate(token)
	if err != nil {
		return LoginResult{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)

	accessToken := platform.NewToken("access")
	refreshToken := platform.NewToken("refresh")
	s.tokens[accessToken] = user.ID
	s.tokens[refreshToken] = user.ID
	s.recordAudit("user", user.ID, "refresh", "session", accessToken, "token refreshed")
	if err := s.saveLocked(); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *Service) Logout(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.tokens[token]
	if !ok {
		return ErrUnauthorized
	}
	delete(s.tokens, token)
	s.recordAudit("user", userID, "logout", "session", token, "token invalidated")
	if err := s.saveLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) Authenticate(token string) (domain.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.User{}, ErrUnauthorized
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.tokens[token]
	if !ok {
		return domain.User{}, ErrUnauthorized
	}
	user, ok := s.users[userID]
	if !ok {
		return domain.User{}, ErrUnauthorized
	}
	return *user, nil
}

func (s *Service) ListPlans() []domain.Plan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.plans, func(plan *domain.Plan) string { return plan.ID })
}

func (s *Service) ListNodes() []domain.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.nodes, func(node *domain.Node) string { return node.ID })
}

func (s *Service) ListNotices() []domain.Notice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.notices, func(notice *domain.Notice) string { return notice.ID })
}

func (s *Service) ListUsers() []domain.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.users, func(user *domain.User) string { return user.Email })
}

func (s *Service) ListTopupOrders() []domain.TopupOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.topupOrders, func(order *domain.TopupOrder) string { return order.CreatedAt.Format(time.RFC3339Nano) })
}

func (s *Service) ListRedeemOrders() []domain.RedeemOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortValues(s.redeemOrders, func(order *domain.RedeemOrder) string { return order.CreatedAt.Format(time.RFC3339Nano) })
}

func (s *Service) ListAuditLogs() []domain.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.AuditLog, 0, len(s.auditLogs))
	for i := len(s.auditLogs) - 1; i >= 0; i-- {
		result = append(result, *s.auditLogs[i])
	}
	return result
}

func (s *Service) AdminAPIKey() string {
	return s.config.AdminAPIKey
}

func (s *Service) AlipayEnabled() bool {
	return s.alipay != nil && s.alipay.Enabled()
}

func (s *Service) StorageBackend() string {
	if s.store == nil {
		return "memory"
	}
	return s.store.Name()
}

func (s *Service) recordAudit(actorType, actorID, action, resource, resourceID, description string) {
	s.auditLogs = append(s.auditLogs, &domain.AuditLog{
		ID:          platform.NewID("audit"),
		ActorType:   actorType,
		ActorID:     actorID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	})
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func sortValues[T any](items map[string]*T, keyFn func(*T) string) []T {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keyFn(items[keys[i]]) < keyFn(items[keys[j]])
	})
	result := make([]T, 0, len(keys))
	for _, key := range keys {
		result = append(result, *items[key])
	}
	return result
}

func (s *Service) applySnapshot(snapshot state.Snapshot) {
	snapshot = state.NormalizeSnapshot(snapshot)
	s.users = snapshot.Users
	s.usersByEmail = snapshot.UsersByEmail
	s.tokens = snapshot.Tokens
	s.plans = snapshot.Plans
	s.topupOrders = snapshot.TopupOrders
	s.redeemOrders = snapshot.RedeemOrders
	s.ledgerEntries = snapshot.LedgerEntries
	s.entitlements = snapshot.Entitlements
	s.devices = snapshot.Devices
	s.sessions = snapshot.Sessions
	s.nodes = snapshot.Nodes
	s.nodeGroups = snapshot.NodeGroups
	s.notices = snapshot.Notices
	s.roles = snapshot.Roles
	s.auditLogs = snapshot.AuditLogs
}

func (s *Service) saveLocked() error {
	return s.store.Save(context.Background(), state.Snapshot{
		Users:         s.users,
		UsersByEmail:  s.usersByEmail,
		Tokens:        s.tokens,
		Plans:         s.plans,
		TopupOrders:   s.topupOrders,
		RedeemOrders:  s.redeemOrders,
		LedgerEntries: s.ledgerEntries,
		Entitlements:  s.entitlements,
		Devices:       s.devices,
		Sessions:      s.sessions,
		Nodes:         s.nodes,
		NodeGroups:    s.nodeGroups,
		Notices:       s.notices,
		Roles:         s.roles,
		AuditLogs:     s.auditLogs,
	})
}

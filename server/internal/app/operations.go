package app

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"vpn/server/internal/domain"
	"vpn/server/internal/payment"
	"vpn/server/internal/platform"
)

type TopupInput struct {
	Points int    `json:"points"`
	Scene  string `json:"scene,omitempty"`
}

type ConfirmTopupInput struct {
	OrderID string `json:"order_id"`
	TradeNo string `json:"trade_no"`
	Status  string `json:"status"`
}

type TopupCheckout struct {
	Order   domain.TopupOrder    `json:"order"`
	Payment *payment.AppCheckout `json:"payment,omitempty"`
}

type AdminTopupConfirmInput struct {
	OrderID string `json:"order_id"`
	TradeNo string `json:"trade_no"`
}

type AdminTopupQueryInput struct {
	OrderID string `json:"order_id"`
}

type RedeemInput struct {
	PlanID string `json:"plan_id"`
}

type BindDeviceInput struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type ConnectInput struct {
	DeviceID      string `json:"device_id"`
	EntitlementID string `json:"entitlement_id"`
	NodeID        string `json:"node_id"`
	Protocol      string `json:"protocol"`
}

type AgentBootstrapInput struct {
	NodeID string `json:"node_id"`
	Token  string `json:"token"`
}

type AgentHeartbeatInput struct {
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

func (s *Service) CreateTopup(userID string, input TopupInput) (domain.TopupOrder, error) {
	if input.Points <= 0 {
		return domain.TopupOrder{}, ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return domain.TopupOrder{}, ErrUnauthorized
	}
	if !user.EmailVerified {
		return domain.TopupOrder{}, fmt.Errorf("%w: email verification required", ErrConflict)
	}

	order := &domain.TopupOrder{
		ID:             platform.NewID("topup"),
		UserID:         userID,
		Points:         input.Points,
		AmountCNY:      float64(input.Points) / 10,
		Status:         "pending",
		PaymentChannel: "alipay",
		CreatedAt:      time.Now().UTC(),
	}
	s.topupOrders[order.ID] = order
	s.recordAudit("user", userID, "create_topup", "topup_order", order.ID, "created alipay topup order")
	if err := s.saveLocked(); err != nil {
		return domain.TopupOrder{}, err
	}
	return *order, nil
}

func (s *Service) CreateTopupCheckout(userID string, input TopupInput) (TopupCheckout, error) {
	order, err := s.CreateTopup(userID, input)
	if err != nil {
		return TopupCheckout{}, err
	}

	checkout, err := s.alipay.CreateAppCheckout(order)
	if err != nil {
		return TopupCheckout{}, err
	}
	return TopupCheckout{
		Order:   order,
		Payment: checkout,
	}, nil
}

func (s *Service) ConfirmTopup(input ConfirmTopupInput) (domain.TopupOrder, error) {
	if input.OrderID == "" || input.TradeNo == "" {
		return domain.TopupOrder{}, ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.topupOrders[input.OrderID]
	if !ok {
		return domain.TopupOrder{}, ErrNotFound
	}
	if order.Status == "paid" {
		return *order, nil
	}
	if strings.ToLower(input.Status) != "paid" {
		order.Status = "failed"
		s.recordAudit("system", "alipay", "callback_failed", "topup_order", order.ID, "topup marked failed")
		if err := s.saveLocked(); err != nil {
			return domain.TopupOrder{}, err
		}
		return *order, nil
	}

	order.Status = "paid"
	order.TradeNo = input.TradeNo
	order.PaidAt = time.Now().UTC()
	s.addLedgerEntryLocked(order.UserID, order.ID, "topup", order.Points, "alipay recharge")
	s.recordAudit("system", "alipay", "callback_paid", "topup_order", order.ID, "alipay topup confirmed")
	if err := s.saveLocked(); err != nil {
		return domain.TopupOrder{}, err
	}
	return *order, nil
}

func (s *Service) ConfirmTopupNotification(values url.Values) (domain.TopupOrder, error) {
	notification, err := s.alipay.VerifyNotification(values)
	if err != nil {
		return domain.TopupOrder{}, err
	}

	s.mu.RLock()
	order, ok := s.topupOrders[notification.OrderID]
	s.mu.RUnlock()
	if !ok {
		return domain.TopupOrder{}, ErrNotFound
	}
	if notification.AppID != "" && s.alipay.Enabled() && notification.AppID != s.config.Alipay.AppID {
		return domain.TopupOrder{}, fmt.Errorf("%w: alipay app_id mismatch", ErrConflict)
	}
	if notification.TotalAmount != "" && notification.TotalAmount != fmt.Sprintf("%.2f", order.AmountCNY) {
		return domain.TopupOrder{}, fmt.Errorf("%w: alipay amount mismatch", ErrConflict)
	}

	status := "failed"
	if notification.TradeStatus == "TRADE_SUCCESS" || notification.TradeStatus == "TRADE_FINISHED" {
		status = "paid"
	}
	return s.ConfirmTopup(ConfirmTopupInput{
		OrderID: notification.OrderID,
		TradeNo: notification.TradeNo,
		Status:  status,
	})
}

func (s *Service) ListUserTopupOrders(userID string) []domain.TopupOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.TopupOrder, 0)
	for _, order := range s.topupOrders {
		if order.UserID == userID {
			result = append(result, *order)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func (s *Service) GetTopupOrder(userID, orderID string) (domain.TopupOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.topupOrders[orderID]
	if !ok || order.UserID != userID {
		return domain.TopupOrder{}, ErrNotFound
	}
	return *order, nil
}

func (s *Service) AdminConfirmTopup(input AdminTopupConfirmInput) (domain.TopupOrder, error) {
	if input.TradeNo == "" {
		input.TradeNo = "ADMIN_MANUAL_" + platform.NewID("trade")
	}
	return s.ConfirmTopup(ConfirmTopupInput{
		OrderID: input.OrderID,
		TradeNo: input.TradeNo,
		Status:  "paid",
	})
}

func (s *Service) QueryUserTopupOrder(userID, orderID string) (domain.TopupOrder, error) {
	order, err := s.GetTopupOrder(userID, orderID)
	if err != nil {
		return domain.TopupOrder{}, err
	}
	return s.syncTopupOrder(order)
}

func (s *Service) AdminQueryTopup(input AdminTopupQueryInput) (domain.TopupOrder, error) {
	s.mu.RLock()
	order, ok := s.topupOrders[input.OrderID]
	s.mu.RUnlock()
	if !ok {
		return domain.TopupOrder{}, ErrNotFound
	}
	return s.syncTopupOrder(*order)
}

func (s *Service) syncTopupOrder(order domain.TopupOrder) (domain.TopupOrder, error) {
	if order.Status == "paid" {
		return order, nil
	}
	if !s.AlipayEnabled() {
		return order, fmt.Errorf("%w: alipay query is not configured", ErrConflict)
	}

	result, err := s.alipay.QueryTrade(order)
	if err != nil {
		return domain.TopupOrder{}, err
	}
	if result.OutTradeNo != "" && result.OutTradeNo != order.ID {
		return domain.TopupOrder{}, fmt.Errorf("%w: alipay order mismatch", ErrConflict)
	}
	if result.TotalAmount != "" && result.TotalAmount != fmt.Sprintf("%.2f", order.AmountCNY) {
		return domain.TopupOrder{}, fmt.Errorf("%w: alipay amount mismatch", ErrConflict)
	}

	switch result.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return s.ConfirmTopup(ConfirmTopupInput{
			OrderID: order.ID,
			TradeNo: result.TradeNo,
			Status:  "paid",
		})
	case "TRADE_CLOSED":
		s.mu.Lock()
		defer s.mu.Unlock()
		latest, ok := s.topupOrders[order.ID]
		if !ok {
			return domain.TopupOrder{}, ErrNotFound
		}
		if latest.Status == "paid" {
			return *latest, nil
		}
		latest.Status = "failed"
		if result.TradeNo != "" {
			latest.TradeNo = result.TradeNo
		}
		s.recordAudit("system", "alipay", "query_closed", "topup_order", latest.ID, "alipay trade query marked order failed")
		if err := s.saveLocked(); err != nil {
			return domain.TopupOrder{}, err
		}
		return *latest, nil
	default:
		s.mu.Lock()
		defer s.mu.Unlock()
		latest, ok := s.topupOrders[order.ID]
		if !ok {
			return domain.TopupOrder{}, ErrNotFound
		}
		if result.TradeNo != "" && latest.TradeNo == "" {
			latest.TradeNo = result.TradeNo
			if err := s.saveLocked(); err != nil {
				return domain.TopupOrder{}, err
			}
		}
		return *latest, nil
	}
}

func (s *Service) WalletBalance(userID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.ledgerEntries[userID]
	if len(entries) == 0 {
		return 0
	}
	return entries[len(entries)-1].Balance
}

func (s *Service) ListWalletLedger(userID string) []domain.WalletLedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.ledgerEntries[userID]
	result := make([]domain.WalletLedgerEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, *entry)
	}
	return result
}

func (s *Service) Redeem(userID string, input RedeemInput) (domain.RedeemOrder, domain.Entitlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, ok := s.plans[input.PlanID]
	if !ok {
		return domain.RedeemOrder{}, domain.Entitlement{}, ErrNotFound
	}
	balance := s.currentBalanceLocked(userID)
	if balance < plan.PricePoints {
		return domain.RedeemOrder{}, domain.Entitlement{}, ErrInsufficientPoint
	}

	now := time.Now().UTC()
	entitlement := &domain.Entitlement{
		ID:                    platform.NewID("ent"),
		UserID:                userID,
		PlanID:                plan.ID,
		Status:                "active",
		SupportedProtocols:    append([]string{}, plan.SupportedProtocols...),
		NodeGroups:            append([]string{}, plan.NodeGroups...),
		MaxBoundDevices:       plan.MaxBoundDevices,
		MaxConcurrentSessions: plan.MaxConcurrentSessions,
		StartsAt:              now,
		EndsAt:                now.AddDate(0, 0, plan.DurationDays),
		CreatedAt:             now,
	}
	order := &domain.RedeemOrder{
		ID:            platform.NewID("redeem"),
		UserID:        userID,
		PlanID:        plan.ID,
		PlanName:      plan.Name,
		PointsSpent:   plan.PricePoints,
		DurationDays:  plan.DurationDays,
		Status:        "paid",
		EntitlementID: entitlement.ID,
		CreatedAt:     now,
	}

	s.entitlements[entitlement.ID] = entitlement
	s.redeemOrders[order.ID] = order
	s.addLedgerEntryLocked(userID, order.ID, "redeem", -plan.PricePoints, "redeem entitlement "+plan.Name)
	s.recordAudit("user", userID, "redeem_plan", "entitlement", entitlement.ID, "user redeemed plan")
	if err := s.saveLocked(); err != nil {
		return domain.RedeemOrder{}, domain.Entitlement{}, err
	}
	return *order, *entitlement, nil
}

func (s *Service) ListEntitlements(userID string) []domain.Entitlement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Entitlement, 0)
	for _, entitlement := range s.entitlements {
		if entitlement.UserID == userID {
			result = append(result, *entitlement)
		}
	}
	return result
}

func (s *Service) ListDevices(userID string) []domain.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Device, 0)
	for _, device := range s.devices {
		if device.UserID == userID {
			result = append(result, *device)
		}
	}
	return result
}

func (s *Service) BindDevice(userID string, input BindDeviceInput) (domain.Device, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Platform) == "" {
		return domain.Device{}, ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.boundDeviceCountLocked(userID) >= s.deviceLimitLocked(userID) {
		return domain.Device{}, fmt.Errorf("%w: device limit reached", ErrConflict)
	}

	device := &domain.Device{
		ID:        platform.NewID("device"),
		UserID:    userID,
		Name:      strings.TrimSpace(input.Name),
		Platform:  strings.ToLower(strings.TrimSpace(input.Platform)),
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	s.devices[device.ID] = device
	s.recordAudit("user", userID, "bind_device", "device", device.ID, "device bound")
	if err := s.saveLocked(); err != nil {
		return domain.Device{}, err
	}
	return *device, nil
}

func (s *Service) UnbindDevice(userID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	device, ok := s.devices[deviceID]
	if !ok || device.UserID != userID {
		return ErrNotFound
	}
	for _, session := range s.sessions {
		if session.DeviceID == deviceID && session.Status == "connected" {
			return fmt.Errorf("%w: device still connected", ErrConflict)
		}
	}
	delete(s.devices, deviceID)
	s.recordAudit("user", userID, "unbind_device", "device", deviceID, "device removed")
	if err := s.saveLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) Connect(userID string, input ConnectInput) (domain.ConnectionSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entitlement, ok := s.entitlements[input.EntitlementID]
	if !ok || entitlement.UserID != userID {
		return domain.ConnectionSession{}, ErrNotFound
	}
	if entitlement.Status != "active" || time.Now().UTC().After(entitlement.EndsAt) {
		return domain.ConnectionSession{}, fmt.Errorf("%w: entitlement inactive", ErrConflict)
	}
	if !contains(entitlement.SupportedProtocols, strings.ToLower(input.Protocol)) {
		return domain.ConnectionSession{}, fmt.Errorf("%w: unsupported protocol", ErrInvalidInput)
	}

	device, ok := s.devices[input.DeviceID]
	if !ok || device.UserID != userID {
		return domain.ConnectionSession{}, ErrNotFound
	}
	node, ok := s.nodes[input.NodeID]
	if !ok || node.Status != "online" {
		return domain.ConnectionSession{}, ErrNotFound
	}
	if !contains(entitlement.NodeGroups, node.GroupID) {
		return domain.ConnectionSession{}, fmt.Errorf("%w: node not allowed", ErrConflict)
	}
	if s.activeSessionCountLocked(entitlement.ID) >= entitlement.MaxConcurrentSessions {
		return domain.ConnectionSession{}, fmt.Errorf("%w: concurrent session limit reached", ErrConflict)
	}

	device.LastSeenAt = time.Now().UTC()
	session := &domain.ConnectionSession{
		ID:            platform.NewID("sess"),
		UserID:        userID,
		DeviceID:      input.DeviceID,
		EntitlementID: entitlement.ID,
		NodeID:        input.NodeID,
		Protocol:      strings.ToLower(input.Protocol),
		Status:        "connected",
		StartedAt:     time.Now().UTC(),
	}
	s.sessions[session.ID] = session
	s.recordAudit("user", userID, "connect", "session", session.ID, "vpn session connected")
	if err := s.saveLocked(); err != nil {
		return domain.ConnectionSession{}, err
	}
	return *session, nil
}

func (s *Service) Disconnect(userID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || session.UserID != userID {
		return ErrNotFound
	}
	session.Status = "disconnected"
	session.EndedAt = time.Now().UTC()
	s.recordAudit("user", userID, "disconnect", "session", session.ID, "vpn session disconnected")
	if err := s.saveLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetProfile(userID, protocol, deviceID, entitlementID, nodeID string) (domain.ProtocolProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[deviceID]
	if !ok || device.UserID != userID {
		return domain.ProtocolProfile{}, ErrNotFound
	}
	entitlement, ok := s.entitlements[entitlementID]
	if !ok || entitlement.UserID != userID {
		return domain.ProtocolProfile{}, ErrNotFound
	}
	node, ok := s.nodes[nodeID]
	if !ok {
		return domain.ProtocolProfile{}, ErrNotFound
	}

	protocol = strings.ToLower(protocol)
	return domain.ProtocolProfile{
		ID:            platform.NewID("profile"),
		UserID:        userID,
		DeviceID:      deviceID,
		EntitlementID: entitlementID,
		NodeID:        nodeID,
		Protocol:      protocol,
		Config:        s.buildProfileConfig(protocol, *device, *entitlement, *node),
		Metadata: map[string]string{
			"node_region": node.Region,
			"device_name": device.Name,
		},
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) AgentBootstrap(input AgentBootstrapInput) (domain.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[input.NodeID]
	if !ok || node.BootstrapToken != input.Token {
		return domain.Node{}, ErrUnauthorized
	}
	node.Status = "online"
	node.LastHeartbeatAt = time.Now().UTC()
	s.recordAudit("agent", node.ID, "bootstrap", "node", node.ID, "node agent bootstrapped")
	if err := s.saveLocked(); err != nil {
		return domain.Node{}, err
	}
	return *node, nil
}

func (s *Service) AgentHeartbeat(input AgentHeartbeatInput) (domain.Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, ok := s.nodes[input.NodeID]
	if !ok {
		return domain.Node{}, ErrNotFound
	}
	if input.Status != "" {
		node.Status = input.Status
	}
	node.LastHeartbeatAt = time.Now().UTC()
	s.recordAudit("agent", node.ID, "heartbeat", "node", node.ID, "node heartbeat updated")
	if err := s.saveLocked(); err != nil {
		return domain.Node{}, err
	}
	return *node, nil
}

func (s *Service) addLedgerEntryLocked(userID, orderID, entryType string, delta int, note string) {
	balance := s.currentBalanceLocked(userID) + delta
	entry := &domain.WalletLedgerEntry{
		ID:          platform.NewID("ledger"),
		UserID:      userID,
		OrderID:     orderID,
		Type:        entryType,
		PointsDelta: delta,
		Balance:     balance,
		Note:        note,
		CreatedAt:   time.Now().UTC(),
	}
	s.ledgerEntries[userID] = append(s.ledgerEntries[userID], entry)
}

func (s *Service) currentBalanceLocked(userID string) int {
	entries := s.ledgerEntries[userID]
	if len(entries) == 0 {
		return 0
	}
	return entries[len(entries)-1].Balance
}

func (s *Service) boundDeviceCountLocked(userID string) int {
	count := 0
	for _, device := range s.devices {
		if device.UserID == userID && device.Status == "active" {
			count++
		}
	}
	return count
}

func (s *Service) deviceLimitLocked(userID string) int {
	limit := 3
	for _, entitlement := range s.entitlements {
		if entitlement.UserID == userID && entitlement.Status == "active" && time.Now().UTC().Before(entitlement.EndsAt) && entitlement.MaxBoundDevices > limit {
			limit = entitlement.MaxBoundDevices
		}
	}
	return limit
}

func (s *Service) activeSessionCountLocked(entitlementID string) int {
	count := 0
	for _, session := range s.sessions {
		if session.EntitlementID == entitlementID && session.Status == "connected" {
			count++
		}
	}
	return count
}

func (s *Service) buildProfileConfig(protocol string, device domain.Device, entitlement domain.Entitlement, node domain.Node) string {
	switch protocol {
	case "wireguard":
		return fmt.Sprintf("[Interface]\n# device=%s\nPrivateKey = generated-on-client\n[Peer]\nEndpoint = %s\nAllowedIPs = 0.0.0.0/0\n", device.ID, node.WireGuardEndpoint)
	case "ikev2":
		return fmt.Sprintf("remote=%s\nauth=eap-mschapv2\nuser=%s\nentitlement=%s\n", node.IKEv2Endpoint, device.UserID, entitlement.ID)
	default:
		return "unsupported protocol"
	}
}

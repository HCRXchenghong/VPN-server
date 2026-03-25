package state

import "vpn/server/internal/domain"

type Snapshot struct {
	Users         map[string]*domain.User                `json:"users"`
	UsersByEmail  map[string]string                      `json:"users_by_email"`
	Tokens        map[string]string                      `json:"tokens"`
	Plans         map[string]*domain.Plan                `json:"plans"`
	TopupOrders   map[string]*domain.TopupOrder          `json:"topup_orders"`
	RedeemOrders  map[string]*domain.RedeemOrder         `json:"redeem_orders"`
	LedgerEntries map[string][]*domain.WalletLedgerEntry `json:"ledger_entries"`
	Entitlements  map[string]*domain.Entitlement         `json:"entitlements"`
	Devices       map[string]*domain.Device              `json:"devices"`
	Sessions      map[string]*domain.ConnectionSession   `json:"sessions"`
	Nodes         map[string]*domain.Node                `json:"nodes"`
	NodeGroups    map[string]*domain.NodeGroup           `json:"node_groups"`
	Notices       map[string]*domain.Notice              `json:"notices"`
	Roles         map[string]*domain.AdminRole           `json:"roles"`
	AuditLogs     []*domain.AuditLog                     `json:"audit_logs"`
}

func NewSnapshot() Snapshot {
	return Snapshot{
		Users:         map[string]*domain.User{},
		UsersByEmail:  map[string]string{},
		Tokens:        map[string]string{},
		Plans:         map[string]*domain.Plan{},
		TopupOrders:   map[string]*domain.TopupOrder{},
		RedeemOrders:  map[string]*domain.RedeemOrder{},
		LedgerEntries: map[string][]*domain.WalletLedgerEntry{},
		Entitlements:  map[string]*domain.Entitlement{},
		Devices:       map[string]*domain.Device{},
		Sessions:      map[string]*domain.ConnectionSession{},
		Nodes:         map[string]*domain.Node{},
		NodeGroups:    map[string]*domain.NodeGroup{},
		Notices:       map[string]*domain.Notice{},
		Roles:         map[string]*domain.AdminRole{},
		AuditLogs:     []*domain.AuditLog{},
	}
}

func NormalizeSnapshot(snapshot Snapshot) Snapshot {
	normalized := NewSnapshot()
	if snapshot.Users != nil {
		normalized.Users = snapshot.Users
	}
	if snapshot.UsersByEmail != nil {
		normalized.UsersByEmail = snapshot.UsersByEmail
	}
	if snapshot.Tokens != nil {
		normalized.Tokens = snapshot.Tokens
	}
	if snapshot.Plans != nil {
		normalized.Plans = snapshot.Plans
	}
	if snapshot.TopupOrders != nil {
		normalized.TopupOrders = snapshot.TopupOrders
	}
	if snapshot.RedeemOrders != nil {
		normalized.RedeemOrders = snapshot.RedeemOrders
	}
	if snapshot.LedgerEntries != nil {
		normalized.LedgerEntries = snapshot.LedgerEntries
	}
	if snapshot.Entitlements != nil {
		normalized.Entitlements = snapshot.Entitlements
	}
	if snapshot.Devices != nil {
		normalized.Devices = snapshot.Devices
	}
	if snapshot.Sessions != nil {
		normalized.Sessions = snapshot.Sessions
	}
	if snapshot.Nodes != nil {
		normalized.Nodes = snapshot.Nodes
	}
	if snapshot.NodeGroups != nil {
		normalized.NodeGroups = snapshot.NodeGroups
	}
	if snapshot.Notices != nil {
		normalized.Notices = snapshot.Notices
	}
	if snapshot.Roles != nil {
		normalized.Roles = snapshot.Roles
	}
	if snapshot.AuditLogs != nil {
		normalized.AuditLogs = snapshot.AuditLogs
	}
	return normalized
}

package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"vpn/server/internal/app"
)

type API struct {
	service *app.Service
}

func New(service *app.Service) *API {
	return &API{service: service}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("POST /auth/register", a.handleRegister)
	mux.HandleFunc("POST /auth/login", a.handleLogin)
	mux.HandleFunc("POST /auth/refresh", a.handleRefresh)
	mux.HandleFunc("POST /auth/logout", a.withAuth(a.handleLogout))
	mux.HandleFunc("POST /auth/verify-email", a.withAuth(a.handleVerifyEmail))

	mux.HandleFunc("GET /plans", a.handleListPlans)
	mux.HandleFunc("POST /wallet/topups/alipay", a.withAuth(a.handleCreateTopup))
	mux.HandleFunc("POST /wallet/topups/alipay/callback", a.handleConfirmTopup)
	mux.HandleFunc("GET /wallet/topups", a.withAuth(a.handleTopupOrders))
	mux.HandleFunc("GET /wallet/topups/", a.withAuth(a.handleTopupOrder))
	mux.HandleFunc("POST /wallet/topups/query", a.withAuth(a.handleQueryTopupOrder))
	mux.HandleFunc("GET /wallet/ledger", a.withAuth(a.handleWalletLedger))
	mux.HandleFunc("POST /redeems", a.withAuth(a.handleRedeem))
	mux.HandleFunc("GET /entitlements", a.withAuth(a.handleEntitlements))
	mux.HandleFunc("GET /nodes", a.withAuth(a.handleNodes))

	mux.HandleFunc("GET /devices", a.withAuth(a.handleDevices))
	mux.HandleFunc("POST /devices/bind", a.withAuth(a.handleBindDevice))
	mux.HandleFunc("POST /devices/unbind", a.withAuth(a.handleUnbindDevice))
	mux.HandleFunc("POST /sessions/connect", a.withAuth(a.handleConnect))
	mux.HandleFunc("POST /sessions/disconnect", a.withAuth(a.handleDisconnect))
	mux.HandleFunc("GET /profiles/", a.withAuth(a.handleProfile))

	mux.HandleFunc("GET /notices", a.handleNotices)
	mux.HandleFunc("POST /agent/bootstrap", a.handleAgentBootstrap)
	mux.HandleFunc("POST /agent/heartbeat", a.handleAgentHeartbeat)

	mux.HandleFunc("GET /admin/users", a.withAdmin(a.handleAdminUsers))
	mux.HandleFunc("GET /admin/orders", a.withAdmin(a.handleAdminOrders))
	mux.HandleFunc("POST /admin/orders/topups/confirm", a.withAdmin(a.handleAdminConfirmTopup))
	mux.HandleFunc("POST /admin/orders/topups/query", a.withAdmin(a.handleAdminQueryTopup))
	mux.HandleFunc("GET /admin/points", a.withAdmin(a.handleAdminPoints))
	mux.HandleFunc("GET /admin/plans", a.withAdmin(a.handleAdminPlans))
	mux.HandleFunc("GET /admin/nodes", a.withAdmin(a.handleAdminNodes))
	mux.HandleFunc("GET /admin/notices", a.withAdmin(a.handleAdminNotices))
	mux.HandleFunc("GET /admin/audit-logs", a.withAdmin(a.handleAdminAuditLogs))
	return withCORS(mux)
}

func (a *API) withAuth(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		user, err := a.service.Authenticate(token)
		if err != nil {
			writeError(w, err)
			return
		}
		next(w, r, user.ID)
	}
}

func (a *API) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Admin-Key") != a.service.AdminAPIKey() {
			writeError(w, app.ErrUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input app.RegisterInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	user, err := a.service.Register(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input app.LoginInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	result, err := a.service.Login(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	result, err := a.service.Refresh(input.RefreshToken)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request, _ string) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if err := a.service.Logout(token); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := a.service.VerifyEmail(userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *API) handleListPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListPlans())
}

func (a *API) handleCreateTopup(w http.ResponseWriter, r *http.Request, userID string) {
	var input app.TopupInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	checkout, err := a.service.CreateTopupCheckout(userID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, checkout)
}

func (a *API) handleConfirmTopup(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if err := r.ParseForm(); err != nil {
			writeText(w, http.StatusBadRequest, "failure")
			return
		}
		if _, err := a.service.ConfirmTopupNotification(r.PostForm); err != nil {
			writeText(w, http.StatusBadRequest, "failure")
			return
		}
		writeText(w, http.StatusOK, "success")
		return
	}

	var input app.ConfirmTopupInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	order, err := a.service.ConfirmTopup(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) handleTopupOrders(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, a.service.ListUserTopupOrders(userID))
}

func (a *API) handleTopupOrder(w http.ResponseWriter, r *http.Request, userID string) {
	orderID := strings.TrimPrefix(r.URL.Path, "/wallet/topups/")
	order, err := a.service.GetTopupOrder(userID, orderID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) handleQueryTopupOrder(w http.ResponseWriter, r *http.Request, userID string) {
	var input struct {
		OrderID string `json:"order_id"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	order, err := a.service.QueryUserTopupOrder(userID, input.OrderID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) handleWalletLedger(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"balance": a.service.WalletBalance(userID),
		"items":   a.service.ListWalletLedger(userID),
	})
}

func (a *API) handleRedeem(w http.ResponseWriter, r *http.Request, userID string) {
	var input app.RedeemInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	order, entitlement, err := a.service.Redeem(userID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"order": order, "entitlement": entitlement})
}

func (a *API) handleEntitlements(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, a.service.ListEntitlements(userID))
}

func (a *API) handleNodes(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, a.service.ListNodes())
}

func (a *API) handleDevices(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, a.service.ListDevices(userID))
}

func (a *API) handleBindDevice(w http.ResponseWriter, r *http.Request, userID string) {
	var input app.BindDeviceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	device, err := a.service.BindDevice(userID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (a *API) handleUnbindDevice(w http.ResponseWriter, r *http.Request, userID string) {
	var input struct {
		DeviceID string `json:"device_id"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	if err := a.service.UnbindDevice(userID, input.DeviceID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *API) handleConnect(w http.ResponseWriter, r *http.Request, userID string) {
	var input app.ConnectInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	session, err := a.service.Connect(userID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (a *API) handleDisconnect(w http.ResponseWriter, r *http.Request, userID string) {
	var input struct {
		SessionID string `json:"session_id"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	if err := a.service.Disconnect(userID, input.SessionID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *API) handleProfile(w http.ResponseWriter, r *http.Request, userID string) {
	protocol := strings.TrimPrefix(r.URL.Path, "/profiles/")
	profile, err := a.service.GetProfile(
		userID,
		protocol,
		r.URL.Query().Get("device_id"),
		r.URL.Query().Get("entitlement_id"),
		r.URL.Query().Get("node_id"),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) handleNotices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListNotices())
}

func (a *API) handleAgentBootstrap(w http.ResponseWriter, r *http.Request) {
	var input app.AgentBootstrapInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	node, err := a.service.AgentBootstrap(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	var input app.AgentHeartbeatInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	node, err := a.service.AgentHeartbeat(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *API) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListUsers())
}

func (a *API) handleAdminOrders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"topups":  a.service.ListTopupOrders(),
		"redeems": a.service.ListRedeemOrders(),
	})
}

func (a *API) handleAdminConfirmTopup(w http.ResponseWriter, r *http.Request) {
	var input app.AdminTopupConfirmInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	order, err := a.service.AdminConfirmTopup(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) handleAdminQueryTopup(w http.ResponseWriter, r *http.Request) {
	var input app.AdminTopupQueryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, app.ErrInvalidInput)
		return
	}
	order, err := a.service.AdminQueryTopup(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (a *API) handleAdminPoints(w http.ResponseWriter, r *http.Request) {
	users := a.service.ListUsers()
	type summary struct {
		UserID   string `json:"user_id"`
		Email    string `json:"email"`
		Balance  int    `json:"balance"`
		Verified bool   `json:"verified"`
	}
	result := make([]summary, 0, len(users))
	for _, user := range users {
		result = append(result, summary{
			UserID:   user.ID,
			Email:    user.Email,
			Balance:  a.service.WalletBalance(user.ID),
			Verified: user.EmailVerified,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) handleAdminPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListPlans())
}

func (a *API) handleAdminNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListNodes())
}

func (a *API) handleAdminNotices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListNotices())
}

func (a *API) handleAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListAuditLogs())
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeText(w http.ResponseWriter, status int, payload string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(payload))
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, app.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, app.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, app.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, app.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, app.ErrInsufficientPoint):
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

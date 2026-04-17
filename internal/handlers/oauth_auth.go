package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/akozadaev/go_es_analytical_system/internal/auth"
)

type oauthCredsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// OAuthRegister проксирует POST /users на go_oauth2_server (саморегистрация).
func (h *Handlers) OAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.cfg.OAuth2BrowserProxyEnabled() {
		http.Error(w, "OAuth2 browser proxy is not configured", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	target := strings.TrimRight(strings.TrimSpace(h.cfg.OAuth2ServerURL), "/") + "/users"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		log.Printf("oauth register: build request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.oauthHTTPClient().Do(req)
	if err != nil {
		log.Printf("oauth register: upstream: %v", err)
		http.Error(w, "OAuth2 server unreachable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	upBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(upBody)
}

// OAuthLogin выполняет Resource Owner Password grant на go_oauth2_server и возвращает JSON токенов клиенту.
func (h *Handlers) OAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.cfg.OAuth2BrowserProxyEnabled() {
		http.Error(w, "OAuth2 browser proxy is not configured", http.StatusServiceUnavailable)
		return
	}

	var in oauthCredsRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", in.Username)
	form.Set("password", in.Password)
	form.Set("client_id", strings.TrimSpace(h.cfg.OAuth2ClientID))
	form.Set("client_secret", strings.TrimSpace(h.cfg.OAuth2ClientSecret))
	scope := strings.TrimSpace(h.cfg.OAuth2Scope)
	if scope == "" {
		scope = "read"
	}
	form.Set("scope", scope)

	target := strings.TrimRight(strings.TrimSpace(h.cfg.OAuth2ServerURL), "/") + "/token"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		log.Printf("oauth login: build request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.oauthHTTPClient().Do(req)
	if err != nil {
		log.Printf("oauth login: upstream: %v", err)
		http.Error(w, "OAuth2 server unreachable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	upBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(upBody)
}

func (h *Handlers) oauthHTTPClient() *http.Client {
	if h == nil {
		return &http.Client{Timeout: 15 * time.Second}
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// GetMe возвращает данные текущего пользователя по Bearer access token (роли для UI).
func (h *Handlers) GetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.cfg.OAuth2AuthEnabled() {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"oauth2_api_auth": false,
			"authenticated":   false,
		})
		return
	}

	sub, ok := auth.SubjectFromContext(r.Context())
	if !ok || sub.UserID == "" {
		writeAuthJSON(w, http.StatusUnauthorized, "missing_bearer_token", "Authorization: Bearer access_token is required")
		return
	}

	mapIndexer := auth.HasMapIndexerAccess(sub.Roles)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"oauth2_api_auth": true,
		"authenticated":   true,
		"user_id":         sub.UserID,
		"client_id":       sub.ClientID,
		"roles":           sub.Roles,
		"map_indexer":     mapIndexer,
	})
}

func writeAuthJSON(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

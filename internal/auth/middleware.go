package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/akozadaev/go_es_analytical_system/internal/config"
	"github.com/gorilla/mux"
)

// Middleware возвращает mux.MiddlewareFunc: защищает маршруты при включённой OAuth2-конфигурации.
// Публичные пути: /health, /readiness, /swagger (префикс).
// Режимы (как в go_oauth2_server): приоритет — OAUTH2_INTROSPECT_URL, иначе локальная проверка JWT по OAUTH2_JWT_SECRET.
func Middleware(cfg *config.Config) mux.MiddlewareFunc {
	if cfg == nil || (!cfg.OAuth2AuthEnabled()) {
		return func(next http.Handler) http.Handler { return next }
	}

	introspectURL := strings.TrimSpace(cfg.OAuth2IntrospectURL)
	jwtSecret := []byte(strings.TrimSpace(cfg.OAuth2JWTSecret))

	httpClient := &http.Client{Timeout: 10 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			token := BearerToken(r)
			if token == "" {
				writeAuthJSON(w, http.StatusUnauthorized, "missing_bearer_token", "Authorization: Bearer access_token is required")
				return
			}

			var sub Subject
			var err error
			if introspectURL != "" {
				sub, err = IntrospectToken(r.Context(), httpClient, introspectURL, token)
			} else {
				sub, err = ValidateJWTAccessToken(token, jwtSecret)
			}
			if err != nil {
				log.Printf("oauth2 auth: %v", err)
				writeAuthJSON(w, http.StatusUnauthorized, "invalid_token", "access token rejected")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithSubject(r.Context(), sub)))
		})
	}
}

func isPublicPath(path string) bool {
	switch {
	case path == "/health" || path == "/readiness":
		return true
	case strings.HasPrefix(path, "/swagger"):
		return true
	case path == "/api/auth/login" || path == "/api/auth/register":
		return true
	case path == "/" || strings.HasPrefix(path, "/app/"):
		return true
	case strings.HasPrefix(path, "/map-indexer/"):
		return true
	default:
		return false
	}
}

func writeAuthJSON(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

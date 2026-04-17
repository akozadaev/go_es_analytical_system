// Package auth — проверка access token от go_oauth2_server (JWT HS256 или POST /introspect).
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Subject — данные субъекта после успешной проверки токена.
type Subject struct {
	UserID   string
	ClientID string
	Roles    []string
}

type ctxKey int

const subjectKey ctxKey = 1

// WithSubject кладёт Subject в контекст запроса.
func WithSubject(ctx context.Context, s Subject) context.Context {
	return context.WithValue(ctx, subjectKey, s)
}

// SubjectFromContext возвращает Subject, если middleware уже отработал.
func SubjectFromContext(ctx context.Context) (Subject, bool) {
	v := ctx.Value(subjectKey)
	if v == nil {
		return Subject{}, false
	}
	s, ok := v.(Subject)
	return s, ok
}

// IntrospectRequest соответствует go_oauth2_server/internal/models.IntrospectRequest.
type IntrospectRequest struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty"`
}

// IntrospectResponse соответствует go_oauth2_server/internal/models.IntrospectResponse.
type IntrospectResponse struct {
	Active   bool     `json:"active"`
	ClientID string   `json:"client_id,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Scope    string   `json:"scope,omitempty"`
	Exp      int64    `json:"exp,omitempty"`
}

// ValidateJWTAccessToken повторяет логику validateJWTToken из go_oauth2_server/internal/handlers/handlers.go.
func ValidateJWTAccessToken(tokenString string, jwtSecret []byte) (Subject, error) {
	if len(jwtSecret) == 0 {
		return Subject{}, errors.New("jwt secret is not configured")
	}

	token, err := jwtlib.Parse(tokenString, func(token *jwtlib.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return Subject{}, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return Subject{}, errors.New("invalid claims")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return Subject{}, errors.New("token expired")
		}
	}

	clientID, _ := claims["aud"].(string)
	userID, _ := claims["sub"].(string)

	return Subject{
		UserID:   userID,
		ClientID: clientID,
		Roles:    rolesFromJWTClaims(claims),
	}, nil
}

func rolesFromJWTClaims(claims jwtlib.MapClaims) []string {
	raw, ok := claims["roles"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// IntrospectToken вызывает OAuth2-сервер (RFC 7662-подобный JSON endpoint проекта go_oauth2_server).
func IntrospectToken(ctx context.Context, client *http.Client, introspectURL, accessToken string) (Subject, error) {
	if introspectURL == "" {
		return Subject{}, errors.New("introspect URL is not configured")
	}
	if client == nil {
		client = http.DefaultClient
	}

	body, err := json.Marshal(IntrospectRequest{
		Token:         accessToken,
		TokenTypeHint: "access_token",
	})
	if err != nil {
		return Subject{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectURL, bytes.NewReader(body))
	if err != nil {
		return Subject{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return Subject{}, err
	}
	defer resp.Body.Close()

	var out IntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Subject{}, err
	}
	if resp.StatusCode != http.StatusOK || !out.Active {
		return Subject{}, errors.New("token not active")
	}

	return Subject{
		UserID:   out.UserID,
		ClientID: out.ClientID,
		Roles:    out.Roles,
	}, nil
}

// BearerToken извлекает токен из заголовка Authorization: Bearer …
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const p = "Bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

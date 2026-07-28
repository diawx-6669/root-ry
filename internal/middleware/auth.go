package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const UsernameKey contextKey = "username"
const IsAdminKey contextKey = "isAdmin"

var jwtSecret = []byte("rootry_dev_secret_change_in_prod")

// SetJWTSecret allows the secret to be set from environment variable at startup
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	IsAdmin  bool   `json:"adm"`
	Exp      int64  `json:"exp"`
}

func GenerateToken(userID int64, username string, isAdmin bool) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		Exp:      time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := sign(header + "." + body)
	return header + "." + body + "." + sig, nil
}

func sign(data string) string {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ParseToken(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	// Fix: correct argument order for hmac.Equal — computed sig first, provided sig second
	computed := sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(computed), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	return &claims, nil
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Требуется авторизация"}`))
			return
		}
		claims, err := ParseToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Недействительный токен"}`))
			return
		}
		ctx := context.WithValue(r.Context(), UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, IsAdminKey, claims.IsAdmin)
		next(w, r.WithContext(ctx))
	}
}

func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, _ := r.Context().Value(IsAdminKey).(bool)
		if !isAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Доступ запрещён"}`))
			return
		}
		next(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

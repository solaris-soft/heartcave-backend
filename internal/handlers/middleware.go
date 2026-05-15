package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const customerIDKey contextKey = "customerID"

func CustomerID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(customerIDKey).(int64)
	return id, ok
}

func CustomerRequired(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject, ok := bearerSubject(r, jwtSecret)
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			customerID, err := strconv.ParseInt(subject, 10, 64)
			if err != nil || customerID <= 0 {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := context.WithValue(r.Context(), customerIDKey, customerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminRequired(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subject, ok := bearerSubject(r, jwtSecret)
			if !ok || subject != "admin" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerSubject(r *http.Request, jwtSecret string) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}

	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", false
	}
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", false
	}
	return subject, true
}

package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
)

type AuthService interface {
	HashRefreshToken(token string) string
	MakeAccessToken(userID uuid.UUID) (string, error)
	NewRefreshToken() (string, error)
	ValidateAccessToken(tokenString string) (uuid.UUID, error)
}

type AuthHandler struct {
	DB          database.Querier
	AuthService AuthService
}

func NewAuthHandler(db database.Querier, authService AuthService) AuthHandler {
	return AuthHandler{
		DB:          db,
		AuthService: authService,
	}
}

func (h AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {}

func (h AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {}

func (h AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

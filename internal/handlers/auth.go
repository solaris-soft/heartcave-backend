package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/services"
)

type AuthHandler struct {
	DB          database.Querier
	AuthService services.AuthService
}

func NewAuthHandler(db database.Querier, authService services.AuthService) AuthHandler {
	return AuthHandler{
		DB:          db,
		AuthService: authService,
	}
}

type UserResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

func validateEmail(email string) (string, error) {
	if len(email) == 0 {
		return "", errors.New("invalid email")
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return "", errors.New("invalid email")
	}
	return strings.ToLower(email), nil
}

type refreshMeta struct {
	TokenHash string
	UserID    uuid.UUID
}

type refreshKey struct{}

var refreshCtxKey = refreshKey{}

func (h AuthHandler) issueRefreshToken(ctx context.Context, userID uuid.UUID, r *http.Request) (string, error) {
	rawToken, err := h.AuthService.NewRefreshToken()
	if err != nil {
		return "", err
	}
	_, err = h.DB.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: h.AuthService.HashRefreshToken(rawToken),
		ExpiresAt: time.Now().Add(services.RefreshTokenExpiry),
		UserAgent: sql.NullString{
			String: r.UserAgent(),
			Valid:  true,
		},
		IpAddress: sql.NullString{
			String: r.RemoteAddr,
			Valid:  true,
		},
	})
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

func (h AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	type CreateUserRequest struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}

	if !h.AuthService.ValidatePassword(req.Password) {
		WriteBadRequest(w)
		return
	}
	req.Email, err = validateEmail(req.Email)
	if err != nil {
		WriteBadRequest(w)
		return
	}

	hash, err := h.AuthService.HashPassword(req.Password)
	if err != nil {
		WriteInternalError(w)
		return
	}

	user, err := h.DB.CreateUser(r.Context(),
		database.CreateUserParams{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hash,
		})
	if err != nil {
		WriteBadRequest(w)
		return
	}

	accessToken, err := h.AuthService.MakeAccessToken(user.ID, string(user.Role))
	if err != nil {
		WriteInternalError(w)
		return
	}
	rawRefreshToken, err := h.issueRefreshToken(r.Context(), user.ID, r)
	if err != nil {
		WriteInternalError(w)
		return
	}

	WriteJSON(w, http.StatusCreated, UserResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	})
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	user, err := h.DB.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err != nil {
		WriteUnauthorized(w)
		return
	}
	match, err := h.AuthService.CompareHashPassword(req.Password, user.PasswordHash)
	if err != nil {
		WriteInternalError(w)
		return
	}
	if !match {
		WriteUnauthorized(w)
		return
	}
	accessToken, err := h.AuthService.MakeAccessToken(user.ID, string(user.Role))
	if err != nil {
		WriteInternalError(w)
		return
	}
	refreshToken, err := h.issueRefreshToken(r.Context(), user.ID, r)
	if err != nil {
		WriteInternalError(w)
		return
	}

	WriteJSON(w, http.StatusOK, UserResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	meta, ok := r.Context().Value(refreshCtxKey).(refreshMeta)
	if !ok {
		WriteInternalError(w)
		return
	}

	user, err := h.DB.GetUserByID(r.Context(), meta.UserID)
	if err != nil {
		WriteInternalError(w)
		return
	}

	accessToken, err := h.AuthService.MakeAccessToken(user.ID, string(user.Role))
	if err != nil {
		WriteInternalError(w)
		return
	}
	newRefreshToken, err := h.issueRefreshToken(r.Context(), meta.UserID, r)
	if err != nil {
		WriteInternalError(w)
		return
	}

	type RefreshResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	WriteJSON(w, http.StatusCreated, RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	})
}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		WriteUnauthorized(w)
		return
	}
	err := h.DB.RevokeAllUserRefreshTokens(r.Context(), userID)
	if err != nil {
		WriteInternalError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type UserIDKey struct{}

var userIDKey = UserIDKey{}

type RoleKey struct{}

var roleKey = RoleKey{}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(roleKey).(string)
			if !ok {
				WriteForbidden(w)
				return
			}
			if _, ok := allowed[role]; !ok {
				WriteForbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (h AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := h.AuthService.GetBearerToken(r.Header)
		if err != nil {
			WriteUnauthorized(w)
			return
		}
		userID, role, err := h.AuthService.ValidateAccessToken(token)
		if err != nil {
			WriteUnauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, roleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h AuthHandler) RefreshMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := h.AuthService.GetBearerToken(r.Header)
		if err != nil {
			WriteUnauthorized(w)
			return
		}
		hash := h.AuthService.HashRefreshToken(token)
		refreshToken, err := h.DB.GetRefreshTokenByHash(r.Context(), hash)
		if err != nil {
			WriteUnauthorized(w)
			return
		}
		if refreshToken.RevokedAt.Valid {
			WriteUnauthorized(w)
			return
		}
		if time.Now().After(refreshToken.ExpiresAt) {
			WriteUnauthorized(w)
			return
		}
		err = h.DB.RevokeRefreshToken(r.Context(), hash)
		if err != nil {
			WriteInternalError(w)
			return
		}
		ctx := context.WithValue(r.Context(), refreshCtxKey, refreshMeta{
			TokenHash: hash,
			UserID:    refreshToken.UserID,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/service"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type AuthHandler struct {
	queries        *db.Queries
	logger         *slog.Logger
	adminEmail     string
	adminPassword  string
	hasher         service.Argon2Hasher
	customerIssuer service.JWTIssuer
	adminIssuer    service.JWTIssuer
	limiter        *loginLimiter
}

func NewAuthHandler(queries *db.Queries, logger *slog.Logger, customerJWTSecret, adminJWTSecret, adminEmail, adminPassword string) AuthHandler {
	return AuthHandler{
		queries:        queries,
		logger:         logger,
		adminEmail:     strings.TrimSpace(strings.ToLower(adminEmail)),
		adminPassword:  adminPassword,
		hasher:         service.Argon2Hasher{},
		customerIssuer: service.JWTIssuer{Secret: customerJWTSecret},
		adminIssuer:    service.JWTIssuer{Secret: adminJWTSecret},
		limiter:        newLoginLimiter(10, 15*time.Minute),
	}
}

type registerRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (r *registerRequest) Validate() (errors map[string]string) {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	if !required(r.Name) {
		errors["name"] = "name is required"
	}
	if !strings.Contains(r.Email, "@") {
		errors["email"] = "valid email is required"
	}
	if len(r.Password) < 8 {
		errors["password"] = "password must be at least 8 characters"
	}
	if r.ConfirmPassword != "" && r.Password != r.ConfirmPassword {
		errors["confirmPassword"] = "passwords do not match"
	}
	return errors
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	limitKey := remoteLimitKey(r)
	if !h.limiter.allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	var req registerRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	validationErrs := req.Validate()
	if len(validationErrs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": validationErrs})
		return
	}

	hash, err := h.hasher.Hash(req.Password)
	if err != nil {
		h.logger.Error("hash customer password", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	customer, err := h.queries.CreateCustomerWithPassword(r.Context(), db.CreateCustomerWithPasswordParams{
		Name:         strings.TrimSpace(req.Name),
		Email:        req.Email,
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: hash,
	})
	if err != nil {
		if isSQLiteConstraintError(err) {
			writeError(w, http.StatusConflict, "email already in use")
			return
		}
		h.logger.Error("create customer", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	token, err := h.customerIssuer.IssueCustomerToken(customer.ID)
	if err != nil {
		h.logger.Error("issue customer jwt", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, tokenResponse{Token: token})
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	limitKey := remoteLimitKey(r)
	if !h.limiter.allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	var req loginRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	customer, err := h.queries.GetCustomerByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.limiter.recordFailure(limitKey)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.logger.Error("get customer by email", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.hasher.VerifyPassword(customer.PasswordHash, req.Password); err != nil {
		h.limiter.recordFailure(limitKey)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	h.limiter.reset(limitKey)

	token, err := h.customerIssuer.IssueCustomerToken(customer.ID)
	if err != nil {
		h.logger.Error("issue customer jwt", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (h AuthHandler) AdminLogin(w http.ResponseWriter, r *http.Request) {
	limitKey := remoteLimitKey(r)
	if !h.limiter.allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	var req loginRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if strings.TrimSpace(strings.ToLower(req.Email)) != h.adminEmail || !h.adminPasswordMatches(req.Password) {
		h.limiter.recordFailure(limitKey)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	h.limiter.reset(limitKey)

	token, err := h.adminIssuer.IssueAdminToken()
	if err != nil {
		h.logger.Error("issue admin jwt", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func remoteLimitKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (h AuthHandler) adminPasswordMatches(password string) bool {
	if strings.HasPrefix(h.adminPassword, "$2a$") || strings.HasPrefix(h.adminPassword, "$2b$") || strings.HasPrefix(h.adminPassword, "$2y$") {
		return h.hasher.VerifyPassword(h.adminPassword, password) == nil
	}
	return password == h.adminPassword
}

type loginLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	attempts map[string]loginAttempt
}

type loginAttempt struct {
	count     int
	expiresAt time.Time
}

func newLoginLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{
		limit:    limit,
		window:   window,
		attempts: map[string]loginAttempt{},
	}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt := l.attempts[key]
	if time.Now().After(attempt.expiresAt) {
		delete(l.attempts, key)
		return true
	}
	return attempt.count < l.limit
}

func (l *loginLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	attempt := l.attempts[key]
	if now.After(attempt.expiresAt) {
		attempt = loginAttempt{expiresAt: now.Add(l.window)}
	}
	attempt.count++
	l.attempts[key] = attempt
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func isSQLiteConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT
	}
	return false
}

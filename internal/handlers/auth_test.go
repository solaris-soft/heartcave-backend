package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/solaris-soft/heartcave-backend/internal/services"
	"github.com/solaris-soft/heartcave-backend/internal/testutils"
)

const testJWTSecret = "this-is-a-very-long-secret-key-for-testing-123"

func setupAuthTest(t *testing.T) (*sql.DB, database.Querier, services.AuthService, func()) {
	db, cleanup := testutils.SetupTestDB(t)
	queries := database.New(db)
	authService := services.NewAuthService(testJWTSecret)
	return db, queries, authService, cleanup
}

func TestCreateUser(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "creates user successfully",
			body:       `{"name":"Test User","email":"test@example.com","password":"password123"}`,
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "fails with short password",
			body:       `{"name":"Test User","email":"test2@example.com","password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid email",
			body:       `{"name":"Test User","email":"not-an-email","password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with missing fields",
			body:       `{"name":"Test User"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with duplicate email",
			body:       `{"name":"Test User","email":"test@example.com","password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid JSON",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateUser(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp UserResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID == uuid.Nil {
					t.Error("expected non-nil user ID")
				}
				if resp.AccessToken == "" {
					t.Error("expected access token")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh token")
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	// Create a user first
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Login Test",
		Email:        "login@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "login with valid credentials",
			body:       `{"email":"login@example.com","password":"password123"}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "login with wrong password",
			body:       `{"email":"login@example.com","password":"wrongpassword"}`,
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "login with non-existent email",
			body:       `{"email":"nonexistent@example.com","password":"password123"}`,
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "login with invalid JSON",
			body:       `{"email":`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp UserResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != user.ID {
					t.Errorf("user ID = %v, want %v", resp.ID, user.ID)
				}
				if resp.AccessToken == "" {
					t.Error("expected access token")
				}
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	// Create a user
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Refresh Test",
		Email:        "refresh@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Issue a refresh token
	rawToken, _ := authService.NewRefreshToken()
	tokenHash := authService.HashRefreshToken(rawToken)
	_, err = queries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "refresh with valid token",
			token:      rawToken,
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "refresh with invalid token",
			token:      "invalid-token",
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "refresh with no token",
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-create token for valid case since it gets revoked
			if !tt.wantErr {
				rawToken, _ = authService.NewRefreshToken()
				tokenHash = authService.HashRefreshToken(rawToken)
				_, err = queries.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
					UserID:    user.ID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				})
				if err != nil {
					t.Fatalf("failed to create refresh token: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			// Use chi router to test through middleware
			r := chi.NewRouter()
			r.With(handler.RefreshMiddleware).Post("/refresh", handler.Refresh)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr {
				var resp map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp["access_token"] == "" {
					t.Error("expected access token")
				}
				if resp["refresh_token"] == "" {
					t.Error("expected refresh token")
				}
			}
		})
	}
}

func TestLogout(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	// Create a user
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Logout Test",
		Email:        "logout@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create an access token
	accessToken, _ := authService.MakeAccessToken(user.ID, string(database.UserRoleCustomer))

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "logout with valid token",
			token:      accessToken,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "logout without token",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.With(handler.AuthMiddleware).Post("/logout", handler.Logout)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	_, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	// Create a user
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")
	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Update Test",
		Email:        "update@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	accessToken, _ := authService.MakeAccessToken(user.ID, string(database.UserRoleCustomer))

	tests := []struct {
		name       string
		body       string
		wantStatus int
		checkEmail string
		wantErr    bool
	}{
		{
			name:       "update email",
			body:       `{"email":"updated@example.com"}`,
			wantStatus: http.StatusOK,
			checkEmail: "updated@example.com",
			wantErr:    false,
		},
		{
			name:       "update password",
			body:       `{"password":"newpassword123"}`,
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "update both email and password",
			body:       `{"email":"both@example.com","password":"bothpassword123"}`,
			wantStatus: http.StatusOK,
			checkEmail: "both@example.com",
			wantErr:    false,
		},
		{
			name:       "fails with empty body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with invalid email",
			body:       `{"email":"not-an-email"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails with short password",
			body:       `{"password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "fails without auth",
			body:       `{"email":"noauth@example.com"}`,
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if !tt.wantErr || tt.name != "fails without auth" {
				req.Header.Set("Authorization", "Bearer "+accessToken)
			}
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.With(handler.AuthMiddleware).Put("/users", handler.UpdateUser)
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !tt.wantErr && tt.checkEmail != "" {
				var resp UserResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if !strings.EqualFold(resp.Email, tt.checkEmail) {
					t.Errorf("email = %q, want %q", resp.Email, tt.checkEmail)
				}
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	db, queries, authService, cleanup := setupAuthTest(t)
	defer cleanup()

	handler := NewAuthHandler(queries, authService)

	// Create admin and customer users
	ctx := context.Background()
	hash, _ := authService.HashPassword("password123")

	admin, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Admin",
		Email:        "admin@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	// Update role to admin via raw SQL since CreateUser doesn't accept role
	_, err = db.ExecContext(ctx, "UPDATE users SET role = 'admin' WHERE id = $1", admin.ID)
	if err != nil {
		t.Fatalf("failed to update admin role: %v", err)
	}

	customer, err := queries.CreateUser(ctx, database.CreateUserParams{
		Name:         "Customer",
		Email:        "customer@example.com",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

	adminToken, _ := authService.MakeAccessToken(admin.ID, string(database.UserRoleAdmin))
	customerToken, _ := authService.MakeAccessToken(customer.ID, string(database.UserRoleCustomer))

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "admin can access admin route",
			token:      adminToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "customer cannot access admin route",
			token:      customerToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no token is unauthorized",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()

			r := chi.NewRouter()
			r.With(handler.AuthMiddleware, RequireRole("admin")).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			r.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

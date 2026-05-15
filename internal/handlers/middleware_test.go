package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/solaris-soft/heartcave-backend/internal/service"
)

func TestCustomerRequiredAcceptsCustomerToken(t *testing.T) {
	issuer := service.JWTIssuer{Secret: "test-secret"}
	token, err := issuer.IssueCustomerToken(7)
	if err != nil {
		t.Fatalf("IssueCustomerToken returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	CustomerRequired(issuer.Secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customerID, ok := CustomerID(r.Context())
		if !ok || customerID != 7 {
			t.Fatalf("customer id = %d, %v; want 7, true", customerID, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAdminRequiredRejectsCustomerToken(t *testing.T) {
	issuer := service.JWTIssuer{Secret: "test-secret"}
	token, err := issuer.IssueCustomerToken(7)
	if err != nil {
		t.Fatalf("IssueCustomerToken returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	AdminRequired(issuer.Secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("admin handler should not run for customer token")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

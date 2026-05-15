package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	migrations "github.com/solaris-soft/heartcave-backend/db"
	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

func TestAvailabilityHandlerWithMigratedDB(t *testing.T) {
	database, queries := newContainerBackedSQLite(t)
	loc := time.UTC

	serviceRow := seedService(t, queries, "Consultation", 12000, 60)
	date := time.Date(2026, 5, 14, 0, 0, 0, 0, loc)
	_, err := queries.CreateScheduleEntry(context.Background(), db.CreateScheduleEntryParams{
		DayOfWeek:   int64(date.Weekday()),
		StartTime:   "09:00",
		EndTime:     "12:00",
		SlotMinutes: 60,
	})
	if err != nil {
		t.Fatalf("CreateScheduleEntry returned error: %v", err)
	}
	_, err = queries.CreateBooking(context.Background(), db.CreateBookingParams{
		CustomerID:    seedCustomer(t, queries),
		ServiceID:     serviceRow.ID,
		StartTime:     "2026-05-14T10:00:00",
		EndTime:       "2026-05-14T11:00:00",
		Intentions:    "existing booking",
		Status:        "pending",
		PaymentStatus: "unpaid",
	})
	if err != nil {
		t.Fatalf("CreateBooking returned error: %v", err)
	}

	handler := NewAvailabilityHandler(queries, slog.New(slog.NewTextHandler(os.Stderr, nil)), loc)
	req := httptest.NewRequest(http.MethodGet, "/availability?date=2026-05-14&service_id=1", nil)
	rec := httptest.NewRecorder()

	handler.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var slots []service.TimeSlot
	if err := json.NewDecoder(rec.Body).Decode(&slots); err != nil {
		t.Fatalf("decode slots: %v", err)
	}
	want := []service.TimeSlot{
		{Start: "2026-05-14T09:00:00", End: "2026-05-14T10:00:00"},
		{Start: "2026-05-14T11:00:00", End: "2026-05-14T12:00:00"},
	}
	if len(slots) != len(want) {
		t.Fatalf("slots = %#v, want %#v", slots, want)
	}
	for i := range want {
		if slots[i] != want[i] {
			t.Fatalf("slots[%d] = %#v, want %#v", i, slots[i], want[i])
		}
	}

	database.Close()
}

func TestBookingReservationRequiresScheduleAndPreventsOverlap(t *testing.T) {
	database, queries := newContainerBackedSQLite(t)
	loc := time.UTC
	customerID := seedCustomer(t, queries)
	serviceRow := seedService(t, queries, "Reading", 9000, 60)
	date := time.Date(2026, 5, 14, 0, 0, 0, 0, loc)
	_, err := queries.CreateScheduleEntry(context.Background(), db.CreateScheduleEntryParams{
		DayOfWeek:   int64(date.Weekday()),
		StartTime:   "09:00",
		EndTime:     "12:00",
		SlotMinutes: 60,
	})
	if err != nil {
		t.Fatalf("CreateScheduleEntry returned error: %v", err)
	}

	handler := NewBookingsHandler(database, queries, slog.New(slog.NewTextHandler(os.Stderr, nil)), loc, "sk_test_fake", "whsec_fake", "aud", "http://localhost:4321")
	insideStart := time.Date(2026, 5, 14, 9, 0, 0, 0, loc)
	insideEnd := insideStart.Add(time.Hour)
	if _, err := handler.reserveBooking(context.Background(), customerID, serviceRow, insideStart, insideEnd, "inside"); err != nil {
		t.Fatalf("reserveBooking inside schedule returned error: %v", err)
	}

	if _, err := handler.reserveBooking(context.Background(), customerID, serviceRow, insideStart, insideEnd, "overlap"); err != errBookingUnavailable {
		t.Fatalf("reserveBooking overlap error = %v, want errBookingUnavailable", err)
	}

	outsideStart := time.Date(2026, 5, 14, 13, 0, 0, 0, loc)
	outsideEnd := outsideStart.Add(time.Hour)
	if _, err := handler.reserveBooking(context.Background(), customerID, serviceRow, outsideStart, outsideEnd, "outside"); err != errBookingUnavailable {
		t.Fatalf("reserveBooking outside schedule error = %v, want errBookingUnavailable", err)
	}

	database.Close()
}

func TestBookingResponsesDoNotExposeStripeIDs(t *testing.T) {
	database, queries := newContainerBackedSQLite(t)
	customerID := seedCustomer(t, queries)
	serviceRow := seedService(t, queries, "Session", 8000, 60)
	booking, err := queries.CreateBooking(context.Background(), db.CreateBookingParams{
		CustomerID:    customerID,
		ServiceID:     serviceRow.ID,
		StartTime:     "2026-05-14T09:00:00",
		EndTime:       "2026-05-14T10:00:00",
		Intentions:    "private",
		Status:        "confirmed",
		PaymentStatus: "paid",
	})
	if err != nil {
		t.Fatalf("CreateBooking returned error: %v", err)
	}
	if err := queries.SetBookingCheckoutSession(context.Background(), db.SetBookingCheckoutSessionParams{
		ID:                      booking.ID,
		StripeCheckoutSessionID: "cs_secret_should_not_leak",
	}); err != nil {
		t.Fatalf("SetBookingCheckoutSession returned error: %v", err)
	}

	handler := NewBookingsHandler(database, queries, slog.New(slog.NewTextHandler(os.Stderr, nil)), time.UTC, "sk_test_fake", "whsec_fake", "aud", "http://localhost:4321")
	req := httptest.NewRequest(http.MethodGet, "/admin/bookings", nil)
	rec := httptest.NewRecorder()

	handler.AdminList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if body := rec.Body.String(); json.Valid([]byte(body)) && strings.Contains(body, "cs_secret_should_not_leak") {
		t.Fatalf("admin booking response leaked checkout session id: %s", body)
	}

	database.Close()
}

func newContainerBackedSQLite(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	dir := t.TempDir()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      "alpine:3.20",
			Cmd:        []string{"sh", "-c", "touch /db/.ready && sleep 600"},
			Mounts:     testcontainers.Mounts(testcontainers.BindMount(dir, "/db")),
			WaitingFor: wait.ForFile("/db/.ready"),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("testcontainers unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	database, err := sql.Open("sqlite", filepath.Join(dir, "heartcave-test.db"))
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	database.SetMaxOpenConns(1)
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	goose.SetBaseFS(migrations.Migrations)
	if err := goose.SetDialect("sqlite"); err != nil {
		t.Fatalf("goose.SetDialect returned error: %v", err)
	}
	if err := goose.Up(database, "migrations"); err != nil {
		t.Fatalf("goose.Up returned error: %v", err)
	}

	return database, db.New(database)
}

func seedCustomer(t *testing.T, queries *db.Queries) int64 {
	t.Helper()
	customer, err := queries.CreateCustomerWithPassword(context.Background(), db.CreateCustomerWithPasswordParams{
		Name:         "Test Customer",
		Email:        "customer@example.com",
		Phone:        "0400000000",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateCustomerWithPassword returned error: %v", err)
	}
	return customer.ID
}

func seedService(t *testing.T, queries *db.Queries, name string, price int64, minutes int64) db.Service {
	t.Helper()
	serviceRow, err := queries.CreateService(context.Background(), db.CreateServiceParams{
		Name:        name,
		Description: sql.NullString{String: "A test service", Valid: true},
		Price:       price,
		Minutes:     minutes,
	})
	if err != nil {
		t.Fatalf("CreateService returned error: %v", err)
	}
	return serviceRow
}

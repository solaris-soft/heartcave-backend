package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
	"github.com/solaris-soft/heartcave-backend/internal/service"
	"github.com/stripe/stripe-go/v85"
	checkoutsession "github.com/stripe/stripe-go/v85/checkout/session"
	"github.com/stripe/stripe-go/v85/webhook"
)

type BookingsHandler struct {
	database            *sql.DB
	queries             *db.Queries
	logger              *slog.Logger
	location            *time.Location
	stripeSecretKey     string
	stripeWebhookSecret string
	stripeCurrency      string
	frontendURL         string
}

func NewBookingsHandler(database *sql.DB, queries *db.Queries, logger *slog.Logger, location *time.Location, stripeSecretKey, stripeWebhookSecret, stripeCurrency, frontendURL string) BookingsHandler {
	return BookingsHandler{
		database:            database,
		queries:             queries,
		logger:              logger,
		location:            location,
		stripeSecretKey:     stripeSecretKey,
		stripeWebhookSecret: stripeWebhookSecret,
		stripeCurrency:      strings.ToLower(stripeCurrency),
		frontendURL:         strings.TrimRight(frontendURL, "/"),
	}
}

type createBookingRequest struct {
	ServiceID  int64  `json:"service_id"`
	StartTime  string `json:"start_time"`
	Intentions string `json:"intentions"`
}

type checkoutResponse struct {
	BookingID     int64  `json:"booking_id"`
	CheckoutURL   string `json:"checkout_url"`
	CheckoutID    string `json:"checkout_session_id"`
	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
}

type bookingResponse struct {
	ID             int64  `json:"id"`
	CustomerID     int64  `json:"customer_id,omitempty"`
	ServiceID      int64  `json:"service_id"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Intentions     string `json:"intentions"`
	Status         string `json:"status"`
	PaymentStatus  string `json:"payment_status"`
	ServiceName    string `json:"service_name"`
	ServicePrice   int64  `json:"service_price"`
	ServiceMinutes int64  `json:"service_minutes"`
	CustomerName   string `json:"customer_name,omitempty"`
	CustomerEmail  string `json:"customer_email,omitempty"`
	CustomerPhone  string `json:"customer_phone,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

func (h BookingsHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	customerID, ok := CustomerID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	bookings, err := h.queries.ListBookingsByCustomer(r.Context(), customerID)
	if err != nil {
		h.logger.Error("list customer bookings", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if bookings == nil {
		bookings = []db.ListBookingsByCustomerRow{}
	}
	response := make([]bookingResponse, len(bookings))
	for i, booking := range bookings {
		response[i] = bookingResponse{
			ID:             booking.ID,
			ServiceID:      booking.ServiceID,
			StartTime:      booking.StartTime,
			EndTime:        booking.EndTime,
			Intentions:     booking.Intentions,
			Status:         booking.Status,
			PaymentStatus:  booking.PaymentStatus,
			ServiceName:    booking.ServiceName,
			ServicePrice:   booking.ServicePrice,
			ServiceMinutes: booking.ServiceMinutes,
			CreatedAt:      booking.CreatedAt,
			UpdatedAt:      booking.UpdatedAt,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h BookingsHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.queries.ListBookings(r.Context())
	if err != nil {
		h.logger.Error("list bookings", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if bookings == nil {
		bookings = []db.ListBookingsRow{}
	}
	response := make([]bookingResponse, len(bookings))
	for i, booking := range bookings {
		response[i] = bookingResponse{
			ID:             booking.ID,
			CustomerID:     booking.CustomerID,
			ServiceID:      booking.ServiceID,
			StartTime:      booking.StartTime,
			EndTime:        booking.EndTime,
			Intentions:     booking.Intentions,
			Status:         booking.Status,
			PaymentStatus:  booking.PaymentStatus,
			ServiceName:    booking.ServiceName,
			ServicePrice:   booking.ServicePrice,
			ServiceMinutes: booking.ServiceMinutes,
			CustomerName:   booking.CustomerName,
			CustomerEmail:  booking.CustomerEmail,
			CustomerPhone:  booking.CustomerPhone,
			CreatedAt:      booking.CreatedAt,
			UpdatedAt:      booking.UpdatedAt,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h BookingsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.stripeSecretKey == "" {
		writeError(w, http.StatusServiceUnavailable, "stripe is not configured")
		return
	}

	customerID, ok := CustomerID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createBookingRequest
	if !decodeJSON(r, &req) {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ServiceID <= 0 || strings.TrimSpace(req.StartTime) == "" {
		writeError(w, http.StatusBadRequest, "service_id and start_time are required")
		return
	}

	serviceRow, err := h.queries.GetServiceByID(r.Context(), req.ServiceID)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	start, err := localDateTime(req.StartTime, h.location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "start_time must use YYYY-MM-DDTHH:MM[:SS]")
		return
	}
	end := start.Add(time.Duration(serviceRow.Minutes) * time.Minute)

	booking, err := h.reserveBooking(r.Context(), customerID, serviceRow, start, end, strings.TrimSpace(req.Intentions))
	if err != nil {
		if errors.Is(err, errBookingUnavailable) {
			writeError(w, http.StatusConflict, "booking time is unavailable")
			return
		}
		h.logger.Error("reserve booking", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	customer, err := h.queries.GetCustomerByID(r.Context(), customerID)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}

	checkout, err := h.createCheckoutSession(booking, customer, serviceRow)
	if err != nil {
		h.logger.Error("create stripe checkout session", "err", err)
		_ = h.queries.UpdateBookingStatus(r.Context(), db.UpdateBookingStatusParams{
			Status: "cancelled",
			ID:     booking.ID,
		})
		writeError(w, http.StatusBadGateway, "unable to create checkout session")
		return
	}

	if err := h.queries.SetBookingCheckoutSession(r.Context(), db.SetBookingCheckoutSessionParams{
		StripeCheckoutSessionID: checkout.ID,
		ID:                      booking.ID,
	}); err != nil {
		h.logger.Error("set booking checkout session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, checkoutResponse{
		BookingID:     booking.ID,
		CheckoutURL:   checkout.URL,
		CheckoutID:    checkout.ID,
		Status:        booking.Status,
		PaymentStatus: booking.PaymentStatus,
	})
}

func (h BookingsHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.stripeWebhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "stripe webhook is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "unable to read request body")
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), h.stripeWebhookSecret)
	if err != nil {
		h.logger.Warn("stripe webhook signature rejected", "err", err)
		writeError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	if event.ID != "" {
		if _, err := h.queries.RecordStripeEvent(r.Context(), db.RecordStripeEventParams{ID: event.ID, EventType: string(event.Type)}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.logger.Error("record stripe event", "err", err, "event_id", event.ID)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	if err := h.handleStripeEvent(r.Context(), event); err != nil {
		if event.ID != "" {
			_ = h.queries.DeleteStripeEvent(r.Context(), event.ID)
		}
		h.logger.Error("handle stripe event", "err", err, "event_id", event.ID, "event_type", event.Type)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if event.ID != "" {
		if err := h.queries.MarkStripeEventProcessed(r.Context(), event.ID); err != nil {
			h.logger.Error("mark stripe event processed", "err", err, "event_id", event.ID)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h BookingsHandler) handleStripeEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		var checkout stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &checkout); err != nil {
			return err
		}
		if checkout.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
			paymentIntentID := ""
			if checkout.PaymentIntent != nil {
				paymentIntentID = checkout.PaymentIntent.ID
			}
			if err := h.queries.ConfirmBookingByCheckoutSession(ctx, db.ConfirmBookingByCheckoutSessionParams{
				StripePaymentIntentID:   paymentIntentID,
				StripeCheckoutSessionID: checkout.ID,
			}); err != nil {
				return err
			}
		}
	case "checkout.session.expired":
		var checkout stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &checkout); err == nil {
			if err := h.queries.CancelBookingByCheckoutSession(ctx, checkout.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

var errBookingUnavailable = errors.New("booking unavailable")

func (h BookingsHandler) reserveBooking(ctx context.Context, customerID int64, serviceRow db.Service, start, end time.Time, intentions string) (db.Booking, error) {
	tx, err := h.database.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return db.Booking{}, err
	}
	defer tx.Rollback()

	q := h.queries.WithTx(tx)
	schedule, err := q.GetScheduleByDay(ctx, int64(start.Weekday()))
	if err != nil {
		return db.Booking{}, err
	}
	if !slotInSchedule(start, end, schedule) {
		return db.Booking{}, errBookingUnavailable
	}

	startValue := start.Format(service.LocalDateTimeLayout)
	endValue := end.Format(service.LocalDateTimeLayout)
	count, err := q.CountOverlappingBookings(ctx, db.CountOverlappingBookingsParams{
		EndTime:   endValue,
		StartTime: startValue,
	})
	if err != nil {
		return db.Booking{}, err
	}
	if count > 0 {
		return db.Booking{}, errBookingUnavailable
	}

	booking, err := q.CreateBooking(ctx, db.CreateBookingParams{
		CustomerID:    customerID,
		ServiceID:     serviceRow.ID,
		StartTime:     startValue,
		EndTime:       endValue,
		Intentions:    intentions,
		Status:        "pending",
		PaymentStatus: "unpaid",
	})
	if err != nil {
		return db.Booking{}, err
	}
	if err := tx.Commit(); err != nil {
		return db.Booking{}, err
	}
	return booking, nil
}

func slotInSchedule(start, end time.Time, schedule []db.AdminSchedule) bool {
	for _, entry := range schedule {
		windowStart, err := scheduleClock(start, entry.StartTime)
		if err != nil {
			continue
		}
		windowEnd, err := scheduleClock(start, entry.EndTime)
		if err != nil {
			continue
		}
		if !start.Before(windowStart) && !end.After(windowEnd) {
			return true
		}
	}
	return false
}

func scheduleClock(date time.Time, value string) (time.Time, error) {
	clock, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, date.Location()), nil
}

func (h BookingsHandler) createCheckoutSession(booking db.Booking, customer db.Customer, serviceRow db.Service) (*stripe.CheckoutSession, error) {
	stripe.Key = h.stripeSecretKey
	successURL := fmt.Sprintf("%s/booking/success?session_id={CHECKOUT_SESSION_ID}", h.frontendURL)
	cancelURL := fmt.Sprintf("%s/booking/cancelled?booking_id=%d", h.frontendURL, booking.ID)
	description := ""
	if serviceRow.Description.Valid {
		description = serviceRow.Description.String
	}

	mode := string(stripe.CheckoutSessionModePayment)
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(mode),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		CustomerEmail:     stripe.String(customer.Email),
		ClientReferenceID: stripe.String(strconv.FormatInt(booking.ID, 10)),
		Metadata: map[string]string{
			"booking_id":  strconv.FormatInt(booking.ID, 10),
			"customer_id": strconv.FormatInt(customer.ID, 10),
			"service_id":  strconv.FormatInt(serviceRow.ID, 10),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(h.stripeCurrency),
					UnitAmount: stripe.Int64(serviceRow.Price),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(serviceRow.Name),
						Description: stripe.String(description),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
	}

	return checkoutsession.New(params)
}

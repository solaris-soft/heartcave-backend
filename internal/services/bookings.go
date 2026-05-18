package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/stripe/stripe-go/v85"
)

var ErrTimeslotUnavailable = errors.New("the requested timeslot is unavailable")

type BookingService interface {
	CreateBooking(
		ctx context.Context,
		customerID uuid.UUID,
		serviceID uuid.UUID,
		startsAt time.Time,
		customerNotes string,
		successURL string,
		cancelURL string,
	) (CreateBookingResult, error)
}

type bookingService struct {
	DB       database.Querier
	Stripe   *stripe.Client
	Currency string
}

func NewBookingService(db database.Querier, stripeClient *stripe.Client, currency string) BookingService {
	return bookingService{
		DB:       db,
		Stripe:   stripeClient,
		Currency: currency,
	}
}

type CreateBookingResult struct {
	Booking        database.Booking
	CheckoutURL    string
}

func (s bookingService) CreateBooking(
	ctx context.Context,
	customerID uuid.UUID,
	serviceID uuid.UUID,
	startsAt time.Time,
	customerNotes string,
	successURL string,
	cancelURL string,
) (CreateBookingResult, error) {
	svc, err := s.DB.GetServiceByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CreateBookingResult{}, fmt.Errorf("service not found: %w", err)
		}
		return CreateBookingResult{}, fmt.Errorf("fetching service: %w", err)
	}

	endsAt := startsAt.Add(time.Duration(svc.SessionMinutes) * time.Minute)

	overlapping, err := s.DB.GetOverlappingBookings(ctx, database.GetOverlappingBookingsParams{
		StartsAt: startsAt,
		EndsAt:   endsAt,
	})
	if err != nil {
		return CreateBookingResult{}, fmt.Errorf("checking availability: %w", err)
	}
	if len(overlapping) > 0 {
		return CreateBookingResult{}, ErrTimeslotUnavailable
	}

	unitAmount, err := priceToCents(svc.Price)
	if err != nil {
		return CreateBookingResult{}, fmt.Errorf("parsing service price: %w", err)
	}

	checkoutParams := &stripe.CheckoutSessionCreateParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String(s.Currency),
					UnitAmount: stripe.Int64(unitAmount),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name: stripe.String(svc.Name),
					},
				},
			},
		},
	}

	session, err := s.Stripe.V1CheckoutSessions.Create(ctx, checkoutParams)
	if err != nil {
		return CreateBookingResult{}, fmt.Errorf("creating stripe checkout session: %w", err)
	}

	booking, err := s.DB.CreateBooking(ctx, database.CreateBookingParams{
		CustomerID: customerID,
		ServiceID:  serviceID,
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		Status:     "pending",
		ServiceName:  svc.Name,
		ServicePrice: svc.Price,
		CustomerNotes: sql.NullString{
			String: customerNotes,
			Valid:  customerNotes != "",
		},
		StripeCheckoutSessionID: sql.NullString{
			String: session.ID,
			Valid:  true,
		},
	})
	if err != nil {
		if isExclusionViolation(err) {
			return CreateBookingResult{}, ErrTimeslotUnavailable
		}
		return CreateBookingResult{}, fmt.Errorf("creating booking: %w", err)
	}

	return CreateBookingResult{
		Booking:     booking,
		CheckoutURL: session.URL,
	}, nil
}

func priceToCents(priceStr string) (int64, error) {
	dollars, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, err
	}
	return int64(dollars * 100), nil
}

func isExclusionViolation(err error) bool {
	return strings.Contains(err.Error(), "no_overlapping_active_bookings")
}

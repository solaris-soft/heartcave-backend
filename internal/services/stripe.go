package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/solaris-soft/heartcave-backend/internal/database"
	"github.com/stripe/stripe-go/v85"
)

type StripeWebhookService interface {
	ProcessEvent(ctx context.Context, event stripe.Event) error
}

type stripeWebhookService struct {
	DB database.Querier
}

func NewStripeWebhookService(db database.Querier) StripeWebhookService {
	return stripeWebhookService{db}
}

func (s stripeWebhookService) ProcessEvent(
	ctx context.Context,
	event stripe.Event,
) error {
	alreadyProcessed, err := s.eventAlreadyProcessed(ctx, event.ID)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		return nil
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentFailed(ctx, event)
	default:
		return s.markProcessed(ctx, event)
	}
}

func (s stripeWebhookService) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("unmarshalling checkout session: %w", err)
	}

	var paymentIntentID sql.NullString
	if session.PaymentIntent != nil {
		paymentIntentID = sql.NullString{
			String: session.PaymentIntent.ID,
			Valid:  true,
		}
	}

	_, err := s.DB.UpdateBookingToPaid(ctx, database.UpdateBookingToPaidParams{
		StripeCheckoutSessionID: sql.NullString{
			String: session.ID,
			Valid:  true,
		},
		StripePaymentIntentID: paymentIntentID,
	})
	if err != nil {
		return fmt.Errorf("updating booking to paid: %w", err)
	}

	return s.markProcessed(ctx, event)
}

func (s stripeWebhookService) handlePaymentFailed(ctx context.Context, event stripe.Event) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return fmt.Errorf("unmarshalling payment intent: %w", err)
	}

	_, err := s.DB.UpdateBookingToFailed(ctx, sql.NullString{
		String: pi.ID,
		Valid:  true,
	})
	if err != nil {
		return fmt.Errorf("updating booking to failed: %w", err)
	}

	return s.markProcessed(ctx, event)
}

func (s stripeWebhookService) markProcessed(ctx context.Context, event stripe.Event) error {
	return s.DB.CreateProcessedStripeEvent(ctx, event.ID)
}

func (s stripeWebhookService) eventAlreadyProcessed(ctx context.Context, eventID string) (bool, error) {
	_, err := s.DB.GetStripeEventByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

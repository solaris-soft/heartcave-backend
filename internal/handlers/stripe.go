package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/solaris-soft/heartcave-backend/internal/services"
	"github.com/stripe/stripe-go/v85/webhook"
)

type StripeWebhookHandler struct {
	webhookSecret string
	Service       services.StripeWebhookService
}

func NewStripeWebhookHandler(webhookSecret string, service services.StripeWebhookService) StripeWebhookHandler {
	if len(webhookSecret) == 0 {
		log.Fatal("Must have a Stripe API key")
	}
	return StripeWebhookHandler{
		webhookSecret: webhookSecret,
		Service:       service,
	}
}

func (h StripeWebhookHandler) HandleStripeWebHook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		WriteBadRequest(w)
		return
	}

	signature := r.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(
		payload,
		signature,
		h.webhookSecret,
	)
	if err != nil {
		WriteBadRequest(w)
		return
	}
	err = h.Service.ProcessEvent(r.Context(), event)
	if err != nil {
		WriteInternalError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

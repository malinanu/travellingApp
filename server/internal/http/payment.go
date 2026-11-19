package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
)

// handleListPaymentMethods returns a user's saved payment methods.
func (s *Server) handleListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	methods, err := s.Store.PaymentMethodsByUser(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	if methods == nil {
		methods = []domain.PaymentMethod{}
	}
	writeJSON(w, http.StatusOK, methods)
}

// handleCreatePaymentMethod used to accept a full card number. Card
// details must never reach Spotto's servers: travellers now pay on the
// payment gateway's hosted checkout, and saved cards will only come back
// as gateway tokens. Kept as 410 so old app builds get a clear answer.
func (s *Server) handleCreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	writeErr(w, http.StatusGone, "Cards are no longer saved in Spotto. You pay securely at checkout when you book.")
}

// handleDeletePaymentMethod deletes a saved card.
func (s *Server) handleDeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := s.Store.DeletePaymentMethod(r.Context(), c.Subject, id); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleSetDefaultPaymentMethod sets a card as default.
func (s *Server) handleSetDefaultPaymentMethod(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := s.Store.SetDefaultPaymentMethod(r.Context(), c.Subject, id); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "default_updated"})
}

// handleVendorSubscriptionCheckout used to record a "succeeded" payment
// without charging anything. Venues and hosts have no paid subscription:
// Spotto earns a commission on bookings instead.
func (s *Server) handleVendorSubscriptionCheckout(w http.ResponseWriter, r *http.Request) {
	writeErr(w, http.StatusGone, "Spotto has no paid vendor subscription. You only pay a commission on bookings.")
}

package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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

// handleCreatePaymentMethod saves a new payment card.
func (s *Server) handleCreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		CardNumber string `json:"cardNumber"`
		ExpMonth   int    `json:"expMonth"`
		ExpYear    int    `json:"expYear"`
		HolderName string `json:"holderName"`
		IsDefault  bool   `json:"isDefault"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	num := strings.ReplaceAll(req.CardNumber, " ", "")
	if len(num) < 13 || len(num) > 19 {
		writeErr(w, http.StatusBadRequest, "invalid card number")
		return
	}
	if req.ExpMonth < 1 || req.ExpMonth > 12 {
		writeErr(w, http.StatusBadRequest, "invalid expiration month")
		return
	}
	if req.ExpYear < time.Now().Year() {
		writeErr(w, http.StatusBadRequest, "card is expired")
		return
	}

	brand := detectCardBrand(num)
	last4 := num[len(num)-4:]

	existing, _ := s.Store.PaymentMethodsByUser(r.Context(), c.Subject)
	isDefault := req.IsDefault || len(existing) == 0

	pm := domain.PaymentMethod{
		ID:         newID("pm"),
		UserID:     c.Subject,
		Brand:      brand,
		Last4:      last4,
		ExpMonth:   req.ExpMonth,
		ExpYear:    req.ExpYear,
		HolderName: sanitizeText(req.HolderName, 100),
		IsDefault:  isDefault,
		CreatedAt:  time.Now(),
	}

	if isDefault && len(existing) > 0 {
		_ = s.Store.SetDefaultPaymentMethod(r.Context(), c.Subject, pm.ID)
	}

	if err := s.Store.CreatePaymentMethod(r.Context(), pm); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save payment method")
		return
	}

	writeJSON(w, http.StatusCreated, pm)
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

// handleVendorSubscriptionCheckout processes vendor tier upgrades.
func (s *Server) handleVendorSubscriptionCheckout(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Tier string `json:"tier"` // starter | pro | enterprise
	}
	if !readJSON(w, r, &req) {
		return
	}

	tier := strings.ToLower(strings.TrimSpace(req.Tier))
	amountCents := 0
	switch tier {
	case "pro":
		amountCents = 2900
	case "enterprise":
		amountCents = 9900
	case "starter":
		amountCents = 0
	default:
		writeErr(w, http.StatusBadRequest, "invalid subscription tier")
		return
	}

	tx := domain.PaymentTransaction{
		ID:          newID("tx"),
		UserID:      c.Subject,
		VendorID:    &v.ID,
		AmountCents: amountCents,
		Currency:    "USD",
		Status:      "succeeded",
		Description: fmt.Sprintf("Spotto Vendor %s Tier Subscription", strings.Title(tier)),
		CreatedAt:   time.Now(),
	}

	_ = s.Store.CreatePaymentTransaction(r.Context(), tx)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "succeeded",
		"tier":        tier,
		"transaction": tx,
	})
}

func detectCardBrand(number string) string {
	if strings.HasPrefix(number, "4") {
		return "Visa"
	}
	if strings.HasPrefix(number, "51") || strings.HasPrefix(number, "52") ||
		strings.HasPrefix(number, "53") || strings.HasPrefix(number, "54") || strings.HasPrefix(number, "55") {
		return "Mastercard"
	}
	if strings.HasPrefix(number, "34") || strings.HasPrefix(number, "37") {
		return "American Express"
	}
	if strings.HasPrefix(number, "6011") || strings.HasPrefix(number, "65") {
		return "Discover"
	}
	return "Card"
}

package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // traveller (default) or vendor
	// Vendor registration extras
	BusinessName string `json:"businessName,omitempty"`
	Category     string `json:"category,omitempty"`
	Town         string `json:"town,omitempty"`
	// Traveller extras
	Country     string   `json:"country,omitempty"`
	TravelStyle string   `json:"travelStyle,omitempty"`
	Interests   []string `json:"interests,omitempty"`
}

type authResp struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !readJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "email and a password of at least 6 characters are required")
		return
	}
	if _, err := s.Store.UserByEmail(r.Context(), req.Email); err == nil {
		writeErr(w, http.StatusConflict, "email already registered")
		return
	}

	role := domain.RoleTraveller
	if req.Role == string(domain.RoleVendor) {
		role = domain.RoleVendor
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	u := domain.User{
		ID:           newID("usr"),
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
		Name:         req.Name,
		Country:      req.Country,
		TravelStyle:  req.TravelStyle,
		Interests:    req.Interests,
		CreatedAt:    time.Now(),
	}

	// Vendor registration creates a pending vendor gated until admin approval.
	if role == domain.RoleVendor {
		v := domain.Vendor{
			ID:          newID("ven"),
			Name:        req.BusinessName,
			Category:    req.Category,
			Town:        req.Town,
			Owner:       req.Name,
			Status:      domain.VendorPending,
			Priority:    "normal",
			Flags:       []string{},
			Tier:        "Free",
			SubmittedAt: time.Now(),
		}
		if v.Name == "" {
			v.Name = req.Name
		}
		if err := s.Store.UpsertVendor(r.Context(), v); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		u.VendorID = &v.ID
	}

	if err := s.Store.CreateUser(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, authResp{Token: token, User: u})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	u, err := s.Store.UserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResp{Token: token, User: u})
}

// meResp includes vendor status so vendor clients can show the pending gate.
type meResp struct {
	domain.User
	VendorStatus *domain.VendorStatus `json:"vendorStatus,omitempty"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	resp := meResp{User: u}
	if u.VendorID != nil {
		if v, err := s.Store.VendorByID(r.Context(), *u.VendorID); err == nil {
			resp.VendorStatus = &v.Status
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Name        *string   `json:"name"`
		Country     *string   `json:"country"`
		TravelStyle *string   `json:"travelStyle"`
		Interests   *[]string `json:"interests"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Country != nil {
		u.Country = *req.Country
	}
	if req.TravelStyle != nil {
		u.TravelStyle = *req.TravelStyle
	}
	if req.Interests != nil {
		u.Interests = *req.Interests
	}
	if err := s.Store.UpdateUserProfile(r.Context(), u.ID, u.Name, u.Country, u.TravelStyle, u.Interests); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// requireVendor loads the caller's vendor, or writes an error and returns false.
func (s *Server) requireVendor(w http.ResponseWriter, r *http.Request) (domain.Vendor, bool) {
	c := auth.FromContext(r.Context())
	if c.VendorID == "" {
		writeErr(w, http.StatusForbidden, "no vendor account")
		return domain.Vendor{}, false
	}
	v, err := s.Store.VendorByID(r.Context(), c.VendorID)
	if err != nil {
		if err == postgres.ErrNotFound {
			writeErr(w, http.StatusForbidden, "no vendor account")
		} else {
			storeErr(w, err)
		}
		return domain.Vendor{}, false
	}
	return v, true
}

// requireApprovedVendor additionally gates on admin approval.
func (s *Server) requireApprovedVendor(w http.ResponseWriter, r *http.Request) (domain.Vendor, bool) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return v, false
	}
	if v.Status != domain.VendorApproved {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":        "vendor_not_approved",
			"vendorStatus": v.Status,
		})
		return v, false
	}
	return v, true
}

package httpapi

// Phone numbers on the vendor portal: team members (owner, managers,
// staff) verify their own number with a one-time code by WhatsApp or SMS,
// and both phone fields get non-blocking hints when a number is already in
// use elsewhere. Numbers are deliberately not unique — one person can hold a
// traveller and a business account with the same mobile — so nothing here
// ever refuses a number.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/phone"
	"github.com/spotto/server/internal/store/postgres"
	"github.com/spotto/server/internal/verifymsg"
)

const (
	phoneCodeTTL       = 10 * time.Minute
	phoneResendAfter   = 60 * time.Second
	phoneMaxAttempts   = 5
	maxPhoneQueryChars = 40
)

func hashPhoneCode(userID, code string) string {
	sum := sha256.Sum256([]byte(userID + ":" + code))
	return hex.EncodeToString(sum[:])
}

func newPhoneCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// handleMyPhoneCheck tells the Account page whether another Spotto account
// already uses the number being typed, so it can suggest a different one.
func (s *Server) handleMyPhoneCheck(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	raw := r.URL.Query().Get("phone")
	if !validateTextLength(w, "Phone", raw, maxPhoneQueryChars) {
		return
	}
	used := false
	if key := phone.Key(raw); key != "" {
		var err error
		if used, err = s.Store.PhoneUsedByOtherUser(r.Context(), key, c.Subject); err != nil {
			storeErr(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]bool{"usedByOtherAccount": used})
}

// handleVendorPhoneCheck backs the Business Profile's contact-number hint:
// a number someone on this business's team has verified as their own is
// fine; otherwise, if another business already lists it as its contact
// number, name that business. Business contact numbers are public listing
// data, so naming the other business reveals nothing private.
func (s *Server) handleVendorPhoneCheck(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("phone")
	if !validateTextLength(w, "Phone", raw, maxPhoneQueryChars) {
		return
	}
	type otherBusiness struct {
		Name string `json:"name"`
	}
	resp := struct {
		OtherBusiness *otherBusiness `json:"otherBusiness,omitempty"`
	}{}
	key := phone.Key(raw)
	if key == "" {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	ours, err := s.Store.VendorTeamHasVerifiedPhone(r.Context(), v.ID, key)
	if err != nil {
		storeErr(w, err)
		return
	}
	if !ours {
		name, found, err := s.Store.OtherVendorWithContactPhone(r.Context(), key, v.ID)
		if err != nil {
			storeErr(w, err)
			return
		}
		if found {
			resp.OtherBusiness = &otherBusiness{Name: name}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePhoneVerifyStart sends a 6-digit code to the caller's saved number
// over the channel they picked. Only a hash of the code is stored.
func (s *Server) handlePhoneVerifyStart(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Channel string `json:"channel"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Channel != verifymsg.ChannelWhatsApp && req.Channel != verifymsg.ChannelSMS {
		writeErr(w, http.StatusBadRequest, "channel must be whatsapp or sms")
		return
	}
	if phone.Key(u.Phone) == "" {
		writeErr(w, http.StatusBadRequest, "Add your phone number first.")
		return
	}
	if phone.E164(u.Phone) == "" {
		writeErr(w, http.StatusBadRequest, "Add your number with its country code (like +94 76 123 4567) to verify it.")
		return
	}
	if u.PhoneVerified {
		writeErr(w, http.StatusConflict, "This number is already verified.")
		return
	}
	if prev, err := s.Store.PhoneVerificationByUser(r.Context(), u.ID); err == nil {
		if wait := phoneResendAfter - time.Since(prev.SentAt); wait > 0 {
			secs := int(math.Ceil(wait.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":   fmt.Sprintf("Wait %d seconds before asking for another code.", secs),
				"retryIn": secs,
			})
			return
		}
	} else if !errors.Is(err, postgres.ErrNotFound) {
		storeErr(w, err)
		return
	}
	if !s.phoneVerifyLimit.allow(u.ID) {
		writeErr(w, http.StatusTooManyRequests, "Too many codes requested — try again in an hour.")
		return
	}
	s.phoneVerifyLimit.recordFailure(u.ID) // counts every send; never reset

	code, err := newPhoneCode()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Verify.Send(r.Context(), req.Channel, u.Phone, code); err != nil {
		other := "SMS"
		if req.Channel == verifymsg.ChannelSMS {
			other = "WhatsApp"
		}
		if errors.Is(err, verifymsg.ErrChannelUnavailable) {
			writeErr(w, http.StatusServiceUnavailable, fmt.Sprintf("Sending codes by %s isn't available yet — try %s.", channelLabel(req.Channel), other))
			return
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("We couldn't send the code by %s. Check your number, or try %s.", channelLabel(req.Channel), other))
		return
	}
	now := time.Now()
	if err := s.Store.UpsertPhoneVerification(r.Context(), postgres.PhoneVerification{
		UserID: u.ID, Phone: u.Phone, Channel: req.Channel,
		CodeHash: hashPhoneCode(u.ID, code), ExpiresAt: now.Add(phoneCodeTTL), SentAt: now,
	}); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channel":   req.Channel,
		"sentTo":    phone.Masked(u.Phone),
		"expiresIn": int(phoneCodeTTL.Seconds()),
		"resendIn":  int(phoneResendAfter.Seconds()),
	})
}

func channelLabel(channel string) string {
	if channel == verifymsg.ChannelWhatsApp {
		return "WhatsApp"
	}
	return "SMS"
}

// handlePhoneVerifyConfirm checks the code and marks the caller's current
// number verified.
func (s *Server) handlePhoneVerifyConfirm(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	code := strings.TrimSpace(req.Code)
	v, err := s.Store.PhoneVerificationByUser(r.Context(), u.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		writeErr(w, http.StatusBadRequest, "Request a code first.")
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	switch {
	case phone.Key(v.Phone) != phone.Key(u.Phone):
		writeErr(w, http.StatusBadRequest, "Your number changed after the code was sent — request a new code.")
		return
	case time.Now().After(v.ExpiresAt):
		writeErr(w, http.StatusBadRequest, "This code has expired — request a new one.")
		return
	case v.Attempts >= phoneMaxAttempts:
		writeErr(w, http.StatusTooManyRequests, "Too many wrong codes — request a new one.")
		return
	}
	if subtle.ConstantTimeCompare([]byte(hashPhoneCode(u.ID, code)), []byte(v.CodeHash)) != 1 {
		if err := s.Store.IncrementPhoneVerificationAttempts(r.Context(), u.ID); err != nil {
			storeErr(w, err)
			return
		}
		writeErr(w, http.StatusBadRequest, "That code isn't right.")
		return
	}
	if err := s.Store.MarkPhoneVerified(r.Context(), u.ID, v.Phone); err != nil {
		storeErr(w, err)
		return
	}
	u, err = s.Store.UserByID(r.Context(), u.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
	writeJSON(w, http.StatusOK, u)
}

// Package domain holds the entities shared by store and handlers.
// Rich nested payloads (place details, trip days, growth blobs) are kept
// as raw JSON documents whose shapes mirror the prototype's data files.
package domain

import (
	"encoding/json"
	"time"
)

type Role string

const (
	RoleTraveller Role = "traveller"
	RoleVendor    Role = "vendor"
	RoleAdmin     Role = "admin"
)

type User struct {
	ID                string          `json:"id"`
	Email             string          `json:"email"`
	PasswordHash      string          `json:"-"`
	Role              Role            `json:"role"`
	Name              string          `json:"name"`
	Country           string          `json:"country"`
	TravelStyle       string          `json:"travelStyle"`
	Interests         []string        `json:"interests"`
	VendorID          *string         `json:"vendorId,omitempty"`
	Extra             json.RawMessage `json:"-"`
	CreatedAt         time.Time       `json:"createdAt"`
	TOTPSecret        string          `json:"-"`
	TOTPPendingSecret string          `json:"-"`
	TOTPEnabled       bool            `json:"totpEnabled"`
	// TokenVersion is embedded in issued JWTs; bumping it (e.g. on 2FA
	// disable) invalidates every token issued before the bump.
	TokenVersion int `json:"-"`
	// ShareLocation is the traveller's opt-in consent for the nearby-
	// travellers feature; Lat/Lng/LocationUpdatedAt are only ever written
	// when it's true, and cleared when it's turned off.
	ShareLocation     bool       `json:"shareLocation"`
	Lat               *float64   `json:"-"`
	Lng               *float64   `json:"-"`
	LocationUpdatedAt *time.Time `json:"-"`
	Phone             string     `json:"phone"`
	// VendorRole distinguishes a vendor account's own owner from invited
	// staff — staff can use the vendor's existing tools (offers, scan/
	// redeem) but not team management. Meaningless for non-vendor roles.
	VendorRole        string `json:"vendorRole,omitempty"`
	NotifyRedemptions bool   `json:"notifyRedemptions"`
	NotifyReviews     bool   `json:"notifyReviews"`
	NotifyPayouts     bool   `json:"notifyPayouts"`
	NotifyGrowthTips  bool   `json:"notifyGrowthTips"`
	// GoogleID links this account to a Google identity for OAuth sign-in;
	// nil for accounts that have never signed in with Google.
	GoogleID *string `json:"-"`
}

const (
	VendorRoleOwner   = "owner"
	VendorRoleStaff   = "staff"
	VendorRoleManager = "manager"
)

type VendorStatus string

const (
	VendorPending   VendorStatus = "pending"
	VendorApproved  VendorStatus = "approved"
	VendorRejected  VendorStatus = "rejected"
	VendorEdits     VendorStatus = "edits"     // edits requested
	VendorSuspended VendorStatus = "suspended" // pulled from the app after admin review
)

type Vendor struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Category    string          `json:"category"` // BUSINESS_TYPES id
	Town        string          `json:"town"`
	RegionID    *string         `json:"regionId,omitempty"`
	Owner       string          `json:"owner"`
	Status      VendorStatus    `json:"status"`
	Priority    string          `json:"priority"` // high/normal/low
	Docs        bool            `json:"docs"`
	Photos      int             `json:"photos"`
	WifiScore   *float64        `json:"wifi,omitempty"`
	Flags       []string        `json:"flags"`
	Note        string          `json:"note"`
	Tier        string          `json:"tier"`    // Premium/Free
	Profile     json.RawMessage `json:"profile"` // hours, amenities, blurb, wifi, bestTimes…
	Menu        json.RawMessage `json:"menu"`    // sections/items document
	SubmittedAt time.Time       `json:"submittedAt"`
	Lat         float64         `json:"lat"`
	Lng         float64         `json:"lng"`
	// Joined fields (not columns), populated by admin map/near queries.
	BadReviewCount int  `json:"badReviewCount,omitempty"`
	Flagged        bool `json:"flagged,omitempty"`
}

type OfferStatus string

const (
	OfferDraft  OfferStatus = "draft"
	OfferLive   OfferStatus = "live"
	OfferEnding OfferStatus = "ending"
	OfferEnded  OfferStatus = "ended"
)

type Offer struct {
	ID       string      `json:"id"`
	VendorID string      `json:"vendorId"`
	Title    string      `json:"title"`
	Type     string      `json:"type"` // Percentage/Amount/Freebie/Bundle
	Value    string      `json:"value"`
	Status   OfferStatus `json:"status"`
	StartAt  string      `json:"start"` // display strings kept from prototype
	EndAt    string      `json:"end"`
	Limit    int         `json:"limit"`
	// Claimed/Redeemed are live-computed from the claims table (see
	// Store.ClaimCountsByOfferIDs), not stored columns — never set these
	// directly when writing an Offer.
	Claimed  int `json:"claimed"`
	Redeemed int `json:"redeemed"`
	// Views counts real traveller impressions (POST /offers/{id}/view).
	Views int `json:"views"`
	// UpdatedAt is used to pick a vendor's "headline" offer to sync into
	// their Place's traveller-facing deal (see syncVendorPlace).
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

type ClaimStatus string

const (
	ClaimClaimed  ClaimStatus = "claimed"
	ClaimRedeemed ClaimStatus = "redeemed"
)

type Claim struct {
	ID      string `json:"id"`
	Code    string `json:"code"` // SPOTTO-XXXX, unique
	PlaceID string `json:"placeId"`
	// OfferID links back to the specific vendor Offer this was claimed
	// from — nil for claims made before this field existed, or where the
	// submitted offerId didn't resolve to a real offer for that vendor.
	OfferID     *string     `json:"offerId,omitempty"`
	OfferLabel  string      `json:"offerLabel"`
	OfferValue  string      `json:"offerValue"`
	VendorID    *string     `json:"vendorId,omitempty"` // vendor allowed to redeem
	TravellerID string      `json:"travellerId"`
	Status      ClaimStatus `json:"status"`
	ClaimedAt   time.Time   `json:"claimedAt"`
	RedeemedAt  *time.Time  `json:"redeemedAt,omitempty"`
	// Joined traveller display fields (not columns on claims).
	TravellerName    string `json:"travellerName,omitempty"`
	TravellerCountry string `json:"travellerCountry,omitempty"`
	TravellerStyle   string `json:"travellerStyle,omitempty"`
}

// Place is curated traveller content; Data mirrors the Dart Place model
// (wifi, deal, tags, menu, coordinates, vibes seed…).
type Place struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	VendorID *string         `json:"vendorId,omitempty"`
	Data     json.RawMessage `json:"data"`
}

type Trip struct {
	OwnerID   string          `json:"-"`
	Name      string          `json:"name"`
	Code      string          `json:"code"`
	StartDate string          `json:"startDate"`
	Collab    []string        `json:"collaborators"`
	Days      json.RawMessage `json:"days"` // same shape as Flutter Trip.toJson()['days']
	UpdatedAt time.Time       `json:"-"`
}

type Group struct {
	OwnerID   string   `json:"-"`
	Name      string   `json:"name"`
	Code      string   `json:"code"`
	MemberIDs []string `json:"memberIds"`
}

type FeedStatus string

const (
	FeedOK     FeedStatus = "ok"
	FeedReview FeedStatus = "review"
	FeedFlag   FeedStatus = "flag"
)

type FeedItem struct {
	ID        int64      `json:"id"`
	Vendor    string     `json:"vendor"`
	Town      string     `json:"town"`
	Type      string     `json:"type"` // Offer/Event
	Text      string     `json:"text"`
	Status    FeedStatus `json:"status"`
	Reason    string     `json:"reason,omitempty"`
	Resolved  string     `json:"resolved,omitempty"` // approved/removed/warned
	CreatedAt time.Time  `json:"createdAt"`
}

type Region struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Travelers int     `json:"travelers"`
	Vendors   int     `json:"vendors"`
}

// Review is a traveller's rating/text for a Place. Ratings of 1-2 count as
// "bad" for the purposes of flagging the owning vendor on the admin map.
type Review struct {
	ID          string    `json:"id"`
	PlaceID     string    `json:"placeId"`
	VendorID    *string   `json:"vendorId,omitempty"`
	TravellerID string    `json:"travellerId"`
	Rating      int       `json:"rating"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"createdAt"`
	// Joined display field (not a column).
	TravellerName string `json:"travellerName,omitempty"`
}

type Subscription struct {
	ID       string     `json:"id"`
	VendorID *string    `json:"vendorId,omitempty"`
	Vendor   string     `json:"vendor"`
	Town     string     `json:"town"`
	Tier     string     `json:"tier"` // Premium/Free
	MRR      int        `json:"mrr"`
	RenewsAt *time.Time `json:"renewsAt,omitempty"`
	State    string     `json:"state"` // active/soon/overdue/trial/free
	Since    string     `json:"since"`
}

type AudioGuide struct {
	ID        string          `json:"id"`
	Site      string          `json:"site"`
	Region    string          `json:"region"`
	Lat       float64         `json:"lat"`
	Lng       float64         `json:"lng"`
	Lang      string          `json:"lang"`
	Published bool            `json:"published"`
	Plays     int             `json:"plays"`
	Depths    json.RawMessage `json:"depths"`   // ["Quick","Standard"]
	Duration  json.RawMessage `json:"duration"` // {"Quick":"4m",…}
}

type VendorNotification struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`
	Tone   string `json:"tone"`
	Title  string `json:"title"`
	Sub    string `json:"sub"`
	Time   string `json:"time"`
	Unread bool   `json:"unread"`
}

// VendorInvite is an expiring, single-use code an owner generates to bring
// a staff or manager member onto their vendor account. It's emailed to the
// invitee (see mail.Mailer) but the code/link is also always returned to
// the owner, so the invite still works as a copy-link if email delivery
// isn't configured or bounces.
type VendorInvite struct {
	ID       string `json:"id"`
	VendorID string `json:"vendorId"`
	Code     string `json:"code"`
	// Email is who the invite was addressed to (informational — anyone
	// with the code can still redeem it, same as before this field existed).
	Email          string     `json:"email,omitempty"`
	Role           string     `json:"role"` // staff or manager
	CreatedBy      string     `json:"createdBy"`
	CreatedAt      time.Time  `json:"createdAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	AcceptedAt     *time.Time `json:"acceptedAt,omitempty"`
	AcceptedUserID *string    `json:"acceptedUserId,omitempty"`
	Revoked        bool       `json:"revoked"`
}

// VendorProfileGrant is a time-limited exception letting a normally-locked-out
// manager edit Business Profile: the owner picks a duration (15/30/60 min)
// and a PIN is generated. PinExpiresAt bounds how long the PIN itself is
// redeemable (fixed, short — independent of DurationMins, so a grant doesn't
// silently burn down unread in an inbox); AccessExpiresAt is set only once
// redeemed, to RedeemedAt+DurationMins, and is what actually gates writes.
type VendorProfileGrant struct {
	ID              string     `json:"id"`
	VendorID        string     `json:"vendorId"`
	GranteeID       string     `json:"granteeId"`
	CreatedBy       string     `json:"createdBy"`
	PinHash         string     `json:"-"`
	DurationMins    int        `json:"durationMins"`
	CreatedAt       time.Time  `json:"createdAt"`
	PinExpiresAt    time.Time  `json:"pinExpiresAt"`
	RedeemedAt      *time.Time `json:"redeemedAt,omitempty"`
	AccessExpiresAt *time.Time `json:"accessExpiresAt,omitempty"`
	RevokedAt       *time.Time `json:"revokedAt,omitempty"`
	RevokedBy       string     `json:"revokedBy,omitempty"`
}

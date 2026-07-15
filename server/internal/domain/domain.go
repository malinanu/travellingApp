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
}

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
	Claimed  int         `json:"claimed"`
	Redeemed int         `json:"redeemed"`
}

type ClaimStatus string

const (
	ClaimClaimed  ClaimStatus = "claimed"
	ClaimRedeemed ClaimStatus = "redeemed"
)

type Claim struct {
	ID          string      `json:"id"`
	Code        string      `json:"code"` // SPOTTO-XXXX, unique
	PlaceID     string      `json:"placeId"`
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

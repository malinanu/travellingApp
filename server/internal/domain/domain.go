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
	TravelStyle       []string        `json:"travelStyle"`
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
	// UsageAnalytics is the app's "Usage analytics" switch (on unless the
	// traveller turns it off): while on, the rough ~1 km area they use the
	// map from is recorded for the admin geo page (traveller_presence).
	UsageAnalytics bool `json:"usageAnalytics"`
	// Publisher lets the account write travel content in the Command
	// Center (set by an admin); admins can always write.
	Publisher bool `json:"publisher"`
	Lat               *float64   `json:"-"`
	Lng               *float64   `json:"-"`
	LocationUpdatedAt *time.Time `json:"-"`
	Phone             string     `json:"phone"`
	// PhoneVerified is true once the account confirmed Phone with a one-time
	// code (WhatsApp or SMS); changing the number clears it.
	PhoneVerified bool `json:"phoneVerified"`
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
	AppleID  *string `json:"-"`
	// AvatarURL is the public R2 URL of the account's profile picture, or
	// "" to fall back to initials.
	AvatarURL string `json:"avatarUrl"`
}

type UserPushToken struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	DeviceToken string    `json:"deviceToken"`
	Platform    string    `json:"platform"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PaymentMethod struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Brand      string    `json:"brand"`
	Last4      string    `json:"last4"`
	ExpMonth   int       `json:"expMonth"`
	ExpYear    int       `json:"expYear"`
	HolderName string    `json:"holderName"`
	IsDefault  bool      `json:"isDefault"`
	CreatedAt  time.Time `json:"createdAt"`
}

type PaymentTransaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	VendorID    *string   `json:"vendorId,omitempty"`
	AmountCents int       `json:"amountCents"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Media is one uploaded object stored in Cloudflare R2 — see internal/media
// and internal/http/media.go. Kind is image|video|doc; Role distinguishes
// cover/gallery/promo/doc/avatar/review/contrib/menu_item.
type Media struct {
	ID        string `json:"id"`
	OwnerType string `json:"ownerType"`
	OwnerID   string `json:"ownerId"`
	Kind      string `json:"kind"`
	Role      string `json:"role"`
	ObjectKey string `json:"objectKey"`
	URL       string `json:"url"`
	// ThumbKey/ThumbURL are only ever set for Kind == "image" — a small
	// (~320px) JPEG derivative generated in the background by the
	// thumbnail worker (see ThumbStatus, internal/http/thumbnail_worker.go),
	// same key-stored/URL-derived pairing as ObjectKey/URL. Empty for media
	// uploaded before this existed, for non-image kinds, while generation
	// is still pending, or if it failed — clients fall back to URL in that
	// case rather than treating it as an error.
	ThumbKey string `json:"-"`
	ThumbURL string `json:"thumbUrl,omitempty"`
	// MicroThumb is a tiny (~24px) heavily-compressed JPEG inlined as a
	// base64 data URI — generated alongside ThumbKey, shown instantly as a
	// blur-up placeholder without any extra network fetch. Empty under the
	// same conditions as ThumbURL.
	MicroThumb string `json:"microThumb,omitempty"`
	// ThumbStatus drives the thumbnail worker's sweep (pending/done/error)
	// — never client-facing.
	ThumbStatus string    `json:"-"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	UploadedBy  string    `json:"uploadedBy"`
	Status      string    `json:"status"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
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

// BRCertOCRStatus tracks the Gemini pre-fill step for a vendor's Business
// Registration certificate — see internal/gemini and br_cert_scheduler.go.
type BRCertOCRStatus string

const (
	BRCertOCRNone    BRCertOCRStatus = ""        // no BR cert doc uploaded yet
	BRCertOCRPending BRCertOCRStatus = "pending" // doc uploaded, scheduler hasn't run OCR yet
	BRCertOCRDone    BRCertOCRStatus = "done"    // OCR ran (found a candidate, or didn't — see BRCertNumber)
	BRCertOCRError   BRCertOCRStatus = "error"   // the Gemini call itself failed, not just "no number found"
)

// BRCertVerifyStatus tracks verification of BRCertNumber against Sri
// Lanka's DRC eROC registry — see internal/eroc and br_cert_scheduler.go.
type BRCertVerifyStatus string

const (
	BRCertVerifyNone         BRCertVerifyStatus = ""              // no cert number to verify yet
	BRCertVerifyPending      BRCertVerifyStatus = "pending"       // queued for the eROC sweep
	BRCertVerifyVerified     BRCertVerifyStatus = "verified"      // eROC status:true
	BRCertVerifyInvalid      BRCertVerifyStatus = "invalid"       // eROC status:false — definitive, not retried
	BRCertVerifyManualReview BRCertVerifyStatus = "manual_review" // exhausted the retry budget on transient failures
)

type Vendor struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"` // BUSINESS_TYPES id
	Town     string `json:"town"`
	// Country is an ISO 3166-1 alpha-2 code (e.g. "LK", "US"), used to pick
	// the right public-holiday calendar — see internal/holidays. Defaults
	// to "LK" (see migration 009) since every vendor predating this field
	// really is Sri Lanka.
	Country     string          `json:"country"`
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
	// GooglePlaceID is the Google Maps Place ID captured when the vendor
	// found their business via the registration wizard's Google Places
	// search — nil when the owner typed everything manually. Stored as a
	// first-class column (not only inside places.data JSONB) so that
	// enrichWithPlaceDetails can look up live details for vendor-registered
	// places and so duplicate Google-listing registrations can be caught.
	GooglePlaceID *string `json:"googlePlaceId,omitempty"`
	// BRCertNumber is the vendor's Business Registration certificate number
	// (e.g. "PV00334782") — Gemini-suggested from the uploaded "br_cert"
	// media (see internal/gemini), always editable by the vendor before/
	// after that, and verified in the background against eROC (see
	// internal/eroc). See BRCertOCRStatus/BRCertVerifyStatus for state.
	BRCertNumber       string             `json:"brCertNumber"`
	BRCertOCRStatus    BRCertOCRStatus    `json:"brCertOcrStatus"`
	BRCertVerifyStatus BRCertVerifyStatus `json:"brCertVerifyStatus"`
	BRCertCompanyName  string             `json:"brCertCompanyName,omitempty"` // eROC's company_name
	BRCertMessage      string             `json:"brCertMessage,omitempty"`     // eROC's message, or last error
	BRCertAttempts     int                `json:"brCertAttempts,omitempty"`    // consecutive transient verify failures
	BRCertVerifiedAt   *time.Time         `json:"brCertVerifiedAt,omitempty"`
	// Joined fields (not columns), populated by admin map/near queries.
	BadReviewCount int  `json:"badReviewCount,omitempty"`
	Flagged        bool `json:"flagged,omitempty"`
	// Vibe is the vendor's synced-place crowdsourced vibe reading —
	// populated only by handleVendorBusiness (vendor's own view) and
	// handleAdminVendorVibe (admin's view), nil everywhere else. Distinct
	// from the vendor-entered Wi-Fi/speed "Connectivity & vibe" card in the
	// profile page, which is self-reported, not traveller check-in data.
	Vibe *VibeAggregate `json:"vibe,omitempty"`
}

type DailyQuest struct {
	Goal              int    `json:"goal"`
	Progress          int    `json:"progress"`
	RewardDescription string `json:"rewardDescription"`
	IsComplete        bool   `json:"isComplete"`
	SponsorName       string `json:"sponsorName,omitempty"`
	SponsorID         string `json:"sponsorId,omitempty"`
	// Date is the UTC quest day this status is for — clients key any saved
	// copy by it rather than by their own clock.
	Date        string     `json:"date"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	RedeemedAt  *time.Time `json:"redeemedAt,omitempty"`
}

// QuestCompletion is a traveller finishing one day's quest, joined with who
// they are and whether the sponsor has scanned their reward yet.
type QuestCompletion struct {
	TravellerID       string     `json:"travellerId"`
	TravellerName     string     `json:"travellerName"`
	TravellerEmail    string     `json:"travellerEmail,omitempty"`
	Date              string     `json:"date"`
	VendorID          *string    `json:"vendorId,omitempty"`
	SponsorName       string     `json:"sponsorName"`
	Goal              int        `json:"goal"`
	RewardDescription string     `json:"rewardDescription"`
	CompletedAt       time.Time  `json:"completedAt"`
	RedeemedAt        *time.Time `json:"redeemedAt,omitempty"`
}

// QuestCompletionFilter narrows ListQuestCompletions. Empty fields don't
// filter; Since is an inclusive 'YYYY-MM-DD'; Before pages by completed_at.
type QuestCompletionFilter struct {
	Date        string
	Since       string
	VendorID    string
	TravellerID string
	Before      *time.Time
	Limit       int
}

type VendorSponsorship struct {
	VendorID          string    `json:"vendorId"`
	RewardDescription string    `json:"rewardDescription"`
	RewardLimit       int       `json:"rewardLimit"`
	IsActive          bool      `json:"isActive"`
	Status            string    `json:"status"` // pending, approved, rejected
	CreatedAt         time.Time `json:"createdAt"`
}

type QuestSchedule struct {
	Date              string `json:"date"`
	VendorID          string `json:"vendorId"`
	Goal              int    `json:"goal"`
	SponsorName       string `json:"sponsorName,omitempty"`
	RewardDescription string `json:"rewardDescription,omitempty"`
	// Filled by ListQuestSchedules only.
	Completions int `json:"completions"`
	Redemptions int `json:"redemptions"`
}

type QuestClaim struct {
	ID          string    `json:"id"`
	TravellerID string    `json:"travellerId"`
	VendorID    string    `json:"vendorId"`
	Date        string    `json:"date"`
	ClaimedAt   time.Time `json:"claimedAt"`
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
	TravellerName    string   `json:"travellerName,omitempty"`
	TravellerCountry string   `json:"travellerCountry,omitempty"`
	TravellerStyle   []string `json:"travellerStyle,omitempty"`
}

// Place is curated traveller content; Data mirrors the Dart Place model
// (wifi, deal, tags, menu, coordinates, vibes seed…).
type Place struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	VendorID *string         `json:"vendorId,omitempty"`
	Data     json.RawMessage `json:"data"`
	// Embedding is a semantic-search vector for Local Guide (see
	// internal/recommend, internal/http/embedding_worker.go). Its own
	// column, not part of Data, and never serialized to travellers.
	Embedding []float64 `json:"-"`
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
	// Photos are the review's attached photos (media role "review"), filled
	// in by handlePlaceReviews — not a column on reviews.
	Photos []ReviewPhoto `json:"photos,omitempty"`
}

// ReviewPhoto is one attached review photo. ThumbKey/ThumbURL mirror
// Media's own split (see Media.ThumbKey/ThumbURL): ThumbKey is the raw R2
// key for a small derivative, never serialized directly — handlePlaceReviews
// resolves and signs it into ThumbURL, which is empty for photos uploaded
// before thumbnails existed or where generation failed, in which case
// clients fall back to URL.
type ReviewPhoto struct {
	URL        string `json:"url"`
	ThumbKey   string `json:"-"`
	ThumbURL   string `json:"thumbUrl,omitempty"`
	MicroThumb string `json:"microThumb,omitempty"`
}

// VibeCheckin is one traveller's timestamped snapshot rating of a place
// across all three dimensions at once — a traveller can have many of these
// per place over time (one per visit), never edited, only added to.
type VibeCheckin struct {
	PlaceID     string    `json:"placeId"`
	TravellerID string    `json:"travellerId"`
	Productive  int       `json:"productive"`
	Loud        int       `json:"loud"`
	Crowded     int       `json:"crowded"`
	CreatedAt   time.Time `json:"createdAt"`
}

// VibeAggregate is a place's current vibe reading — the average of each
// traveller's most recent check-in (not an all-time blend), so it stays
// representative of how the place is right now rather than years ago.
type VibeAggregate struct {
	Productive float64 `json:"productive"`
	Loud       float64 `json:"loud"`
	Crowded    float64 `json:"crowded"`
	SampleSize int     `json:"sampleSize"`
}

// GuideConversation is one traveller's ongoing chat with the Local Guide
// assistant (see internal/http/guide.go) — a traveller can have many over
// time, resumed via GET /guide/conversations/latest.
type GuideConversation struct {
	ID          string    `json:"id"`
	TravellerID string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GuideMessage is one turn in a GuideConversation. Only the user-visible
// text is stored — intermediate tool-call/tool-result exchanges the model
// used to arrive at an assistant reply are not persisted; each new user
// message re-derives whatever tool calls it needs from the conversation's
// prior plain-text turns, so there's nothing to replay.
type GuideMessage struct {
	ID             int64           `json:"-"`
	ConversationID string          `json:"conversationId"`
	Role           string          `json:"role"` // "user" | "assistant"
	Content        string          `json:"content"`
	PlaceIDs       []string        `json:"placeIds,omitempty"` // places referenced/recommended in this turn
	Payload        json.RawMessage `json:"payload,omitempty"`  // card shown with an assistant turn; see guide_planner.go
	CreatedAt      time.Time       `json:"createdAt"`
}

// GuideRequest is one row per POST /guide/chat call, success or failure —
// admin-dashboard telemetry, distinct from GuideMessage which only records
// successful turns (see internal/http/guide.go's recordGuideRequest).
type GuideRequest struct {
	ConversationID    string    `json:"conversationId"`
	TravellerID       string    `json:"-"`
	Status            string    `json:"status"` // "ok" | "rate_limited" | "error"
	ErrorMessage      string    `json:"errorMessage,omitempty"`
	LatencyMs         int       `json:"latencyMs"`
	ToolsUsed         []string  `json:"toolsUsed"`
	UsedSemanticQuery bool      `json:"usedSemanticQuery"`
	InputTokens       int       `json:"inputTokens"`
	OutputTokens      int       `json:"outputTokens"`
	CreatedAt         time.Time `json:"createdAt"`
}

// GuideStats is the admin Local Guide dashboard's aggregate summary.
type GuideStats struct {
	TotalConversations   int     `json:"totalConversations"`
	TotalMessages        int     `json:"totalMessages"`
	MessagesToday        int     `json:"messagesToday"`
	Messages7d           int     `json:"messages7d"`
	Requests7d           int     `json:"requests7d"`
	OkCount7d            int     `json:"okCount7d"`
	RateLimitedCount7d   int     `json:"rateLimitedCount7d"`
	ErrorCount7d         int     `json:"errorCount7d"`
	AvgLatencyMs7d       float64 `json:"avgLatencyMs7d"`
	SemanticQueryCount7d int     `json:"semanticQueryCount7d"`
	PlacesEmbedded       int     `json:"placesEmbedded"`
	PlacesTotal          int     `json:"placesTotal"`
}

// ToolUsageCount is one row of GuideStats' tool-usage breakdown.
type ToolUsageCount struct {
	Tool  string `json:"tool"`
	Count int    `json:"count"`
}

// AIUsageEvent is one upstream Gemini call from any feature, recorded for
// the admin dashboard (see internal/http/ai_usage.go).
type AIUsageEvent struct {
	Feature         string
	Kind            string // "chat" | "embed"
	Model           string
	Status          string // "ok" | "rate_limited" | "error"
	ErrorMessage    string
	InputTokens     int
	OutputTokens    int
	TokensEstimated bool
	Items           int
	LatencyMs       int
}

// AIUsageRow aggregates one feature+model's Gemini calls over the dashboard's
// three windows. "Today" is the Pacific-time day, matching when Google resets
// daily quotas.
type AIUsageRow struct {
	Feature           string `json:"feature"`
	Kind              string `json:"kind"`
	Model             string `json:"model"`
	RequestsToday     int    `json:"requestsToday"`
	Requests7d        int    `json:"requests7d"`
	Requests30d       int    `json:"requests30d"`
	Errors7d          int    `json:"errors7d"`
	RateLimited7d     int    `json:"rateLimited7d"`
	InputTokensToday  int    `json:"inputTokensToday"`
	OutputTokensToday int    `json:"outputTokensToday"`
	InputTokens7d     int    `json:"inputTokens7d"`
	OutputTokens7d    int    `json:"outputTokens7d"`
	InputTokens30d    int    `json:"inputTokens30d"`
	OutputTokens30d   int    `json:"outputTokens30d"`
	TokensEstimated   bool   `json:"tokensEstimated"`
}

// AIDailyPoint is one Pacific-time day in the dashboard's 14-day chart.
type AIDailyPoint struct {
	Day          string `json:"day"` // YYYY-MM-DD
	Requests     int    `json:"requests"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

// AIQuota is the admin-entered daily limits "left to use" is measured
// against — Gemini exposes no remaining-quota API.
type AIQuota struct {
	ChatRequestsPerDay  int `json:"chatRequestsPerDay"`
	ChatTokensPerDay    int `json:"chatTokensPerDay"`
	EmbedRequestsPerDay int `json:"embedRequestsPerDay"`
}

// AIQuotaUsage is today's (Pacific-time) consumption against AIQuota.
// Embedding requests count each embedded item, as Google does for batches.
type AIQuotaUsage struct {
	ChatRequestsToday  int `json:"chatRequestsToday"`
	ChatTokensToday    int `json:"chatTokensToday"`
	EmbedRequestsToday int `json:"embedRequestsToday"`
}

// EmbeddingStats is the place-embedding backfill's progress for the current
// embedding model.
type EmbeddingStats struct {
	Model    string `json:"model"`
	Total    int    `json:"total"`
	Embedded int    `json:"embedded"` // embedded with the current model
	Pending  int    `json:"pending"`  // never embedded, or embedded with another model
}

// AIErrorEntry is one failed Gemini call from any feature, with the real
// upstream error text — the traveller only ever sees a generic message, so
// this is the diagnostic surface for auth/quota/model-retirement failures.
type AIErrorEntry struct {
	Feature      string    `json:"feature"`
	Model        string    `json:"model"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"errorMessage"`
	CreatedAt    time.Time `json:"createdAt"`
}

// GuideConversationSummary is one row in the admin dashboard's browsable
// conversation list.
type GuideConversationSummary struct {
	ID            string    `json:"id"`
	TravellerName string    `json:"travellerName"`
	MessageCount  int       `json:"messageCount"`
	LastMessage   string    `json:"lastMessage"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// GuideUserSummary is one row in the admin dashboard's browsable user list.
type GuideUserSummary struct {
	TravellerID   string    `json:"travellerId"`
	TravellerName string    `json:"travellerName"`
	TotalMessages int       `json:"totalMessages"`
	ErrorCount    int       `json:"errorCount"`
	InputTokens   int       `json:"inputTokens"`
	OutputTokens  int       `json:"outputTokens"`
	TripsPlanned  int       `json:"tripsPlanned"`
	LastActiveAt  time.Time `json:"lastActiveAt"`
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

type VendorNotification struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`
	Tone   string `json:"tone"`
	Title  string `json:"title"`
	Sub    string `json:"sub"`
	Time   string `json:"time"`
	Unread bool   `json:"unread"`
}

// AdminNotification mirrors VendorNotification's shape but is never stored
// as a discrete row — see Store.AdminNotifications, which derives these
// live from current pending/flagged/bad-review counts each request.
type AdminNotification struct {
	ID     string `json:"id"`
	Icon   string `json:"icon"`
	Tone   string `json:"tone"`
	Title  string `json:"title"`
	Sub    string `json:"sub"`
	Time   string `json:"time"`
	Unread bool   `json:"unread"`
}

// AdminAuditLog records a single destructive action a platform admin took
// (approve/reject/suspend a vendor, resolve a feed item, revoke a vendor
// grant, upgrade a subscription) — see httpapi.(*Server).logAdminAction.
// ActorEmail/ActorName are denormalized at write time so entries stay
// readable even if the acting admin's account is later deleted.
type AdminAuditLog struct {
	ID         int64           `json:"id"`
	ActorID    string          `json:"actorId"`
	ActorEmail string          `json:"actorEmail"`
	ActorName  string          `json:"actorName"`
	Action     string          `json:"action"`
	TargetType string          `json:"targetType"`
	TargetID   string          `json:"targetId"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
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

type PlatformAnalytics struct {
	TotalUsers    int `json:"totalUsers"`
	TotalVendors  int `json:"totalVendors"`
	TotalClaims   int `json:"totalClaims"`
	TotalReviews  int `json:"totalReviews"`
}

// Session is one signed-in device (a user_sessions row). The refresh token
// itself is never stored or returned.
type Session struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Client      string     `json:"client"` // app | admin | vendor
	DeviceLabel string     `json:"deviceLabel"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  time.Time  `json:"lastUsedAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	RotatedAt   *time.Time `json:"-"`
}

// AdminUserSummary is one row of the admin Users list.
type AdminUserSummary struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Role           Role       `json:"role"`
	VendorName     string     `json:"vendorName,omitempty"`
	Publisher      bool       `json:"publisher"`
	CreatedAt      time.Time  `json:"createdAt"`
	ActiveSessions int        `json:"activeSessions"`
	LastActiveAt   *time.Time `json:"lastActiveAt,omitempty"`
}

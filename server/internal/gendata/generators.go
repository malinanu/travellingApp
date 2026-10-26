package gendata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func genTravellers(ctx context.Context, store *postgres.Store, passwordHash string, n int, now time.Time) ([]domain.User, error) {
	out := make([]domain.User, 0, n)
	for i := 0; i < n; i++ {
		style := gofakeit.RandomString(travelStyles)
		numInterests := gofakeit.Number(1, 3)
		pool := append([]string{}, interestPool...)
		gofakeit.ShuffleStrings(pool)
		u := domain.User{
			ID:           newID("usr"),
			Email:        uniqueEmail(),
			PasswordHash: passwordHash,
			Role:         domain.RoleTraveller,
			Name:         gofakeit.Name(),
			Country:      gofakeit.Country(),
			TravelStyle:  []string{style},
			Interests:    pool[:numInterests],
			CreatedAt:    now.Add(-time.Duration(gofakeit.Number(1, 180)) * 24 * time.Hour),
		}
		if err := store.CreateUser(ctx, u); err != nil {
			return nil, fmt.Errorf("traveller %s: %w", u.Email, err)
		}
		out = append(out, u)
	}
	return out, nil
}

func genAdmin(ctx context.Context, store *postgres.Store, passwordHash string, now time.Time) (domain.User, error) {
	u := domain.User{
		ID:           newID("usr"),
		Email:        uniqueEmail(),
		PasswordHash: passwordHash,
		Role:         domain.RoleAdmin,
		Name:         gofakeit.Name(),
		Country:      "Sri Lanka",
		CreatedAt:    now.Add(-90 * 24 * time.Hour),
	}
	if err := store.CreateUser(ctx, u); err != nil {
		return u, fmt.Errorf("admin: %w", err)
	}
	return u, nil
}

func weightedVendorStatus() domain.VendorStatus {
	// 70% approved, 10% pending, 8% rejected, 7% edits, 5% suspended
	roll := gofakeit.Number(1, 100)
	switch {
	case roll <= 70:
		return domain.VendorApproved
	case roll <= 80:
		return domain.VendorPending
	case roll <= 88:
		return domain.VendorRejected
	case roll <= 95:
		return domain.VendorEdits
	default:
		return domain.VendorSuspended
	}
}

func vendorBusinessName(category string) string {
	switch gofakeit.Number(0, 2) {
	case 0:
		return fmt.Sprintf("%s's %s", gofakeit.FirstName(), category)
	case 1:
		return fmt.Sprintf("%s %s", gofakeit.City(), category)
	default:
		return fmt.Sprintf("%s %s", capFirst(gofakeit.Adjective()), category)
	}
}

// genVendorsAndPlaces creates n vendors (+ one user account each) spread
// across the real regions, and a curated place for every approved vendor
// plus a handful of extra standalone points of interest.
func genVendorsAndPlaces(ctx context.Context, store *postgres.Store, passwordHash string, n int, regions []domain.Region, now time.Time) ([]domain.Vendor, []domain.User, []domain.Place, error) {
	vendors := make([]domain.Vendor, 0, n)
	vendorUsers := make([]domain.User, 0, n)
	places := make([]domain.Place, 0, n+12)
	vendorCategories := domain.OnboardableCategories()

	for i := 0; i < n; i++ {
		region := regions[gofakeit.Number(0, len(regions)-1)]
		category := vendorCategories[gofakeit.Number(0, len(vendorCategories)-1)].ID
		status := weightedVendorStatus()
		id := newID("ven")
		var wifi *float64
		if gofakeit.Number(1, 100) <= 80 {
			wifi = ptr(gofakeit.Float64Range(3, 10))
		}
		var flags []string
		if flag := gofakeit.RandomString(vendorNoteFlags); flag != "" && gofakeit.Number(1, 100) <= 20 {
			flags = []string{flag}
		} else {
			flags = []string{}
		}
		v := domain.Vendor{
			ID:          id,
			Name:        vendorBusinessName(category),
			Category:    category,
			Town:        region.Name,
			RegionID:    ptr(region.ID),
			Owner:       gofakeit.Name(),
			Status:      status,
			Priority:    gofakeit.RandomString([]string{"high", "normal", "low"}),
			Docs:        gofakeit.Bool(),
			Photos:      gofakeit.Number(0, 15),
			WifiScore:   wifi,
			Flags:       flags,
			Tier:        gofakeit.RandomString([]string{"Free", "Free", "Free", "Premium"}), // ~25% Premium
			SubmittedAt: now.Add(-time.Duration(gofakeit.Number(1, 200)) * 24 * time.Hour),
			Lat:         jitter(region.Lat, 0.05),
			Lng:         jitter(region.Lng, 0.05),
		}
		if err := store.UpsertVendor(ctx, v); err != nil {
			return nil, nil, nil, fmt.Errorf("vendor %s: %w", v.Name, err)
		}
		vendors = append(vendors, v)

		vu := domain.User{
			ID:           newID("usr"),
			Email:        uniqueEmail(),
			PasswordHash: passwordHash,
			Role:         domain.RoleVendor,
			Name:         v.Owner,
			Country:      "Sri Lanka",
			VendorID:     ptr(v.ID),
			CreatedAt:    v.SubmittedAt,
		}
		if err := store.CreateUser(ctx, vu); err != nil {
			return nil, nil, nil, fmt.Errorf("vendor user %s: %w", vu.Email, err)
		}
		vendorUsers = append(vendorUsers, vu)

		if status == domain.VendorApproved {
			p, err := genCuratedPlace(v)
			if err != nil {
				return nil, nil, nil, err
			}
			if err := store.UpsertPlace(ctx, p); err != nil {
				return nil, nil, nil, fmt.Errorf("place %s: %w", p.ID, err)
			}
			places = append(places, p)
		}
	}

	// A handful of extra points of interest with no vendor account —
	// beaches, viewpoints, markets — same as the trip-catalog POIs added
	// earlier this session, but generated instead of hand-listed.
	extra := 12
	for i := 0; i < extra; i++ {
		region := regions[gofakeit.Number(0, len(regions)-1)]
		p, err := genPOIPlace(region)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := store.UpsertPlace(ctx, p); err != nil {
			return nil, nil, nil, fmt.Errorf("poi place %s: %w", p.ID, err)
		}
		places = append(places, p)
	}

	return vendors, vendorUsers, places, nil
}

func genMenu() []map[string]any {
	sections := []string{"Breakfast", "Mains", "Drinks"}
	var out []map[string]any
	for si, sec := range sections {
		var items []map[string]any
		for ii := 0; ii < gofakeit.Number(2, 3); ii++ {
			items = append(items, map[string]any{
				"id":    fmt.Sprintf("m%d%d", si, ii),
				"name":  gofakeit.RandomString(menuItemNames),
				"price": gofakeit.Number(300, 2200),
				"desc":  gofakeit.RandomString(menuDescTemplates),
				"icon":  "UtensilsCrossed",
			})
		}
		out = append(out, map[string]any{"id": fmt.Sprintf("s%d", si), "name": sec, "items": items})
	}
	return out
}

func genCuratedPlace(v domain.Vendor) (domain.Place, error) {
	tagPool := append([]string{}, placeTags...)
	gofakeit.ShuffleStrings(tagPool)
	tags := tagPool[:gofakeit.Number(2, 4)]

	stylePool := append([]string{}, travelStyles...)
	gofakeit.ShuffleStrings(stylePool)
	styles := stylePool[:gofakeit.Number(1, 2)]

	hasDeal := gofakeit.Number(1, 100) <= 60
	deal := map[string]any{"active": false, "label": "", "tag": ""}
	if hasDeal {
		deal = map[string]any{
			"active": true,
			"label":  gofakeit.RandomString(dealLabelTemplates),
			"tag":    gofakeit.RandomString([]string{"Perk", "20% OFF", "Happy Hr", "Workshop"}),
		}
	}

	data := map[string]any{
		"type":       v.Category,
		"icon":       gofakeit.RandomString(placeIcons),
		"color":      gofakeit.RandomString(placeColors),
		"lat":        v.Lat,
		"lng":        v.Lng,
		"rating":     gofakeit.Float64Range(3.5, 5.0),
		"reviews":    gofakeit.Number(5, 500),
		"distance":   fmt.Sprintf("%d m", gofakeit.Number(100, 1500)),
		"priceLevel": gofakeit.RandomString([]string{"$", "$$", "$$$"}),
		"wifi": map[string]any{
			"score": gofakeit.Float64Range(3, 10),
			"speed": fmt.Sprintf("%d Mbps", gofakeit.Number(10, 100)),
			"label": gofakeit.RandomString([]string{"Okay", "Good", "Blazing"}),
		},
		"hours":     "7:00 AM – 9:00 PM",
		"openNow":   gofakeit.Bool(),
		"bestTimes": []string{gofakeit.RandomString([]string{"morning", "afternoon", "evening"})},
		"deal":      deal,
		"tags":      tags,
		"blurb":     gofakeit.RandomString(blurbTemplates),
		"seaView":   gofakeit.Bool(),
		"photo":     gofakeit.RandomString([]string{"cowork", "surf", "sunset", "clay", "batik"}),
		"menu":      genMenu(),
		"cost":      gofakeit.Number(500, 6000),
		"styles":    styles,
		"hint":      fmt.Sprintf("%d:00", gofakeit.Number(6, 21)),
		"curated":   true,
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return domain.Place{}, err
	}
	return domain.Place{ID: v.ID, Name: v.Name, VendorID: ptr(v.ID), Data: raw}, nil
}

func genPOIPlace(region domain.Region) (domain.Place, error) {
	typ := gofakeit.RandomString(poiTypes)
	stylePool := append([]string{}, travelStyles...)
	gofakeit.ShuffleStrings(stylePool)
	data := map[string]any{
		"type":    typ,
		"icon":    gofakeit.RandomString(placeIcons),
		"color":   gofakeit.RandomString(placeColors),
		"lat":     jitter(region.Lat, 0.06),
		"lng":     jitter(region.Lng, 0.06),
		"cost":    gofakeit.Number(0, 3000),
		"styles":  stylePool[:gofakeit.Number(1, 3)],
		"hint":    fmt.Sprintf("%d:00", gofakeit.Number(6, 21)),
		"curated": false,
	}
	name := fmt.Sprintf("%s %s", region.Name, typ)
	raw, err := json.Marshal(data)
	if err != nil {
		return domain.Place{}, err
	}
	return domain.Place{ID: newID("poi"), Name: name, Data: raw}, nil
}

func offerValueFor(typ string) string {
	switch typ {
	case "Percentage":
		return fmt.Sprintf("%d%% off", gofakeit.Number(10, 40))
	case "Amount":
		return fmt.Sprintf("Rs %d off", gofakeit.Number(200, 1000))
	case "Bundle":
		return fmt.Sprintf("%d-for-%d", gofakeit.Number(4, 6), gofakeit.Number(2, 3))
	default:
		return "Free item"
	}
}

func genOffers(ctx context.Context, store *postgres.Store, vendors []domain.Vendor) ([]domain.Offer, error) {
	var out []domain.Offer
	for _, v := range vendors {
		if v.Status != domain.VendorApproved {
			continue
		}
		n := gofakeit.Number(1, 3)
		for i := 0; i < n; i++ {
			typ := gofakeit.RandomString(offerTypes)
			limit := gofakeit.Number(20, 100)
			claimed := gofakeit.Number(0, limit)
			redeemed := gofakeit.Number(0, claimed)
			o := domain.Offer{
				ID:       newID("off"),
				VendorID: v.ID,
				Title:    fmt.Sprintf("%s — %s", gofakeit.RandomString(offerLeadPhrases), offerValueFor(typ)),
				Type:     typ,
				Value:    offerValueFor(typ),
				Status:   domain.OfferLive,
				StartAt:  "Today 07:00",
				EndAt:    "Today 21:00",
				Limit:    limit,
				Claimed:  claimed,
				Redeemed: redeemed,
			}
			if err := store.UpsertOffer(ctx, o); err != nil {
				return nil, fmt.Errorf("offer %s: %w", o.ID, err)
			}
			out = append(out, o)
		}
	}
	return out, nil
}

func genClaims(ctx context.Context, store *postgres.Store, travellers []domain.User, offers []domain.Offer, n int, now time.Time) ([]domain.Claim, error) {
	if len(travellers) == 0 || len(offers) == 0 {
		return nil, nil
	}
	used := map[string]bool{}
	var out []domain.Claim
	for i := 0; i < n; i++ {
		t := travellers[gofakeit.Number(0, len(travellers)-1)]
		o := offers[gofakeit.Number(0, len(offers)-1)]
		var code string
		for {
			code = newCode("SPOTTO-", 4)
			if !used[code] {
				used[code] = true
				break
			}
		}
		claimedAt := now.Add(-time.Duration(gofakeit.Number(0, 24*30)) * time.Hour)
		redeemed := gofakeit.Number(1, 100) <= 40
		c := domain.Claim{
			ID:          newID("clm"),
			Code:        code,
			PlaceID:     o.VendorID, // vendors' curated place shares the vendor's own id
			OfferLabel:  o.Title,
			OfferValue:  o.Value,
			VendorID:    ptr(o.VendorID),
			TravellerID: t.ID,
			Status:      domain.ClaimClaimed,
			ClaimedAt:   claimedAt,
		}
		if redeemed {
			c.Status = domain.ClaimRedeemed
			c.RedeemedAt = ptr(claimedAt.Add(time.Duration(gofakeit.Number(1, 48)) * time.Hour))
		}
		if err := store.CreateClaim(ctx, c); err != nil {
			return nil, fmt.Errorf("claim %s: %w", c.Code, err)
		}
		out = append(out, c)
	}
	return out, nil
}

func reviewText(rating int) string {
	// Combinatoric (aspect × phrase × closing) rather than gofakeit-generated
	// filler text in the middle of the sentence — gofakeit.Comment() reads as
	// nonsensical business jargon ("does this growth scale") when spliced
	// into a review, which is worse than a smaller but readable combination
	// space. 9 aspects × ~6 phrases × ~4 closings still gives well over 200
	// distinct reviews.
	aspect := capFirst(gofakeit.RandomString(reviewAspects))
	var phrase, closing string
	switch {
	case rating >= 4:
		phrase = gofakeit.RandomString(positivePhrases)
		closing = gofakeit.RandomString(closingPositive)
	case rating == 3:
		phrase = gofakeit.RandomString(neutralPhrases)
		closing = gofakeit.RandomString(closingPositive)
	default:
		phrase = gofakeit.RandomString(negativePhrases)
		closing = gofakeit.RandomString(closingNegative)
	}
	return fmt.Sprintf("%s %s. %s", aspect, phrase, closing)
}

func weightedRating() int {
	roll := gofakeit.Number(1, 100)
	switch {
	case roll <= 55:
		return 5
	case roll <= 80:
		return 4
	case roll <= 90:
		return 3
	case roll <= 96:
		return 2
	default:
		return 1
	}
}

func genReviews(ctx context.Context, store *postgres.Store, travellers []domain.User, places []domain.Place, n int, now time.Time) ([]domain.Review, error) {
	if len(travellers) == 0 || len(places) == 0 {
		return nil, nil
	}
	var out []domain.Review
	for i := 0; i < n; i++ {
		t := travellers[gofakeit.Number(0, len(travellers)-1)]
		p := places[gofakeit.Number(0, len(places)-1)]
		rating := weightedRating()
		rv := domain.Review{
			ID:          newID("rev"),
			PlaceID:     p.ID,
			VendorID:    p.VendorID,
			TravellerID: t.ID,
			Rating:      rating,
			Text:        reviewText(rating),
			CreatedAt:   now.Add(-time.Duration(gofakeit.Number(0, 24*90)) * time.Hour),
		}
		if err := store.UpsertReview(ctx, rv); err != nil {
			return nil, fmt.Errorf("review %s: %w", rv.ID, err)
		}
		// UpsertReview is ON CONFLICT(place_id, traveller_id) DO UPDATE, so a
		// repeat pairing at this volume just overwrites in place rather than
		// erroring — out may end up with fewer distinct rows than n if that
		// happens a few times, which is fine for test data.
		out = append(out, rv)
	}
	return out, nil
}

func genGroups(ctx context.Context, store *postgres.Store, travellers []domain.User, n int) (int, error) {
	if len(travellers) < 3 {
		return 0, nil
	}
	created := 0
	pool := append([]domain.User{}, travellers...)
	for i := len(pool) - 1; i > 0; i-- {
		j := gofakeit.Number(0, i)
		pool[i], pool[j] = pool[j], pool[i]
	}
	idx := 0
	for g := 0; g < n && idx+3 <= len(pool); g++ {
		owner := pool[idx]
		size := gofakeit.Number(2, 3)
		if idx+1+size > len(pool) {
			size = len(pool) - idx - 1
		}
		members := make([]string, 0, size)
		for j := 0; j < size; j++ {
			members = append(members, pool[idx+1+j].ID)
		}
		idx += 1 + size
		group := domain.Group{
			OwnerID:   owner.ID,
			Name:      fmt.Sprintf("%s's Crew", gofakeit.FirstName()),
			Code:      newCode("", 6),
			MemberIDs: members,
		}
		if err := store.UpsertGroup(ctx, group); err != nil {
			return created, fmt.Errorf("group for %s: %w", owner.ID, err)
		}
		created++
	}
	return created, nil
}

func genSubscriptions(ctx context.Context, store *postgres.Store, vendors []domain.Vendor, now time.Time) (int, error) {
	created := 0
	for _, v := range vendors {
		sub := domain.Subscription{
			ID:       newID("sub"),
			VendorID: ptr(v.ID),
			Vendor:   v.Name,
			Town:     v.Town,
			Tier:     v.Tier,
			Since:    v.SubmittedAt.Format("2006-01-02"),
		}
		if v.Tier == "Premium" {
			sub.MRR = gofakeit.Number(15, 60)
			sub.State = gofakeit.RandomString([]string{"active", "active", "soon", "overdue"})
			sub.RenewsAt = ptr(now.Add(time.Duration(gofakeit.Number(1, 30)) * 24 * time.Hour))
		} else {
			sub.MRR = 0
			sub.State = "free"
		}
		if err := store.UpsertSubscription(ctx, sub); err != nil {
			return created, fmt.Errorf("subscription %s: %w", sub.ID, err)
		}
		created++
	}
	return created, nil
}

func genVendorBlobs(ctx context.Context, store *postgres.Store, vendors []domain.Vendor, now time.Time) error {
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for _, v := range vendors {
		if v.Status != domain.VendorApproved {
			continue
		}
		var week []map[string]any
		for _, d := range days {
			week = append(week, map[string]any{"day": d, "views": gofakeit.Number(50, 400), "redemptions": gofakeit.Number(5, 60)})
		}
		dashboard := map[string]any{
			"kpis": []map[string]any{
				{"id": "offers", "label": "Active Offers", "value": gofakeit.Number(1, 5), "icon": "Tag", "tone": "teal"},
				{"id": "redeem", "label": "Total Redemptions", "value": gofakeit.Number(20, 400), "icon": "Ticket", "tone": "coral"},
				{"id": "views", "label": "Profile Views", "value": gofakeit.Number(200, 3000), "icon": "Eye", "tone": "sun"},
			},
			"week": week,
		}
		growth := map[string]any{
			"flash": map[string]any{
				"radiusKm": gofakeit.Number(1, 5), "inRange": gofakeit.Number(20, 300),
				"nomads": gofakeit.Number(5, 100), "surfers": gofakeit.Number(5, 100),
				"lastSentMins": gofakeit.Number(180, 1440),
			},
			"stampCard": map[string]any{
				"reward": "Free item", "stampsNeeded": gofakeit.Number(4, 8),
				"buckets":           []map[string]any{{"away": 1, "count": gofakeit.Number(5, 40), "label": "One stamp away"}},
				"totalHolders":      gofakeit.Number(20, 300),
				"redeemedThisMonth": gofakeit.Number(5, 60),
			},
			"btsClips": []map[string]any{},
			"boost": map[string]any{
				"event": map[string]any{"name": gofakeit.RandomString(boostEventNames), "when": "Fri · 5–9 PM"},
				"tiers": []map[string]any{
					{"id": "local", "label": "Local map top spot", "reach": "~1,200 travelers", "days": 3, "price": 12, "icon": "MapPin"},
				},
			},
		}
		reports := map[string]any{"week": week}

		dashboardJSON, _ := json.Marshal(dashboard)
		growthJSON, _ := json.Marshal(growth)
		reportsJSON, _ := json.Marshal(reports)

		if err := store.UpsertVendorBlob(ctx, v.ID, "dashboard", dashboardJSON); err != nil {
			return fmt.Errorf("dashboard blob %s: %w", v.ID, err)
		}
		if err := store.UpsertVendorBlob(ctx, v.ID, "growth", growthJSON); err != nil {
			return fmt.Errorf("growth blob %s: %w", v.ID, err)
		}
		if err := store.UpsertVendorBlob(ctx, v.ID, "reports", reportsJSON); err != nil {
			return fmt.Errorf("reports blob %s: %w", v.ID, err)
		}

		for i := 0; i < gofakeit.Number(1, 3); i++ {
			ni := gofakeit.Number(0, len(notificationTitles)-1)
			n := domain.VendorNotification{
				ID:     newID("ntf"),
				Icon:   gofakeit.RandomString([]string{"Bell", "Tag", "Star"}),
				Tone:   gofakeit.RandomString([]string{"info", "success", "warning"}),
				Title:  notificationTitles[ni],
				Sub:    notificationSubs[ni],
				Time:   now.Add(-time.Duration(gofakeit.Number(1, 72)) * time.Hour).Format("2006-01-02 15:04"),
				Unread: gofakeit.Bool(),
			}
			if err := store.UpsertVendorNotification(ctx, v.ID, n); err != nil {
				return fmt.Errorf("notification %s: %w", v.ID, err)
			}
		}
	}
	return nil
}

// syncRegionCounts recomputes each region's vendor count from the actual
// generated vendors (a real, attributable number). Travellers aren't tied
// to a region anywhere in the schema, so that count is a plausible random
// share of the total rather than a truly-computed figure — noted as a
// limitation, not silently pretended to be exact.
func syncRegionCounts(ctx context.Context, store *postgres.Store, regions []domain.Region, totalTravellers int, vendors []domain.Vendor) error {
	vendorCounts := map[string]int{}
	for _, v := range vendors {
		if v.RegionID != nil {
			vendorCounts[*v.RegionID]++
		}
	}
	for _, r := range regions {
		r.Vendors = vendorCounts[r.ID]
		r.Travelers = gofakeit.Number(totalTravellers/len(regions)/2, totalTravellers/2)
		if err := store.UpsertRegion(ctx, r); err != nil {
			return fmt.Errorf("region count %s: %w", r.ID, err)
		}
	}
	return nil
}

func buildSampleLogins(admin domain.User, vendorUsers []domain.User, travellers []domain.User) []string {
	lines := []string{
		fmt.Sprintf("admin:     %s / %s", admin.Email, TestPassword),
	}
	for i := 0; i < 2 && i < len(vendorUsers); i++ {
		lines = append(lines, fmt.Sprintf("vendor:    %s / %s", vendorUsers[i].Email, TestPassword))
	}
	for i := 0; i < 2 && i < len(travellers); i++ {
		lines = append(lines, fmt.Sprintf("traveller: %s / %s", travellers[i].Email, TestPassword))
	}
	return lines
}

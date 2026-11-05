package domain

import "time"

// PackageFeature represents a distinct feature capability that can be allocated
// to a vendor tier or held in the master feature pool.
type PackageFeature struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // discovery | marketing | loyalty | operations | insights | trust
	Icon        string `json:"icon"`
}

// VendorPackage represents a subscription tier for local business vendors.
type VendorPackage struct {
	ID                 string           `json:"id"`
	Slug               string           `json:"slug"` // starter | growth | pro
	Name               string           `json:"name"` // Starter | Growth | Pro
	Tagline            string           `json:"tagline"`
	MonthlyPriceUSD    float64          `json:"monthlyPriceUsd"`    // 3.99 | 11.99 | 22.99
	YearlyPriceUSD     float64          `json:"yearlyPriceUsd"`     // 39.90 | 119.90 | 229.90
	Badge              string           `json:"badge,omitempty"`    // "Most Popular" | "Best Value"
	Highlighted        bool             `json:"highlighted"`
	ActiveOffers       int              `json:"activeOffers"`       // 1, 5, -1 (unlimited)
	FlashSalesPerMonth int              `json:"flashSalesPerMonth"` // 0, 2, -1 (unlimited)
	TeamSeats          int              `json:"teamSeats"`          // 1, 3, -1 (unlimited)
	Features           []PackageFeature `json:"features"`
	ExcludedFeatures   []string         `json:"excludedFeatures,omitempty"`
	SortOrder          int              `json:"sortOrder"`
}

// PackagesConfig represents the complete package configuration stored in platform_settings.
type PackagesConfig struct {
	Packages    []VendorPackage  `json:"packages"`
	FeaturePool []PackageFeature `json:"featurePool"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// DefaultMasterFeatures provides the canonical master pool of all Spotto features.
func DefaultMasterFeatures() []PackageFeature {
	return []PackageFeature{
		{
			ID:          "feat_map_pin",
			Name:        "Verified Map Pin & Profile",
			Description: "Pin on the traveller discovery map, photos, operating hours, tags, and contact links.",
			Category:    "discovery",
			Icon:        "MapPin",
		},
		{
			ID:          "feat_br_badge",
			Name:        "BR Verification Badge",
			Description: "Official business registration trust checkmark verified against government records.",
			Category:    "trust",
			Icon:        "ShieldCheck",
		},
		{
			ID:          "feat_active_offers_1",
			Name:        "1 Active Always-On Deal / Offer",
			Description: "Publish 1 deal or discount visible to all nearby travellers.",
			Category:    "marketing",
			Icon:        "Tag",
		},
		{
			ID:          "feat_active_offers_5",
			Name:        "Up to 5 Active Offers",
			Description: "Run 5 concurrent promotions (happy hours, combo lunches, sunset specials).",
			Category:    "marketing",
			Icon:        "Layers",
		},
		{
			ID:          "feat_active_offers_unlimited",
			Name:        "Unlimited Active Offers",
			Description: "Uncapped simultaneous deals, seasonal tasting menus, and event specials.",
			Category:    "marketing",
			Icon:        "Infinity",
		},
		{
			ID:          "feat_qr_scanner",
			Name:        "Single-Use QR Scanner",
			Description: "Anti-fraud scanner tool for verifying and redeeming traveller voucher claims.",
			Category:    "operations",
			Icon:        "QrCode",
		},
		{
			ID:          "feat_staff_seat_1",
			Name:        "1 Staff Scanner Seat",
			Description: "Single staff login credential for counter redemption scans.",
			Category:    "operations",
			Icon:        "User",
		},
		{
			ID:          "feat_stamp_cards",
			Name:        "Digital Stamp Cards",
			Description: "Automated loyalty stamps on traveller smartphones driving repeat visits.",
			Category:    "loyalty",
			Icon:        "Award",
		},
		{
			ID:          "feat_flash_sales_2",
			Name:        "2 Geofenced 3km Flash Sales / mo",
			Description: "Broadcast high-urgency notifications to active travellers within 3km of your venue.",
			Category:    "marketing",
			Icon:        "Zap",
		},
		{
			ID:          "feat_flash_sales_unlimited",
			Name:        "Unlimited Geofenced Flash Sales",
			Description: "Real-time flash sales push whenever you have open tables or empty surf slots.",
			Category:    "marketing",
			Icon:        "Radio",
		},
		{
			ID:          "feat_reservations",
			Name:        "Table & Session Reservations",
			Description: "Accept direct bookings for tables, surf sessions, or tours with confirmation SMS.",
			Category:    "operations",
			Icon:        "Calendar",
		},
		{
			ID:          "feat_promo_reel",
			Name:        "Behind-the-Scenes Promo Reel",
			Description: "Featured short video in travellers' daily inspiration feed highlighting your venue.",
			Category:    "marketing",
			Icon:        "Film",
		},
		{
			ID:          "feat_team_seats_3",
			Name:        "3 Team Seats (1 Manager + 2 Scanners)",
			Description: "Multi-staff access with separate permissions for shift leads and cashiers.",
			Category:    "operations",
			Icon:        "Users",
		},
		{
			ID:          "feat_priority_ranking",
			Name:        "Priority Category Ranking",
			Description: "Promoted placement near the top of category lists (e.g. #1 cafe in Mirissa).",
			Category:    "discovery",
			Icon:        "TrendingUp",
		},
		{
			ID:          "feat_quest_sponsorship",
			Name:        "Town Daily Quest Sponsorship Priority",
			Description: "Featured as a primary checkpoint on travellers' gamified island quest trails.",
			Category:    "marketing",
			Icon:        "Compass",
		},
		{
			ID:          "feat_vip_top_spot",
			Name:        "VIP Top-Spot Map Pin",
			Description: "Prominent golden map marker with pulsing halo that stands out at all zoom levels.",
			Category:    "discovery",
			Icon:        "Crown",
		},
		{
			ID:          "feat_geo_heatmap",
			Name:        "Geo Demand Heatmap Analytics",
			Description: "Live traveller footfall density, competitor trends, and seasonal tourist influx maps.",
			Category:    "insights",
			Icon:        "BarChart3",
		},
		{
			ID:          "feat_team_seats_unlimited",
			Name:        "Up to 5 Managers + Unlimited Staff",
			Description: "Full enterprise governance, multi-shift scanner accounts, and audit log tracking.",
			Category:    "operations",
			Icon:        "ShieldAlert",
		},
	}
}

// DefaultPackagesConfig generates the default 3 vendor packages ($3.99, $11.99, $22.99)
// and initial feature allocations.
func DefaultPackagesConfig() PackagesConfig {
	features := DefaultMasterFeatures()
	fMap := make(map[string]PackageFeature, len(features))
	for _, f := range features {
		fMap[f.ID] = f
	}

	starterFeatures := []PackageFeature{
		fMap["feat_map_pin"],
		fMap["feat_br_badge"],
		fMap["feat_active_offers_1"],
		fMap["feat_qr_scanner"],
		fMap["feat_staff_seat_1"],
	}

	growthFeatures := []PackageFeature{
		fMap["feat_map_pin"],
		fMap["feat_br_badge"],
		fMap["feat_active_offers_5"],
		fMap["feat_qr_scanner"],
		fMap["feat_stamp_cards"],
		fMap["feat_flash_sales_2"],
		fMap["feat_reservations"],
		fMap["feat_promo_reel"],
		fMap["feat_team_seats_3"],
		fMap["feat_priority_ranking"],
	}

	proFeatures := []PackageFeature{
		fMap["feat_map_pin"],
		fMap["feat_br_badge"],
		fMap["feat_active_offers_unlimited"],
		fMap["feat_qr_scanner"],
		fMap["feat_stamp_cards"],
		fMap["feat_flash_sales_unlimited"],
		fMap["feat_reservations"],
		fMap["feat_promo_reel"],
		fMap["feat_quest_sponsorship"],
		fMap["feat_vip_top_spot"],
		fMap["feat_geo_heatmap"],
		fMap["feat_team_seats_unlimited"],
	}

	return PackagesConfig{
		Packages: []VendorPackage{
			{
				ID:                 "pkg_starter",
				Slug:               "starter",
				Name:               "Starter",
				Tagline:            "For getting verified and visible on the traveller map",
				MonthlyPriceUSD:    3.99,
				YearlyPriceUSD:     39.90,
				Badge:              "",
				Highlighted:        false,
				ActiveOffers:       1,
				FlashSalesPerMonth: 0,
				TeamSeats:          1,
				Features:           starterFeatures,
				ExcludedFeatures: []string{
					"feat_stamp_cards",
					"feat_flash_sales_2",
					"feat_reservations",
					"feat_geo_heatmap",
					"feat_vip_top_spot",
				},
				SortOrder: 1,
			},
			{
				ID:                 "pkg_growth",
				Slug:               "growth",
				Name:               "Growth",
				Tagline:            "For busy spots driving recurring footfall and flash sales",
				MonthlyPriceUSD:    11.99,
				YearlyPriceUSD:     119.90,
				Badge:              "Most Popular",
				Highlighted:        true,
				ActiveOffers:       5,
				FlashSalesPerMonth: 2,
				TeamSeats:          3,
				Features:           growthFeatures,
				ExcludedFeatures: []string{
					"feat_flash_sales_unlimited",
					"feat_vip_top_spot",
					"feat_geo_heatmap",
					"feat_quest_sponsorship",
				},
				SortOrder: 2,
			},
			{
				ID:                 "pkg_pro",
				Slug:               "pro",
				Name:               "Pro",
				Tagline:            "For premier venues seeking regional leadership and quest fame",
				MonthlyPriceUSD:    22.99,
				YearlyPriceUSD:     229.90,
				Badge:              "Best Value",
				Highlighted:        false,
				ActiveOffers:       -1, // unlimited
				FlashSalesPerMonth: -1, // unlimited
				TeamSeats:          -1, // unlimited staff, 5 managers
				Features:           proFeatures,
				ExcludedFeatures:   []string{},
				SortOrder:          3,
			},
		},
		FeaturePool: features,
		UpdatedAt:   time.Now(),
	}
}

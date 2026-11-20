package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

const occurrenceCols = `id, service_id, start_at, end_at, capacity, status, created_at`

func scanOccurrence(row pgx.Row) (domain.ServiceOccurrence, error) {
	var o domain.ServiceOccurrence
	err := row.Scan(&o.ID, &o.ServiceID, &o.StartAt, &o.EndAt, &o.Capacity, &o.Status, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

func (s *Store) CreateOccurrence(ctx context.Context, o domain.ServiceOccurrence) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO service_occurrences (id, service_id, start_at, end_at, capacity, status)
		VALUES ($1,$2,$3,$4,$5,'scheduled')`, o.ID, o.ServiceID, o.StartAt, o.EndAt, o.Capacity)
	return err
}

func (s *Store) OccurrenceByID(ctx context.Context, id string) (domain.ServiceOccurrence, error) {
	return scanOccurrence(s.Pool.QueryRow(ctx, `SELECT `+occurrenceCols+` FROM service_occurrences WHERE id=$1`, id))
}

// OccurrencesWithRemaining lists a service's occurrences starting from
// `from` (scheduled only, unless includeCancelled), each with its
// remaining capacity filled in, soonest first.
func (s *Store) OccurrencesWithRemaining(ctx context.Context, svc domain.BookableService, from time.Time, includeCancelled bool, limit int) ([]domain.ServiceOccurrence, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT o.id, o.service_id, o.start_at, o.end_at, o.capacity, o.status, o.created_at,
			COALESCE(o.capacity, $3) - COALESCE((
				SELECT CASE WHEN $4 = 'guests' THEN sum(b.party_size) ELSE count(*) END
				FROM bookings b WHERE b.occurrence_id = o.id
				  AND (b.status='confirmed' OR (b.status='awaiting_payment' AND b.hold_expires_at > now()))), 0)
		FROM service_occurrences o
		WHERE o.service_id=$1 AND o.start_at >= $2 AND ($5 OR o.status='scheduled')
		ORDER BY o.start_at
		LIMIT $6`, svc.ID, from, svc.Capacity, string(capacityUnit(svc.CapacityUnit)), includeCancelled, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ServiceOccurrence{}
	for rows.Next() {
		var o domain.ServiceOccurrence
		var remaining int
		if err := rows.Scan(&o.ID, &o.ServiceID, &o.StartAt, &o.EndAt, &o.Capacity, &o.Status, &o.CreatedAt, &remaining); err != nil {
			return nil, err
		}
		if remaining < 0 {
			remaining = 0
		}
		o.Remaining = &remaining
		out = append(out, o)
	}
	return out, rows.Err()
}

// UpdateOccurrence changes a scheduled occurrence's times or capacity.
// Times can only change while nobody has booked it, so no guest is moved
// without being asked.
func (s *Store) UpdateOccurrence(ctx context.Context, o domain.ServiceOccurrence) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE service_occurrences SET start_at=$2, end_at=$3, capacity=$4
		WHERE id=$1 AND status='scheduled'
		  AND (start_at=$2 AND end_at=$3 OR NOT EXISTS (
			SELECT 1 FROM bookings WHERE occurrence_id=$1 AND status IN ('confirmed','awaiting_payment')))`,
		o.ID, o.StartAt, o.EndAt, o.Capacity)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrOccurrenceLocked
	}
	return nil
}

// ErrOccurrenceLocked is returned when an occurrence can't be changed:
// it's cancelled, or it has bookings and the change would move them.
var ErrOccurrenceLocked = errors.New("occurrence can't be changed")

// CancelOccurrence marks an occurrence cancelled so it takes no more
// bookings, and returns the bookings on it that still need cancelling
// (and refunding) by the caller.
func (s *Store) CancelOccurrence(ctx context.Context, id string) ([]domain.Booking, error) {
	tag, err := s.Pool.Exec(ctx, `UPDATE service_occurrences SET status='cancelled' WHERE id=$1 AND status='scheduled'`, id)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.OccurrenceByID(ctx, id); err != nil {
			return nil, err
		}
	}
	return s.bookingList(ctx, `SELECT `+bookingCols+bookingFrom+`
		WHERE b.occurrence_id=$1 AND b.status IN ('confirmed','awaiting_payment')`, id)
}

// ExperienceListing is one bookable experience or event, with what a
// traveller needs to pick it from a list.
type ExperienceListing struct {
	Service        domain.BookableService    `json:"service"`
	VendorID       string                    `json:"vendorId"`
	VendorName     string                    `json:"vendorName"`
	VendorKind     string                    `json:"vendorKind"`
	Category       string                    `json:"category"`
	PlaceID        string                    `json:"placeId"`
	Lat            float64                   `json:"lat"`
	Lng            float64                   `json:"lng"`
	Town           string                    `json:"town"`
	NextOccurrence *domain.ServiceOccurrence `json:"nextOccurrence,omitempty"`
	DistanceKm     *float64                  `json:"distanceKm,omitempty"`
}

// ExperienceFilter narrows ExperienceListings; zero values mean "any".
type ExperienceFilter struct {
	Category string
	Types    []domain.BookingType
	Limit    int
}

// ExperienceListings returns active experience and event services of
// approved vendors that have a map place.
func (s *Store) ExperienceListings(ctx context.Context, f ExperienceFilter) ([]ExperienceListing, error) {
	types := []string{string(domain.BookingTypeExperience), string(domain.BookingTypeEvent)}
	if len(f.Types) > 0 {
		types = types[:0]
		for _, t := range f.Types {
			types = append(types, string(t))
		}
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	cols := ""
	for _, c := range []string{"id", "vendor_id", "type", "name", "description", "duration_mins", "capacity", "capacity_unit",
		"party_size_min", "party_size_max", "guests_per_room", "price_label", "availability", "active", "archived_at",
		"created_at", "updated_at", "price_minor", "currency", "pricing_mode", "free_cancel_hours", "late_cancel_refund_pct",
		"scheduling", "blackout_dates", "meeting_point"} {
		cols += "sv." + c + ", "
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT `+cols+` v.id, v.name, v.kind, v.category, p.id, v.lat, v.lng, v.town
		FROM bookable_services sv
		JOIN vendors v ON v.id = sv.vendor_id AND v.status = 'approved'
		JOIN places p ON p.vendor_id = v.id
		WHERE sv.active AND sv.archived_at IS NULL AND sv.type = ANY($1)
		  AND ($2 = '' OR v.category = $2)
		  AND (sv.scheduling = 'weekly' OR EXISTS (
			SELECT 1 FROM service_occurrences o WHERE o.service_id = sv.id AND o.status = 'scheduled' AND o.start_at > now()))
		ORDER BY sv.updated_at DESC
		LIMIT $3`, types, f.Category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ExperienceListing{}
	for rows.Next() {
		var l ExperienceListing
		var availability, meetingPoint []byte
		svc := &l.Service
		if err := rows.Scan(&svc.ID, &svc.VendorID, &svc.Type, &svc.Name, &svc.Description,
			&svc.DurationMins, &svc.Capacity, &svc.CapacityUnit, &svc.PartySizeMin, &svc.PartySizeMax, &svc.GuestsPerRoom,
			&svc.PriceLabel, &availability, &svc.Active, &svc.ArchivedAt, &svc.CreatedAt, &svc.UpdatedAt,
			&svc.PriceMinor, &svc.Currency, &svc.PricingMode, &svc.FreeCancelHours, &svc.LateCancelRefundPct,
			&svc.Scheduling, &svc.BlackoutDates, &meetingPoint,
			&l.VendorID, &l.VendorName, &l.VendorKind, &l.Category, &l.PlaceID, &l.Lat, &l.Lng, &l.Town); err != nil {
			return nil, err
		}
		if len(availability) > 0 {
			var doc domain.ServiceAvailabilityDoc
			if json.Unmarshal(availability, &doc) == nil {
				svc.Availability = &doc
			}
		}
		if len(meetingPoint) > 0 {
			var mp domain.MeetingPoint
			if json.Unmarshal(meetingPoint, &mp) == nil {
				svc.MeetingPoint = &mp
			}
		}
		if svc.BlackoutDates == nil {
			svc.BlackoutDates = []string{}
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Service.Scheduling != domain.SchedulingOccurrences {
			continue
		}
		next, err := s.OccurrencesWithRemaining(ctx, out[i].Service, time.Now(), false, 1)
		if err != nil {
			return nil, err
		}
		if len(next) > 0 {
			out[i].NextOccurrence = &next[0]
		}
	}
	return out, nil
}

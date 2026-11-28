package domain

import (
	"math/big"
	"time"
)

// FxRate is how many LKR one unit of another currency costs, as set by
// the finance team. Rates are kept in millionths (micros) of an LKR so
// they can be exact integers: 302.5 LKR per USD is 302_500_000.
type FxRate struct {
	Currency         string    `json:"currency"`
	LKRPerUnitMicros int64     `json:"lkrPerUnitMicros"`
	SetBy            string    `json:"setBy"`
	SetAt            time.Time `json:"setAt"`
}

// FxRateMaxAge is how old a rate may be before checkouts in that
// currency are refused.
const FxRateMaxAge = 7 * 24 * time.Hour

// Stale reports whether the rate is too old to charge with at now.
func (r FxRate) Stale(now time.Time) bool { return now.Sub(r.SetAt) > FxRateMaxAge }

// TwoDecimalCurrency reports whether c uses two decimal places, like LKR,
// so minor units convert with the rate alone. Only these are accepted as
// a gateway's charge currency.
func TwoDecimalCurrency(c string) bool {
	switch c {
	case "LKR", "USD", "EUR", "GBP", "AUD", "CAD", "SGD", "NZD", "CHF", "AED", "INR":
		return true
	}
	return false
}

// ConvertFromLKR turns an LKR amount into minor units of the rate's
// currency, adding marginBps (basis points) on top to cover the gateway's
// fees and conversion spread, rounded half up. Both currencies must have
// two decimals.
func ConvertFromLKR(lkrMinor, lkrPerUnitMicros int64, marginBps int) int64 {
	if lkrMinor <= 0 || lkrPerUnitMicros <= 0 {
		return 0
	}
	// lkrMinor × 1e6 × (10000 + margin) / (rate × 10000)
	num := new(big.Int).Mul(big.NewInt(lkrMinor), big.NewInt(1_000_000))
	num.Mul(num, big.NewInt(int64(10_000+marginBps)))
	den := new(big.Int).Mul(big.NewInt(lkrPerUnitMicros), big.NewInt(10_000))
	return divHalfUp(num, den)
}

// ProportionalRefund is how much of a payment charged in another
// currency to give back for an LKR refund: the same share of what was
// charged. A refund of everything paid returns exactly what was charged;
// a partial one rounds down, so refunds can never add up to more than the
// charge.
func ProportionalRefund(refundLKR, paidLKR, chargedMinor int64) int64 {
	if refundLKR <= 0 || paidLKR <= 0 || chargedMinor <= 0 {
		return 0
	}
	if refundLKR >= paidLKR {
		return chargedMinor
	}
	num := new(big.Int).Mul(big.NewInt(refundLKR), big.NewInt(chargedMinor))
	return new(big.Int).Quo(num, big.NewInt(paidLKR)).Int64()
}

func divHalfUp(num, den *big.Int) int64 {
	half := new(big.Int).Quo(den, big.NewInt(2))
	return new(big.Int).Quo(new(big.Int).Add(num, half), den).Int64()
}

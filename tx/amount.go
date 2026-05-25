package tx

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Amount is a token quantity in datum (the smallest indivisible unit).
// For DAG, 1 DAG = 10^8 datum; other tokens may use different decimals.
type Amount int64

// MaxDecimals is the largest decimals value Token can multiply without
// overflowing int64.
const MaxDecimals = 18

// Token constructs an Amount from a human value at the given decimals,
// rounding to the nearest base unit. For exact construction, or for values
// where value*10^decimals exceeds 2^53, use Datum.
func Token(decimals int, value float64) Amount {
	multiplier := int64(1)
	for range decimals {
		multiplier *= 10
	}
	return Amount(math.Round(value * float64(multiplier)))
}

var parseTokenRe = regexp.MustCompile(`^\d+(\.\d+)?$`)

// ParseToken parses a decimal string into an Amount at the given precision.
// Inverse of FormatToken; returns ErrInvalidAmount on bad input.
func ParseToken(decimals int, s string) (Amount, error) {
	if !parseTokenRe.MatchString(s) {
		return 0, fmt.Errorf("parse %q: %w", s, ErrInvalidAmount)
	}

	intPart, fracPart, _ := strings.Cut(s, ".")
	if len(fracPart) > decimals {
		return 0, fmt.Errorf("parse %q: %w", s, ErrInvalidAmount)
	}
	fracPart += strings.Repeat("0", decimals-len(fracPart))

	n, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", s, ErrInvalidAmount)
	}
	return Amount(n), nil
}

// Datum constructs an Amount from raw base units. Negative values are
// accepted; use Validate if non-negativity matters.
func Datum(n int64) Amount {
	return Amount(n)
}

// Validate returns ErrInvalidAmount if a is negative.
func (a Amount) Validate() error {
	if a < 0 {
		return ErrInvalidAmount
	}
	return nil
}

// Int64 returns the raw datum value.
func (a Amount) Int64() int64 {
	return int64(a)
}

// String returns the raw datum value as a decimal string. Use FormatToken
// for human-readable output at known decimal precision.
func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

// FormatToken returns the Amount as a decimal string at the given precision,
// e.g. Amount(100000).FormatToken(8) returns "0.00100000".
func (a Amount) FormatToken(decimals int) string {
	if decimals == 0 {
		return strconv.FormatInt(int64(a), 10)
	}
	divisor := int64(1)
	for range decimals {
		divisor *= 10
	}
	n := int64(a)
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return fmt.Sprintf("%s%d.%0*d", sign, n/divisor, decimals, n%divisor)
}

// FormatTokenCompact returns the Amount as a decimal string at the given
// precision with trailing zeros and trailing dot trimmed, e.g.
// Amount(100000).FormatTokenCompact(8) returns "0.001".
func (a Amount) FormatTokenCompact(decimals int) string {
	s := a.FormatToken(decimals)
	if decimals == 0 {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

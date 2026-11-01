package mtl

import (
	"fmt"
	"github.com/shopspring/decimal"
	"regexp"
)

// Decimal is an exact base-ten financial value. Optional values use *Decimal.
// Percentage rates use percentage units: 0.04 denotes 0.04%, not 4%.
// Addition and multiplication are exact; journal division uses 18 fractional
// places, rounding half away from zero. Decimal.String returns canonical digits.
type Decimal = decimal.Decimal

func ParseDecimal(value string) (Decimal, error) {
	if !regexp.MustCompile(`^[+-]?[0-9]+(\.[0-9]+)?$`).MatchString(value) {
		return Decimal{}, fmt.Errorf("%w: decimal must contain plain base-ten digits", ErrInvalidRequest)
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Decimal{}, fmt.Errorf("%w: decimal: %v", ErrInvalidRequest, err)
	}
	return d, nil
}
func MustDecimal(value string) Decimal {
	d, err := ParseDecimal(value)
	if err != nil {
		panic(err)
	}
	return d
}

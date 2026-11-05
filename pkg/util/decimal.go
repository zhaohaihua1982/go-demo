package util

import (
	"strings"

	"github.com/shopspring/decimal"
)

func sumByString(a, b string, defaultValue decimal.Decimal) decimal.Decimal {
	// 去空格
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	from, err := decimal.NewFromString(a)
	if err != nil {
		from = defaultValue
	}
	to, err := decimal.NewFromString(b)
	if err != nil {
		to = defaultValue
	}
	sum := from.Add(to)
	return sum
}

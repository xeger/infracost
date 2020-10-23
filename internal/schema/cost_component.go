package schema

import (
	"github.com/shopspring/decimal"
)

type CostComponent struct {
	Name                 string
	Unit                 string
	IgnoreIfMissingPrice bool
	ProductFilter        *ProductFilter
	PriceFilter          *PriceFilter
	HourlyQuantity       *decimal.Decimal
	MonthlyQuantity      *decimal.Decimal
	BucketQuantity       *decimal.Decimal
	price                decimal.Decimal
	priceHash            string
	HourlyCost           *decimal.Decimal
	MonthlyCost          *decimal.Decimal
	BucketCost           *decimal.Decimal
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func (c *CostComponent) CalculateCosts() {
	c.fillQuantities()
	if c.HourlyCost == nil && c.MonthlyCost == nil {
		c.HourlyCost = decimalPtr(c.price.Mul(*c.HourlyQuantity))
		c.MonthlyCost = decimalPtr(c.price.Mul(*c.MonthlyQuantity))
	} else if c.HourlyCost == nil {
		c.HourlyCost = decimalPtr(c.MonthlyCost.Div(hourToMonthMultiplier))
	} else if c.MonthlyCost == nil {
		c.MonthlyCost = decimalPtr(c.HourlyCost.Mul(hourToMonthMultiplier))
	}
}

func (c *CostComponent) fillQuantities() {
	if c.HourlyQuantity == nil && c.MonthlyQuantity == nil {
		c.HourlyQuantity = decimalPtr(decimal.Zero)
		c.MonthlyQuantity = decimalPtr(decimal.Zero)
	} else if c.HourlyQuantity == nil {
		c.HourlyQuantity = decimalPtr(c.MonthlyQuantity.Div(hourToMonthMultiplier))
	} else if c.MonthlyQuantity == nil {
		c.MonthlyQuantity = decimalPtr(c.HourlyQuantity.Mul(hourToMonthMultiplier))
	}
}

func (c *CostComponent) SetPrice(price decimal.Decimal) {
	c.price = price
}

func (c *CostComponent) Price() decimal.Decimal {
	return c.price
}

func (c *CostComponent) SetPriceHash(priceHash string) {
	c.priceHash = priceHash
}

func (c *CostComponent) PriceHash() string {
	return c.priceHash
}

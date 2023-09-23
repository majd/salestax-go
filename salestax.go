package salestax

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type (
	TaxExchange string
	TaxArea     string
)

const (
	TaxExchangeBusiness TaxExchange = "business"
	TaxExchangeConsumer TaxExchange = "consumer"

	TaxAreaWorldwide TaxArea = "worldwide"
	TaxAreaNational  TaxArea = "national"
	TaxAreaRegional  TaxArea = "regional"
)

// SalesTax is the sales tax object.
type SalesTax struct {
	// Type is the type of the sales tax.
	Type string
	// Rate is the sales tax rate.
	Rate float32
	// Area is the area of the sales tax.
	Area TaxArea
	// Exchange is the tax exchange type of the sales tax.
	Exchange TaxExchange
	// Charge contains information about the charge types of the sales tax.
	Charge TaxCharge
}

// TaxCharge contains information about the charge types of the sales tax.
type TaxCharge struct {
	// Direct implies that direct-charge rule should be in effect.
	Direct bool
	// Reverse implies that reverse-charge rule should be in effect.
	Reverse bool
}

// Ctrl is the salestax controller.
type Ctrl struct {
	// OriginCountryCode is the country code of the tax registration and liability.
	OriginCountryCode *string
	// RegionalTaxEnabled specifies whether regional taxation is enabled, such as in the EU region.
	// If this value is set to true (VAT OSS threshold is not exceeded), the rate of the origin country will be used.
	RegionalTaxEnabled bool

	regionCountries map[string][]string
	taxRates        map[string]taxRate
}

// GetSalesTax returns the sales tax for the desired country.
// The parameters stateCode and taxNumber are optional.
func (t *Ctrl) GetSalesTax(countryCode string, stateCode *string, taxNumber *string) (*SalesTax, error) {
	countryCode = strings.ToUpper(countryCode)
	if stateCode != nil {
		stateCode = Ptr(strings.ToUpper(*stateCode))
	}
	targetArea, err := t.getTargetArea(countryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get target area: %w", err)
	}

	var countryTax, stateTax *taxRate

	if *targetArea == TaxAreaRegional && !t.RegionalTaxEnabled && t.OriginCountryCode != nil {
		countryTax, err = t.getSalesTaxRate(*t.OriginCountryCode)
		if err != nil {
			return nil, fmt.Errorf("failed to get country tax rate for %s: %w", *t.OriginCountryCode, err)
		}

		stateTax = Ptr(defaultTaxRate)
	} else {
		countryTax, err = t.getSalesTaxRate(countryCode)
		if err != nil {
			return nil, fmt.Errorf("failed to get country tax rate for %s: %w", countryCode, err)
		}

		if countryTax.States != nil && stateCode != nil {
			tax, ok := countryTax.States[*stateCode]
			if !ok {
				stateTax = Ptr(defaultTaxRate)
			}

			stateTax = &tax
		} else {
			stateTax = Ptr(defaultTaxRate)
		}
	}

	taxExchange := TaxExchangeConsumer
	isExempt := false
	totalRate := countryTax.TaxRate + stateTax.TaxRate

	if countryTax.TaxRate > 0 || stateTax.TaxRate > 0 {
		exchangeStatus, exemptStatus, err := t.getTaxExchangeStatus(countryCode, stateCode, taxNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get tax exchange status: %w", err)
		}

		taxExchange = *exchangeStatus
		isExempt = exemptStatus
	}

	taxType := countryTax.TaxType
	if stateTax.TaxRate > 0 {
		if countryTax.TaxRate > 0 {
			taxType = fmt.Sprintf("%s+%s", taxType, stateTax.TaxType)
		} else {
			taxType = stateTax.TaxType
		}
	}

	taxCharge := TaxCharge{}
	if taxType != "none" {
		taxCharge.Direct = !isExempt
		taxCharge.Reverse = isExempt && totalRate > 0
	}

	taxRate := totalRate
	if isExempt {
		totalRate = 0
	}

	return &SalesTax{
		Type:     taxType,
		Rate:     taxRate,
		Area:     *targetArea,
		Exchange: taxExchange,
		Charge:   taxCharge,
	}, nil
}

type taxRate struct {
	TaxType            string             `json:"type"`
	TaxRate            float32            `json:"rate"`
	PreviousRecordings map[string]taxRate `json:"before,omitempty"`
	States             map[string]taxRate `json:"states,omitempty"`
}

var (
	//go:embed res/region_countries.json
	regionCountriesData []byte
	//go:embed res/sales_tax_rates.json
	salesTaxRatesData []byte
	currentTime       = time.Now
	defaultTaxRate    = taxRate{"none", 0, nil, nil}
)

func (t *Ctrl) getTaxExchangeStatus(countryCode string, stateCode *string, taxNumber *string) (status *TaxExchange, exempt bool, err error) {
	targetArea, err := t.getTargetArea(countryCode)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get target area: %w", err)
	}

	hasTotalSalesTax, err := t.hasTotalSalesTax(countryCode, stateCode)
	if err != nil {
		return nil, false, fmt.Errorf("failed to determine whether a sales tax is applicable: %w", err)
	}

	if hasTotalSalesTax {
		if taxNumber != nil {
			return Ptr(TaxExchangeBusiness), *targetArea != TaxAreaNational, nil
		}

		return Ptr(TaxExchangeConsumer), false, nil
	}

	return Ptr(TaxExchangeConsumer), true, nil
}

func (t *Ctrl) hasTotalSalesTax(countryCode string, stateCode *string) (bool, error) {
	countryCode = strings.ToUpper(countryCode)
	if stateCode != nil {
		stateCode = Ptr(strings.ToUpper(*stateCode))
	}

	rate, err := t.getSalesTaxRate(countryCode)
	if err != nil {
		return false, fmt.Errorf("failed to get country tax rate for %s: %w", countryCode, err)
	}

	totalTax := rate.TaxRate

	if stateCode != nil {
		rate, ok := rate.States[*stateCode]
		if ok {
			totalTax += rate.TaxRate
		}
	}

	return totalTax > 0, nil
}

func (t *Ctrl) getSalesTaxRate(countryCode string) (*taxRate, error) {
	rates, err := t.getSalesTaxRates()
	if err != nil {
		return nil, fmt.Errorf("failed to get sales tax rates: %w", err)
	}

	rate, ok := rates[countryCode]
	if !ok {
		return Ptr(defaultTaxRate), nil
	}

	if rate.PreviousRecordings != nil {
		var activeDateKey *string
		var activeDate *time.Time

		currentDate := currentTime()
		dateLayout := "2006-01-02T15:04:05.000Z"

		for dateStr := range rate.PreviousRecordings {
			dateStr := dateStr
			date, err := time.Parse(dateLayout, dateStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
			}

			if currentDate.Before(date) {
				if activeDate == nil || date.Before(*activeDate) {
					activeDate = &date
					activeDateKey = &dateStr
				}
			}
		}

		if activeDateKey != nil {
			return Ptr(rate.PreviousRecordings[*activeDateKey]), nil
		}
	}

	return &rate, nil
}

func (t *Ctrl) getSalesTaxRates() (map[string]taxRate, error) {
	if t.taxRates == nil {
		err := json.Unmarshal(salesTaxRatesData, &t.taxRates)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return t.taxRates, nil
}

func (t *Ctrl) getTargetArea(countryCode string) (*TaxArea, error) {
	var targetArea TaxArea
	targetArea = TaxAreaWorldwide

	if t.OriginCountryCode != nil {
		if *t.OriginCountryCode == countryCode {
			targetArea = TaxAreaNational
		} else {
			regionCountries, err := t.getRegionCountries()
			if err != nil {
				return nil, fmt.Errorf("failed to get region countries: %w", err)
			}

			for _, countries := range regionCountries {
				if slices.Contains(countries, *t.OriginCountryCode) && slices.Contains(countries, countryCode) {
					targetArea = TaxAreaRegional
					break
				}
			}
		}
	}

	return &targetArea, nil
}

func (t *Ctrl) getRegionCountries() (map[string][]string, error) {
	if t.regionCountries == nil {
		err := json.Unmarshal(regionCountriesData, &t.regionCountries)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return t.regionCountries, nil
}

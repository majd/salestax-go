package salestax

import (
	"strconv"
	"testing"
	"time"
)

func Test_GetSalesTax(t *testing.T) {
	testCases := []struct {
		originCountryCode  *string
		regionalTaxEnabled bool
		countryCode        string
		stateCode          *string
		taxNumber          *string
		expectedResult     SalesTax
	}{
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "DE",
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.19,
				Area:     TaxAreaNational,
				Exchange: TaxExchangeConsumer,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "DE",
			taxNumber:          Ptr("DE000000000"),
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.19,
				Area:     TaxAreaNational,
				Exchange: TaxExchangeBusiness,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: false,
			countryCode:        "FR",
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.19,
				Area:     TaxAreaRegional,
				Exchange: TaxExchangeConsumer,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "FR",
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.2,
				Area:     TaxAreaRegional,
				Exchange: TaxExchangeConsumer,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "FR",
			taxNumber:          Ptr("FR000000000"),
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.2,
				Area:     TaxAreaRegional,
				Exchange: TaxExchangeBusiness,
				Charge: TaxCharge{
					Direct:  false,
					Reverse: true,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "CA",
			stateCode:          Ptr("QC"),
			expectedResult: SalesTax{
				Type:     "gst+qst",
				Rate:     0.14975,
				Area:     TaxAreaWorldwide,
				Exchange: TaxExchangeConsumer,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "US",
			stateCode:          Ptr("NY"),
			taxNumber:          Ptr("0123456789"),
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.04,
				Area:     TaxAreaWorldwide,
				Exchange: TaxExchangeBusiness,
				Charge: TaxCharge{
					Direct:  false,
					Reverse: true,
				},
			},
		},
		{
			originCountryCode:  Ptr("DE"),
			regionalTaxEnabled: true,
			countryCode:        "US",
			stateCode:          Ptr("NY"),
			expectedResult: SalesTax{
				Type:     "vat",
				Rate:     0.04,
				Area:     TaxAreaWorldwide,
				Exchange: TaxExchangeConsumer,
				Charge: TaxCharge{
					Direct:  true,
					Reverse: false,
				},
			},
		},
	}

	for _, tc := range testCases {
		ctrl := &Ctrl{
			OriginCountryCode:  tc.originCountryCode,
			RegionalTaxEnabled: tc.regionalTaxEnabled,
		}

		salesTax, err := ctrl.GetSalesTax(tc.countryCode, tc.stateCode, tc.taxNumber)
		if err != nil {
			t.Errorf("got error: %s", err.Error())
			return
		}
		if salesTax.Type != tc.expectedResult.Type {
			t.Errorf("expected %s; got %s", tc.expectedResult.Type, salesTax.Type)
			return
		}
		if salesTax.Rate != tc.expectedResult.Rate {
			t.Errorf("expected %f; got %f", tc.expectedResult.Rate, salesTax.Rate)
			return
		}
		if salesTax.Area != tc.expectedResult.Area {
			t.Errorf("expected %s; got %s", tc.expectedResult.Area, salesTax.Area)
			return
		}
		if salesTax.Exchange != tc.expectedResult.Exchange {
			t.Errorf("expected %s; got %s", tc.expectedResult.Exchange, salesTax.Exchange)
			return
		}
		if salesTax.Charge != tc.expectedResult.Charge {
			t.Errorf("expected %+v; got %+v", tc.expectedResult.Charge, salesTax.Charge)
			return
		}
	}
}

func Test_getTaxExchangeStatus(t *testing.T) {
	testCases := []struct {
		countryCode            string
		stateCode              *string
		taxNumber              *string
		expectedExchangeStatus TaxExchange
		expectedExemptStatus   bool
	}{
		{
			countryCode:            "DE",
			expectedExchangeStatus: TaxExchangeConsumer,
			expectedExemptStatus:   false,
		},
		{
			countryCode:            "DE",
			taxNumber:              Ptr("DE000000000"),
			expectedExchangeStatus: TaxExchangeBusiness,
			expectedExemptStatus:   false,
		},
		{
			countryCode:            "FR",
			taxNumber:              Ptr("FR000000000"),
			expectedExchangeStatus: TaxExchangeBusiness,
			expectedExemptStatus:   true,
		},
		{
			countryCode:            "FR",
			expectedExchangeStatus: TaxExchangeConsumer,
			expectedExemptStatus:   false,
		},
		{
			countryCode:            "??",
			expectedExchangeStatus: TaxExchangeConsumer,
			expectedExemptStatus:   true,
		},
	}

	ctrl := &Ctrl{
		OriginCountryCode: Ptr("DE"),
	}

	for _, tc := range testCases {
		exchangeStatus, exemptStatus, err := ctrl.getTaxExchangeStatus(tc.countryCode, tc.stateCode, tc.taxNumber)
		if err != nil {
			t.Errorf("got error: %s", err.Error())
			return
		}
		if *exchangeStatus != tc.expectedExchangeStatus {
			t.Errorf("expected %s; got %s", tc.expectedExchangeStatus, *exchangeStatus)
			return
		}
		if exemptStatus != tc.expectedExemptStatus {
			t.Errorf("expected %s; got %s", strconv.FormatBool(tc.expectedExemptStatus), strconv.FormatBool(exemptStatus))
			return
		}
	}
}

func Test_hasTotalSalesTax(t *testing.T) {
	testCases := []struct {
		countryCode    string
		stateCode      *string
		expectedResult bool
	}{
		{
			countryCode:    "DE",
			stateCode:      nil,
			expectedResult: true,
		},
		{
			countryCode:    "US",
			stateCode:      Ptr("NY"),
			expectedResult: true,
		},
		{
			countryCode:    "??",
			stateCode:      nil,
			expectedResult: false,
		},
	}

	ctrl := &Ctrl{}

	for _, tc := range testCases {
		res, err := ctrl.hasTotalSalesTax(tc.countryCode, tc.stateCode)
		if err != nil {
			t.Errorf("got error: %s", err.Error())
			return
		}
		if res != tc.expectedResult {
			t.Errorf("expected %s; got %s", strconv.FormatBool(tc.expectedResult), strconv.FormatBool(res))
			return
		}
	}
}

func Test_getSalesTaxRate(t *testing.T) {
	testCases := []struct {
		countryCode string
		rate        float32
		time        *time.Time
	}{
		{
			countryCode: "DE",
			rate:        0.19,
		},
		{
			countryCode: "DE",
			rate:        0.16,
			time:        Ptr(time.Unix(1597914021, 0)),
		},
		{
			countryCode: "FR",
			rate:        0.2,
		},
	}

	ctrl := &Ctrl{}

	for _, tc := range testCases {
		if tc.time != nil {
			currentTime = func() time.Time {
				return *tc.time
			}
		}

		rate, err := ctrl.getSalesTaxRate(tc.countryCode)
		if err != nil {
			t.Errorf("got error: %s", err.Error())
			return
		}
		if rate.TaxRate != tc.rate {
			t.Errorf("expected %f; got %f", tc.rate, rate.TaxRate)
			return
		}

		currentTime = time.Now
	}
}

func Test_getSalesTaxRates(t *testing.T) {
	ctrl := &Ctrl{
		OriginCountryCode: Ptr("DE"),
	}

	rates, err := ctrl.getSalesTaxRates()
	if err != nil {
		t.Errorf("got error: %s", err.Error())
		return
	}

	rate, ok := rates["DE"]
	if !ok {
		t.Errorf("expected %s; got %s", strconv.FormatBool(true), strconv.FormatBool(false))
		return
	}

	if rate.TaxRate != 0.19 {
		t.Errorf("expected %f; got %f", 0.19, rate.TaxRate)
		return
	}

	if rate.TaxType != "vat" {
		t.Errorf("expected %s; got %s", "vat", rate.TaxType)
		return
	}

	if len(rate.PreviousRecordings) != 2 {
		t.Errorf("expected %d; got %d", 2, len(rate.PreviousRecordings))
		return
	}
}

func Test_getTargetArea(t *testing.T) {
	ctrl := &Ctrl{
		OriginCountryCode: Ptr("DE"),
	}

	testCases := []struct {
		countryCode string
		expected    TaxArea
	}{
		{
			countryCode: "DE",
			expected:    TaxAreaNational,
		},
		{
			countryCode: "FR",
			expected:    TaxAreaRegional,
		},
		{
			countryCode: "US",
			expected:    TaxAreaWorldwide,
		},
	}

	for _, tc := range testCases {
		targetArea, err := ctrl.getTargetArea(tc.countryCode)
		if err != nil {
			t.Errorf("got error: %s", err.Error())
			return
		}
		if *targetArea != tc.expected {
			t.Errorf("expected %s; got %s", tc.expected, *targetArea)
			return
		}
	}
}

func Test_getRegionCountries(t *testing.T) {
	ctrl := &Ctrl{}

	res, err := ctrl.getRegionCountries()
	if err != nil {
		t.Errorf("got error: %s", err.Error())
		return
	}

	if len(res["EU"]) != 29 {
		t.Errorf("expected %d; got %d", 29, len(res["EU"]))
		return
	}
}
